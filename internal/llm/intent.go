/**
 * AI 调整意图解析
 *
 * @author Anner
 * Created on 2026/3/14
 */
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var intentJSONCodeBlockPattern = regexp.MustCompile("(?s)```json\\s*(\\{.*?\\})\\s*```")

var allowedAdjustmentIndicatorIDs = map[string]struct{}{
	"limitAbove_month_value":        {},
	"limitAbove_month_rate":         {},
	"limitAbove_cumulative_value":   {},
	"limitAbove_cumulative_rate":    {},
	"eatWearUse_month_rate":         {},
	"microSmall_month_rate":         {},
	"wholesale_month_rate":          {},
	"wholesale_cumulative_rate":     {},
	"retail_month_rate":             {},
	"retail_cumulative_rate":        {},
	"accommodation_month_rate":      {},
	"accommodation_cumulative_rate": {},
	"catering_month_rate":           {},
	"catering_cumulative_rate":      {},
	"totalSocial_cumulative_value":  {},
	"totalSocial_cumulative_rate":   {},
}

// AdjustmentAction 表示一条结构化调整动作。
type AdjustmentAction struct {
	Type        string  `json:"type"`
	IndicatorID string  `json:"indicatorId,omitempty"`
	Value       float64 `json:"value,omitempty"`
	Percent     float64 `json:"percent,omitempty"`
	RuleText    string  `json:"ruleText,omitempty"`
}

// AdjustmentPlan 表示意图解析后的调整计划。
type AdjustmentPlan struct {
	Actions []AdjustmentAction `json:"actions"`
}

// IntentClient 是意图解析依赖的最小 LLM 接口。
type IntentClient interface {
	Chat(ctx context.Context, req ChatRequest, stream func(string) error) (ChatResult, error)
}

// BuildIntentSystemPrompt 返回结构化意图解析提示词。
func BuildIntentSystemPrompt() string {
	return strings.TrimSpace(`你是意图解析器，从用户输入和对话上下文提取调整动作。只输出纯 JSON，不要输出其他任何文字。

分类规则：
- “调到 X”、”设为 X” → set_target
- “调整 X%”、”增加/减少 X%” → adjust_percent
- “不能超过”、”限制”、”加一条规则” → add_rule
- 用户确认上一轮AI提出的调整方案（”可以”、”好的”、”执行”、”确认”）→ 从上下文中提取AI提出的具体调整动作
- 打招呼、纯咨询、提问 → {“actions”:[]}（空数组，不要猜测动作）

合法指标 ID（只能用这些）：
limitAbove_month_value, limitAbove_month_rate, limitAbove_cumulative_value, limitAbove_cumulative_rate,
eatWearUse_month_rate, microSmall_month_rate,
wholesale_month_rate, wholesale_cumulative_rate, retail_month_rate, retail_cumulative_rate,
accommodation_month_rate, accommodation_cumulative_rate, catering_month_rate, catering_cumulative_rate,
totalSocial_cumulative_value, totalSocial_cumulative_rate

常见中文名到 ID 映射：
批发 → wholesale, 零售 → retail, 住宿 → accommodation, 餐饮 → catering,
限上社零额 → limitAbove, 社零额 → totalSocial, 吃穿用 → eatWearUse, 小微 → microSmall,
当月增速 → _month_rate, 累计增速 → _cumulative_rate, 当月值 → _month_value, 累计值 → _cumulative_value

输出格式：
{“actions”:[{“type”:”set_target”,”indicatorId”:”wholesale_month_rate”,”value”:15}]}
`)
}

// ParseIntent 将用户输入解析为结构化调整计划。
// history 提供对话上下文，帮助理解指代和确认类消息（如"可以"、"好的"）。
func ParseIntent(client IntentClient, userMsg string, indicators map[string]float64, history []ChatMessage) (*AdjustmentPlan, error) {
	if client == nil {
		return nil, fmt.Errorf("intent client is nil")
	}

	messages := make([]ChatMessage, 0, len(history)+1)
	// 传入最近几轮对话上下文（最多 6 条），帮助理解指代
	if len(history) > 0 {
		start := 0
		if len(history) > 6 {
			start = len(history) - 6
		}
		for _, m := range history[start:] {
			messages = append(messages, ChatMessage{Role: m.Role, Content: m.Content})
		}
	}
	messages = append(messages, ChatMessage{
		Role:    "user",
		Content: buildIntentUserMessage(userMsg, indicators),
	})

	result, err := client.Chat(context.Background(), ChatRequest{
		Messages: messages,
	}, nil)
	if err != nil {
		return nil, err
	}

	jsonStr, err := extractIntentJSON(result.Content)
	if err != nil {
		return nil, err
	}

	var plan AdjustmentPlan
	if err := json.Unmarshal([]byte(jsonStr), &plan); err != nil {
		return nil, fmt.Errorf("parse adjustment plan failed: %w", err)
	}
	if plan.Actions == nil {
		plan.Actions = []AdjustmentAction{}
	}
	for idx, action := range plan.Actions {
		normalized, err := normalizeAdjustmentAction(idx, action)
		if err != nil {
			return nil, err
		}
		plan.Actions[idx] = normalized
	}
	return &plan, nil
}

func buildIntentUserMessage(userMsg string, indicators map[string]float64) string {
	lines := []string{
		"请把下面用户请求解析为结构化调整计划。",
		"",
		"当前指标：",
	}
	for _, line := range formatIndicatorValues(indicators) {
		lines = append(lines, line)
	}
	lines = append(lines,
		"",
		"用户请求：",
		strings.TrimSpace(userMsg),
		"",
		"请只输出纯 JSON。",
	)
	return strings.Join(lines, "\n")
}

func formatIndicatorValues(indicators map[string]float64) []string {
	if len(indicators) == 0 {
		return []string{"- 暂无指标"}
	}

	keys := make([]string, 0, len(indicators))
	for key := range indicators {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, "- "+key+" = "+strconv.FormatFloat(indicators[key], 'f', -1, 64))
	}
	return lines
}

func extractIntentJSON(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if matches := intentJSONCodeBlockPattern.FindStringSubmatch(trimmed); len(matches) == 2 {
		candidate := strings.TrimSpace(matches[1])
		if json.Valid([]byte(candidate)) {
			return candidate, nil
		}
	}
	if json.Valid([]byte(trimmed)) {
		return trimmed, nil
	}
	return "", fmt.Errorf("intent output is not valid JSON")
}

func normalizeAdjustmentAction(idx int, action AdjustmentAction) (AdjustmentAction, error) {
	if action.Type == "set_indicator" {
		action.Type = "set_target"
	}
	switch action.Type {
	case "set_target":
		if _, ok := allowedAdjustmentIndicatorIDs[action.IndicatorID]; !ok {
			return AdjustmentAction{}, fmt.Errorf("action[%d] indicatorId %q is invalid", idx, action.IndicatorID)
		}
	case "adjust_percent":
		if _, ok := allowedAdjustmentIndicatorIDs[action.IndicatorID]; !ok {
			return AdjustmentAction{}, fmt.Errorf("action[%d] indicatorId %q is invalid", idx, action.IndicatorID)
		}
	case "add_rule":
		if strings.TrimSpace(action.RuleText) == "" {
			return AdjustmentAction{}, fmt.Errorf("action[%d] add_rule requires ruleText", idx)
		}
	default:
		return AdjustmentAction{}, fmt.Errorf("action[%d] type %q is not supported", idx, action.Type)
	}
	return action, nil
}
