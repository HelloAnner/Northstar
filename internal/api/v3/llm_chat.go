/**
 * AI 对话流式接口
 *
 * @author Anner
 * Created on 2026/3/14
 */
package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"northstar/internal/dagcalc"
	"northstar/internal/llm"
)

// LLMChatClient 是 AI 对话依赖的最小模型接口。
type LLMChatClient interface {
	Chat(ctx context.Context, req llm.ChatRequest, stream func(string) error) (llm.ChatResult, error)
}

type llmChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmChatRequest struct {
	SessionID string           `json:"sessionId"`
	Mode      string           `json:"mode"`
	Reasoning bool             `json:"reasoning"`
	Messages  []llmChatMessage `json:"messages"`
}

type llmResultPayload struct {
	Mode             string                   `json:"mode"`
	Reply            string                   `json:"reply"`
	Groups           []dagcalc.IndicatorGroup `json:"groups,omitempty"`
	AppliedRules     []dagcalc.AppliedRule    `json:"appliedRules,omitempty"`
	RuleAdded        *ruleAddedPayload        `json:"ruleAdded,omitempty"`
	// AdjustedTargets 直接调整的指标 ID 列表（不含规则触发的联动变更）
	AdjustedTargets  []string                 `json:"adjustedTargets,omitempty"`
}

type ruleAddedPayload struct {
	Text   string `json:"text"`
	Status string `json:"status"`
}

type llmStreamEvent struct {
	Type    string            `json:"type"`
	Content string            `json:"content,omitempty"`
	Result  *llmResultPayload `json:"result,omitempty"`
	Error   string            `json:"error,omitempty"`
}

type llmStreamWriter struct {
	writer  gin.ResponseWriter
	flusher http.Flusher
}

type llmChatContext struct {
	ctx        context.Context
	request    llmChatRequest
	year       int
	month      int
	userPrompt string
	reasoning  bool
	groups     []dagcalc.IndicatorGroup
}

// StreamLLMChat AI 对话接口（SSE）
// POST /api/llm/chat/stream
func (h *Handler) StreamLLMChat(c *gin.Context) {
	req, err := decodeLLMChatRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	log.Printf("llm chat start: session=%s mode=%s messages=%d", req.SessionID, req.Mode, len(req.Messages))
	chatCtx, err := h.buildLLMChatContext(c.Request.Context(), req)
	if err != nil {
		log.Printf("llm chat context error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stream := newLLMStreamWriter(c)
	result, err := h.runLLMChat(stream, chatCtx)
	if err != nil {
		log.Printf("llm chat run error: %v", err)
		stream.SendError(err)
		return
	}
	if result != nil {
		if err := stream.Send(llmStreamEvent{Type: "result", Result: result}); err != nil {
			log.Printf("llm chat result send error: %v", err)
			return
		}
	}
	_ = stream.Send(llmStreamEvent{Type: "final"})
}

func decodeLLMChatRequest(c *gin.Context) (llmChatRequest, error) {
	var req llmChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return req, err
	}
	return req, nil
}

func (h *Handler) buildLLMChatContext(ctx context.Context, req llmChatRequest) (llmChatContext, error) {
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		return llmChatContext{}, fmt.Errorf("系统未初始化")
	}
	groups, err := dagcalc.CalculateIndicators(h.store, year, month)
	if err != nil {
		return llmChatContext{}, fmt.Errorf("计算指标失败")
	}
	roundIndicatorGroupsInPlace(groups)
	return llmChatContext{
		ctx:        ctx,
		request:    req,
		year:       year,
		month:      month,
		userPrompt: h.getConfigValue("llm_user_prompt", ""),
		reasoning:  req.Reasoning,
		groups:     groups,
	}, nil
}

func toLLMChatRequest(req llmChatRequest, year, month int) llm.ChatRequest {
	messages := make([]llm.ChatMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, llm.ChatMessage{Role: msg.Role, Content: msg.Content})
	}
	return llm.ChatRequest{Messages: messages, Year: year, Month: month}
}

func newLLMStreamWriter(c *gin.Context) *llmStreamWriter {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	flusher, _ := c.Writer.(http.Flusher)
	return &llmStreamWriter{writer: c.Writer, flusher: flusher}
}

func (w *llmStreamWriter) Send(event llmStreamEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := w.writer.Write([]byte("data: " + string(payload) + "\n\n")); err != nil {
		return err
	}
	if w.flusher != nil {
		w.flusher.Flush()
	}
	return nil
}

func (w *llmStreamWriter) SendDelta(content string) error {
	return w.Send(llmStreamEvent{Type: "message_delta", Content: content})
}

func (w *llmStreamWriter) SendError(err error) {
	_ = w.Send(llmStreamEvent{Type: "error", Error: err.Error()})
}

func (h *Handler) runLLMChat(stream *llmStreamWriter, chatCtx llmChatContext) (*llmResultPayload, error) {
	mode := normalizeChatMode(chatCtx.request.Mode)
	if mode == "adjust" {
		return h.runAdjustMode(stream, chatCtx)
	}
	return h.runChatMode(stream, chatCtx)
}

func (h *Handler) runChatMode(stream *llmStreamWriter, chatCtx llmChatContext) (*llmResultPayload, error) {
	_ = stream.Send(llmStreamEvent{Type: "thinking"})
	prompt := h.buildChatPrompt(chatCtx.year, chatCtx.month, chatCtx.groups, chatCtx.userPrompt)
	if chatCtx.reasoning {
		prompt += reasoningSuffix
	}
	client, err := h.newLLMClient(prompt)
	if err != nil {
		return nil, err
	}
	reply, err := h.streamReplyWithReasoning(stream, client, toLLMChatRequest(chatCtx.request, chatCtx.year, chatCtx.month), chatCtx.ctx, chatCtx.reasoning)
	if err != nil {
		return nil, err
	}
	return &llmResultPayload{
		Mode:  "chat",
		Reply: reply,
	}, nil
}

const reasoningSuffix = `

# 深度思考模式（必须遵守）
你现在处于深度思考模式。你必须严格按以下格式输出：

1. 先输出 <think> 标签
2. 在标签内写出完整的分析推理过程（数据对比、逻辑推导、可能原因）
3. 输出 </think> 标签结束推理
4. 然后输出正式的简洁回答

示例格式：
<think>
用户问的是批发增速问题...当前值是15%...对比上月...
</think>

根据分析，批发当月增速...

重要：必须先输出 <think> 再输出 </think>，不可省略。`

func (h *Handler) runAdjustMode(stream *llmStreamWriter, chatCtx llmChatContext) (*llmResultPayload, error) {
	_ = stream.Send(llmStreamEvent{Type: "thinking"})
	userMsg, err := lastUserMessage(chatCtx.request.Messages)
	if err != nil {
		return nil, err
	}

	intentClient, err := h.newLLMClient(llm.BuildIntentSystemPrompt())
	if err != nil {
		return nil, err
	}
	plan, err := llm.ParseIntent(intentClient, userMsg, buildIndicatorValueMap(chatCtx.groups))
	if err != nil {
		return nil, err
	}
	if plan == nil || len(plan.Actions) == 0 {
		return h.runChatMode(stream, chatCtx)
	}

	// 分离三类动作
	targets, ruleActions := splitActions(plan.Actions, buildIndicatorValueMap(chatCtx.groups))

	// 先处理规则添加
	var ruleAdded *ruleAddedPayload
	for _, action := range ruleActions {
		ruleAdded, err = h.addRuleFromChat(action.RuleText)
		if err != nil {
			log.Printf("add rule from chat error: %v", err)
			ruleAdded = &ruleAddedPayload{Text: action.RuleText, Status: "error"}
		}
	}

	// 如果只有规则操作没有调整目标，生成规则添加的总结
	if len(targets) == 0 {
		return h.buildRuleOnlyResult(stream, chatCtx, userMsg, ruleActions, ruleAdded)
	}

	h.engine.SetPeriod(chatCtx.year, chatCtx.month)
	resp, err := runOptimize(h.engine, h.store, chatCtx.year, chatCtx.month, targets)
	if err != nil {
		return nil, err
	}

	summaryPrompt := h.buildChatPrompt(chatCtx.year, chatCtx.month, resp.Groups, chatCtx.userPrompt)
	if chatCtx.reasoning {
		summaryPrompt += reasoningSuffix
	}
	summaryClient, err := h.newLLMClient(summaryPrompt)
	if err != nil {
		return nil, err
	}
	reply, err := h.streamReplyWithReasoning(stream, summaryClient, llm.ChatRequest{
		Messages: []llm.ChatMessage{{
			Role: "user",
			Content: buildAdjustSummaryMessage(
				userMsg,
				plan.Actions,
				resp.Groups,
				resp.AppliedRules,
			),
		}},
		Year:  chatCtx.year,
		Month: chatCtx.month,
	}, chatCtx.ctx, chatCtx.reasoning)
	if err != nil {
		return nil, err
	}

	// 收集直接调整的指标 ID，用于前端绿色高亮
	targetIDs := make([]string, 0, len(targets))
	for id := range targets {
		targetIDs = append(targetIDs, id)
	}

	return &llmResultPayload{
		Mode:            "adjust",
		Reply:           reply,
		Groups:          resp.Groups,
		AppliedRules:    resp.AppliedRules,
		RuleAdded:       ruleAdded,
		AdjustedTargets: targetIDs,
	}, nil
}

// splitActions 将 actions 分为目标调整（set_target + adjust_percent→set_target）和规则添加两组
func splitActions(actions []llm.AdjustmentAction, currentValues map[string]float64) (map[string]float64, []llm.AdjustmentAction) {
	targets := make(map[string]float64)
	var ruleActions []llm.AdjustmentAction
	for _, action := range actions {
		switch action.Type {
		case "set_target":
			targets[action.IndicatorID] = action.Value
		case "adjust_percent":
			current := currentValues[action.IndicatorID]
			targets[action.IndicatorID] = current * (1 + action.Percent/100)
		case "add_rule":
			ruleActions = append(ruleActions, action)
		}
	}
	return targets, ruleActions
}

// addRuleFromChat 通过聊天添加自然语言规则，直接写入数据库
func (h *Handler) addRuleFromChat(ruleText string) (*ruleAddedPayload, error) {
	text := strings.TrimSpace(ruleText)
	if text == "" {
		return nil, fmt.Errorf("规则内容为空")
	}
	_, err := h.store.CreateNaturalRule(text)
	if err != nil {
		return nil, err
	}
	return &ruleAddedPayload{Text: text, Status: "ok"}, nil
}

// buildRuleOnlyResult 仅添加规则时生成回复
func (h *Handler) buildRuleOnlyResult(
	stream *llmStreamWriter,
	chatCtx llmChatContext,
	userMsg string,
	ruleActions []llm.AdjustmentAction,
	ruleAdded *ruleAddedPayload,
) (*llmResultPayload, error) {
	summaryClient, err := h.newLLMClient(h.buildChatPrompt(chatCtx.year, chatCtx.month, chatCtx.groups, chatCtx.userPrompt))
	if err != nil {
		return nil, err
	}
	ruleTexts := make([]string, 0, len(ruleActions))
	for _, a := range ruleActions {
		ruleTexts = append(ruleTexts, a.RuleText)
	}
	summaryMsg := fmt.Sprintf(
		"用户请求：%s\n\n系统已添加以下自然语言规则：\n%s\n\n规则已保存，将在后续对话中自动生效。请用简洁中文告知用户规则已添加。",
		userMsg, strings.Join(ruleTexts, "\n"),
	)
	reply, err := h.streamReplyWithReasoning(stream, summaryClient, llm.ChatRequest{
		Messages: []llm.ChatMessage{{Role: "user", Content: summaryMsg}},
		Year:     chatCtx.year,
		Month:    chatCtx.month,
	}, chatCtx.ctx, false)
	if err != nil {
		return nil, err
	}
	return &llmResultPayload{
		Mode:      "adjust",
		Reply:     reply,
		RuleAdded: ruleAdded,
	}, nil
}

func (h *Handler) newLLMClient(prompt string) (LLMChatClient, error) {
	if h.llmClientFactory != nil {
		client, err := h.llmClientFactory(prompt)
		if err != nil {
			return nil, err
		}
		if client == nil {
			return nil, fmt.Errorf("初始化模型失败: empty client")
		}
		return client, nil
	}

	cfg, err := llm.LoadConfig(h.store)
	if err != nil {
		return nil, err
	}
	return llm.NewTextClient(cfg, prompt)
}

func (h *Handler) buildChatPrompt(year, month int, groups []dagcalc.IndicatorGroup, userPrompt string) string {
	naturalRules := h.loadNaturalRuleTexts()
	return llm.BuildChatSystemPrompt(llm.SystemPromptContext{
		Year:             year,
		Month:            month,
		ConstraintCount:  h.engine.ConstraintCount(),
		NaturalRules:     naturalRules,
		IndicatorSummary: buildIndicatorSummary(groups),
	}, userPrompt)
}

func (h *Handler) loadNaturalRuleTexts() []string {
	rules, err := h.store.ListNaturalRules(true)
	if err != nil {
		return nil
	}
	texts := make([]string, 0, len(rules))
	for _, r := range rules {
		texts = append(texts, r.Text)
	}
	return texts
}

func (h *Handler) streamReply(
	stream *llmStreamWriter,
	client LLMChatClient,
	req llm.ChatRequest,
	ctx context.Context,
) (string, error) {
	return h.streamReplyWithReasoning(stream, client, req, ctx, false)
}

func (h *Handler) streamReplyWithReasoning(
	stream *llmStreamWriter,
	client LLMChatClient,
	req llm.ChatRequest,
	ctx context.Context,
	reasoning bool,
) (string, error) {
	var builder strings.Builder
	streamed := false

	// 深度思考模式使用 reasoning parser 拆分 <think> 标签
	var parser *reasoningParser
	if reasoning {
		parser = newReasoningParser(stream)
	}

	result, err := client.Chat(ctx, req, func(chunk string) error {
		streamed = true
		builder.WriteString(chunk)
		if parser != nil {
			return parser.Feed(chunk)
		}
		return stream.SendDelta(chunk)
	})
	if err != nil {
		return "", err
	}

	// 刷新 parser 缓冲区
	if parser != nil {
		_ = parser.Flush()
	}

	content := strings.TrimSpace(result.Content)
	if !streamed && content != "" {
		// LLM 未走流式回调，将完整内容分小块逐步推送，模拟流式体验
		runes := []rune(content)
		chunkSize := 8
		for i := 0; i < len(runes); i += chunkSize {
			end := i + chunkSize
			if end > len(runes) {
				end = len(runes)
			}
			chunk := string(runes[i:end])
			builder.WriteString(chunk)
			if parser != nil {
				if err := parser.Feed(chunk); err != nil {
					return "", err
				}
			} else {
				if err := stream.SendDelta(chunk); err != nil {
					return "", err
				}
			}
			if end < len(runes) {
				time.Sleep(20 * time.Millisecond)
			}
		}
		if parser != nil {
			_ = parser.Flush()
		}
	}
	reply := strings.TrimSpace(builder.String())
	if reply == "" {
		reply = content
	}
	return reply, nil
}

func normalizeChatMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "chat":
		return "chat"
	case "adjust":
		return "adjust"
	default:
		return "chat"
	}
}

func lastUserMessage(messages []llmChatMessage) (string, error) {
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.EqualFold(strings.TrimSpace(messages[i].Role), "user") {
			content := strings.TrimSpace(messages[i].Content)
			if content != "" {
				return content, nil
			}
		}
	}
	return "", fmt.Errorf("缺少用户消息")
}

func buildIndicatorSummary(groups []dagcalc.IndicatorGroup) string {
	lines := make([]string, 0, 16)
	for _, group := range groups {
		for _, indicator := range group.Indicators {
			lines = append(lines,
				fmt.Sprintf("- %s %s = %.0f%s", indicator.ID, indicator.Name, indicator.Value, indicator.Unit),
			)
		}
	}
	if len(lines) == 0 {
		return "- 暂无指标快照"
	}
	return strings.Join(lines, "\n")
}

func buildIndicatorValueMap(groups []dagcalc.IndicatorGroup) map[string]float64 {
	values := make(map[string]float64, 16)
	for _, group := range groups {
		for _, indicator := range group.Indicators {
			values[indicator.ID] = indicator.Value
		}
	}
	return values
}

func buildAdjustSummaryMessage(
	userMsg string,
	actions []llm.AdjustmentAction,
	groups []dagcalc.IndicatorGroup,
	appliedRules []dagcalc.AppliedRule,
) string {
	lines := []string{
		"请根据以下真实执行结果，生成一段给用户看的简洁中文总结。",
		"",
		"用户原始请求：",
		userMsg,
		"",
		"已执行动作：",
	}
	for _, action := range actions {
		lines = append(lines,
			fmt.Sprintf("- %s -> %.0f", action.IndicatorID, action.Value),
		)
	}
	lines = append(lines, "", "规则生效情况：")
	ruleLines := formatAppliedRules(appliedRules)
	lines = append(lines, ruleLines...)
	lines = append(lines, "", "当前指标结果：")
	lines = append(lines, strings.Split(buildIndicatorSummary(groups), "\n")...)
	lines = append(lines,
		"",
		"要求：",
		"1. 明确说明系统已经实际执行了哪些调整",
		"2. 如有规则裁剪或过滤，要直接告诉用户",
		"3. 不要编造未执行的企业修改",
	)
	return strings.Join(lines, "\n")
}

func formatAppliedRules(appliedRules []dagcalc.AppliedRule) []string {
	if len(appliedRules) == 0 {
		return []string{"- 本次没有规则额外干预"}
	}

	lines := make([]string, 0, len(appliedRules))
	for _, rule := range appliedRules {
		switch rule.Type {
		case "clamp_target":
			lines = append(lines,
				fmt.Sprintf("- 规则 %s：%s 目标从 %.0f 裁剪到 %.0f",
					rule.RuleID,
					rule.IndicatorID,
					rule.BeforeValue,
					rule.AfterValue,
				),
			)
		case "filter_allocation":
			lines = append(lines,
				fmt.Sprintf("- 规则 %s：%s 参与企业从 %d 家过滤到 %d 家",
					rule.RuleID,
					rule.IndicatorID,
					rule.BeforeCount,
					rule.AfterCount,
				),
			)
		case "compensate":
			lines = append(lines,
				fmt.Sprintf("- 规则 %s：触发联动补偿 %s -> %.0f",
					rule.RuleID,
					rule.EnsureID,
					rule.TargetValue,
				),
			)
		default:
			lines = append(lines, fmt.Sprintf("- 规则 %s：%s", rule.RuleID, rule.Type))
		}
	}
	sort.Strings(lines)
	return lines
}

func (h *Handler) getConfigValue(key string, fallback string) string {
	value, err := h.store.GetConfig(key)
	if err != nil {
		return fallback
	}
	return value
}
