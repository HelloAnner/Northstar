/**
 * 规则转换器
 *
 * @author Anner
 * Created on 2026/3/14
 */

package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"northstar/internal/llm"
	"northstar/internal/store"
)

var jsonCodeBlockPattern = regexp.MustCompile("(?s)```json\\s*(\\{.*?\\}|\\[.*?\\])\\s*```")

var allowedIndicatorIDs = map[string]struct{}{
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

var allowedFilterModes = map[string]struct{}{
	"positive_current":    {},
	"negative_current":    {},
	"large_scale_only":    {},
	"exclude_small_micro": {},
}

// LLM 调用超时时间
const llmCallTimeout = 90 * time.Second

type ConverterClient interface {
	Chat(ctx context.Context, req llm.ChatRequest, stream func(string) error) (llm.ChatResult, error)
}

type ruleConfigStore interface {
	SetConfig(key, value string) error
}

type ruleReloader interface {
	ReloadRules() error
}

// ValidationError 表示一条规则的结构化校验错误。
type ValidationError struct {
	RuleID  string
	Field   string
	Message string
}

// Converter 负责将 rules.md 转成 role.json 并热重载。
type Converter struct {
	llm      ConverterClient
	rolePath string
	mdPath   string
	store    ruleConfigStore
	engine   ruleReloader
}

// NewConverter 创建规则转换器。
func NewConverter(st *store.Store, engine ruleReloader, rolePath string, mdPath string) (*Converter, error) {
	cfg, err := llm.LoadConfig(st)
	if err != nil {
		return nil, err
	}
	client, err := llm.NewClient(cfg, buildConvertSystemPrompt())
	if err != nil {
		return nil, err
	}
	return &Converter{
		llm:      client,
		rolePath: rolePath,
		mdPath:   mdPath,
		store:    st,
		engine:   engine,
	}, nil
}

// NewConverterWithClient 使用外部注入的客户端创建转换器，便于测试与集成验证。
func NewConverterWithClient(
	client ConverterClient,
	st *store.Store,
	engine interface{ ReloadRules() error },
	rolePath string,
	mdPath string,
) *Converter {
	return &Converter{
		llm:      client,
		rolePath: rolePath,
		mdPath:   mdPath,
		store:    st,
		engine:   engine,
	}
}

// ConvertAsync 异步执行规则转换。
func (c *Converter) ConvertAsync(mdContent string) {
	if c == nil {
		return
	}
	_ = c.writeRunning()
	_ = c.writeStep("initializing")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				_ = c.writeError(fmt.Errorf("转换异常中断: %v", r))
			}
		}()
		if err := c.convertAndApply(mdContent); err != nil {
			_ = c.writeError(err)
		}
	}()
}

func (c *Converter) convertAndApply(mdContent string) error {
	_ = c.writeStep("calling_llm")
	jsonStr, err := c.convert(mdContent)
	if err != nil {
		return err
	}
	_ = c.writeStep("writing_json")
	if err := c.writeRoleJSON(jsonStr); err != nil {
		return err
	}
	_ = c.writeStep("reloading")
	if c.engine != nil {
		if err := c.engine.ReloadRules(); err != nil {
			return err
		}
	}
	return c.writeSuccess()
}

func (c *Converter) convert(mdContent string) (string, error) {
	if c == nil || c.llm == nil {
		return "", fmt.Errorf("规则转换器未初始化")
	}

	content := strings.TrimSpace(mdContent)
	if content == "" {
		loaded, err := c.loadMarkdownContent()
		if err != nil {
			return "", err
		}
		content = loaded
	}

	messages := []llm.ChatMessage{{
		Role:    "user",
		Content: buildConvertUserMessage(content),
	}}
	var lastError string
	for attempt := 0; attempt < 3; attempt++ {
		_ = c.writeAttempt(attempt+1, 3)
		ctx, cancel := context.WithTimeout(context.Background(), llmCallTimeout)
		result, err := c.llm.Chat(ctx, llm.ChatRequest{Messages: messages}, nil)
		cancel()
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败 (第%d次): %w", attempt+1, err)
		}
		_ = c.writeStep("validating")
		jsonStr, err := extractJSON(result.Content)
		if err != nil {
			lastError = fmt.Sprintf("第%d次: 输出无法提取为 JSON: %v", attempt+1, err)
			messages = appendRetryMessages(messages, result.Content,
				fmt.Sprintf("输出无法提取为 JSON，错误：%v，请只输出纯 JSON。", err))
			continue
		}
		errs := validateRoleJSON(jsonStr)
		if len(errs) == 0 {
			return normalizeRoleJSON(jsonStr, c.mdPath, time.Now().UTC())
		}
		lastError = fmt.Sprintf("第%d次: %s", attempt+1, buildValidationErrorMessage(errs))
		messages = appendRetryMessages(messages, result.Content, buildValidationErrorMessage(errs))
	}
	return "", fmt.Errorf("3次重试仍失败，最后一次错误: %s", lastError)
}

func appendRetryMessages(messages []llm.ChatMessage, assistant string, user string) []llm.ChatMessage {
	return append(messages,
		llm.ChatMessage{Role: "assistant", Content: assistant},
		llm.ChatMessage{Role: "user", Content: user},
	)
}

func extractJSON(content string) (string, error) {
	trimmed := strings.TrimSpace(content)
	if matches := jsonCodeBlockPattern.FindStringSubmatch(trimmed); len(matches) == 2 {
		candidate := strings.TrimSpace(matches[1])
		if json.Valid([]byte(candidate)) {
			return candidate, nil
		}
	}
	if json.Valid([]byte(trimmed)) {
		return trimmed, nil
	}
	return "", fmt.Errorf("未找到合法 JSON")
}

func validateRoleJSON(jsonStr string) []ValidationError {
	var raw rawRoleJSON
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return []ValidationError{{
			Field:   "json",
			Message: fmt.Sprintf("JSON 解析失败：%v", err),
		}}
	}

	errs := make([]ValidationError, 0)
	for _, item := range raw.Rules {
		errs = append(errs, validateRule(item)...)
	}
	return errs
}

func validateRule(item rawRule) []ValidationError {
	switch item.Type {
	case "clamp_target":
		return validateClampRule(item)
	case "filter_allocation":
		return validateFilterRule(item)
	case "compensate":
		return validateCompensateRule(item)
	default:
		return []ValidationError{newValidationError(item.ID, "type", fmt.Sprintf("未知规则类型 %q", item.Type))}
	}
}

func validateClampRule(item rawRule) []ValidationError {
	errs := validateIndicatorField(item.ID, "indicator", item.Indicator)
	if item.Min == nil && item.Max == nil {
		errs = append(errs, newValidationError(item.ID, "min", "min 和 max 不能同时为 null"))
	}
	return errs
}

func validateFilterRule(item rawRule) []ValidationError {
	errs := validateIndicatorField(item.ID, "indicator", item.Indicator)
	if _, ok := allowedFilterModes[item.Filter]; !ok {
		errs = append(errs, newValidationError(item.ID, "filter",
			fmt.Sprintf("%q 不在允许的 filter 枚举中", item.Filter)))
	}
	return errs
}

func validateCompensateRule(item rawRule) []ValidationError {
	errs := validateIndicatorField(item.ID, "trigger", item.Trigger)
	errs = append(errs, validateIndicatorField(item.ID, "ensure", item.Ensure)...)
	if item.Relation != "gte" && item.Relation != "lte" {
		errs = append(errs, newValidationError(item.ID, "relation", "relation 必须为 gte 或 lte"))
	}
	return errs
}

func validateIndicatorField(ruleID string, field string, indicator string) []ValidationError {
	if _, ok := allowedIndicatorIDs[indicator]; ok {
		return nil
	}
	return []ValidationError{newValidationError(ruleID, field,
		fmt.Sprintf("%q 不在允许的指标 ID 列表中", indicator))}
}

func newValidationError(ruleID string, field string, message string) ValidationError {
	return ValidationError{
		RuleID:  ruleID,
		Field:   field,
		Message: message,
	}
}

func buildValidationErrorMessage(errs []ValidationError) string {
	lines := []string{"以下规则校验失败，请修正后只输出纯 JSON："}
	for _, item := range errs {
		lines = append(lines, fmt.Sprintf("- 规则 %s 的字段 %s：%s", blankAsUnknown(item.RuleID), item.Field, item.Message))
	}
	return strings.Join(lines, "\n")
}

func blankAsUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<unknown>"
	}
	return value
}

func buildConvertSystemPrompt() string {
	return strings.TrimSpace(`
你是 Northstar 的规则转换器。

任务：将 rules.md 中的自然语言规则转成 JSON。只输出纯 JSON，不要 Markdown 或解释。

## 输出格式

{
  "version": "1.0",
  "rules": [...]
}

## 规则类型

### 1. clamp_target — 值域约束
字段：id, name, description, type, indicator, min(可选), max(可选)
示例：
{"id":"clamp_limitAbove_month_rate","name":"限上社零额月增速范围约束","description":"限上社零额月增速不低于 -30%，不高于 50%","type":"clamp_target","indicator":"limitAbove_month_rate","min":-30,"max":50}

### 2. filter_allocation — 过滤分配
字段：id, name, description, type, indicator, filter
filter 枚举：positive_current | negative_current | large_scale_only | exclude_small_micro
示例：
{"id":"filter_retail_positive","name":"零售仅正增长企业","description":"零售业仅正增长企业参与分配","type":"filter_allocation","indicator":"retail_month_rate","filter":"positive_current"}

### 3. compensate — 联动补偿
字段：id, name, description, type, trigger, ensure, relation, tolerance
relation 枚举：gte | lte
示例：
{"id":"compensate_limitAbove_to_totalSocial","name":"限上累计增速联动社零总额增速","description":"调整限上社零额累计增速后，社零总额累计增速不得低于限上增速 2 个百分点以内","type":"compensate","trigger":"limitAbove_cumulative_rate","ensure":"totalSocial_cumulative_rate","relation":"gte","tolerance":2}

## 允许的 16 个指标 ID（只能用这些，不要自创）

| 指标 ID | 中文含义 |
|---|---|
| limitAbove_month_value | 限上社零额月值 |
| limitAbove_month_rate | 限上社零额月增速 |
| limitAbove_cumulative_value | 限上社零额累计值 |
| limitAbove_cumulative_rate | 限上社零额累计增速 |
| eatWearUse_month_rate | 吃穿用月增速 |
| microSmall_month_rate | 小微企业月增速 |
| wholesale_month_rate | 批发业月增速 |
| wholesale_cumulative_rate | 批发业累计增速 |
| retail_month_rate | 零售业月增速 |
| retail_cumulative_rate | 零售业累计增速 |
| accommodation_month_rate | 住宿业月增速 |
| accommodation_cumulative_rate | 住宿业累计增速 |
| catering_month_rate | 餐饮业月增速 |
| catering_cumulative_rate | 餐饮业累计增速 |
| totalSocial_cumulative_value | 社零总额累计值 |
| totalSocial_cumulative_rate | 社零总额累计增速 |

## 约束
- min 和 max 是数值，不带百分号（如 -30 表示 -30%）
- min 或 max 可以只填一个，但不能同时为空
- "不低于 X" → min=X；"不高于 X" → max=X
- "不低于 0" → min=0（不需要 max）
- tolerance 是数值，表示允许偏差的百分点
`)
}

func buildConvertUserMessage(mdContent string) string {
	return "请把以下 rules.md 转成 role.json：\n\n" + mdContent
}

func normalizeRoleJSON(jsonStr string, mdPath string, now time.Time) (string, error) {
	var raw rawRoleJSON
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return "", err
	}
	if strings.TrimSpace(raw.Version) == "" {
		raw.Version = "1.0"
	}
	raw.UpdatedAt = now.Format(time.RFC3339)
	if strings.TrimSpace(raw.SourceFile) == "" {
		raw.SourceFile = normalizeSourceFile(mdPath)
	}
	for idx := range raw.Rules {
		if strings.TrimSpace(raw.Rules[idx].Description) == "" {
			raw.Rules[idx].Description = fallbackRuleDescription(raw.Rules[idx])
		}
	}
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func normalizeSourceFile(mdPath string) string {
	trimmed := strings.TrimSpace(filepath.ToSlash(mdPath))
	if trimmed == "" {
		return "config/rules.md"
	}
	if idx := strings.LastIndex(trimmed, "/config/"); idx >= 0 {
		return trimmed[idx+1:]
	}
	if strings.HasSuffix(trimmed, "/rules.md") {
		return "config/rules.md"
	}
	return trimmed
}

func fallbackRuleDescription(rule rawRule) string {
	if strings.TrimSpace(rule.Name) != "" {
		return strings.TrimSpace(rule.Name)
	}
	if strings.TrimSpace(rule.ID) != "" {
		return strings.TrimSpace(rule.ID)
	}
	return "规则描述"
}

func (c *Converter) loadMarkdownContent() (string, error) {
	if strings.TrimSpace(c.mdPath) == "" {
		return "", fmt.Errorf("rules.md 路径为空")
	}
	data, err := os.ReadFile(c.mdPath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *Converter) writeRoleJSON(content string) error {
	if strings.TrimSpace(c.rolePath) == "" {
		return fmt.Errorf("role.json 路径为空")
	}
	if err := os.MkdirAll(filepath.Dir(c.rolePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(c.rolePath, []byte(content), 0644)
}

func (c *Converter) writeRunning() error {
	if c.store == nil {
		return nil
	}
	if err := c.store.SetConfig("rules_convert_status", "running"); err != nil {
		return err
	}
	return c.store.SetConfig("rules_convert_error", "")
}

func (c *Converter) writeSuccess() error {
	if c.store == nil {
		return nil
	}
	if err := c.store.SetConfig("rules_convert_status", "ok"); err != nil {
		return err
	}
	if err := c.store.SetConfig("rules_convert_at", time.Now().Format(time.RFC3339)); err != nil {
		return err
	}
	_ = c.store.SetConfig("rules_convert_step", "")
	_ = c.store.SetConfig("rules_convert_attempt", "")
	return c.store.SetConfig("rules_convert_error", "")
}

func (c *Converter) writeError(err error) error {
	if c.store == nil {
		return nil
	}
	if setErr := c.store.SetConfig("rules_convert_status", "error"); setErr != nil {
		return setErr
	}
	_ = c.store.SetConfig("rules_convert_step", "")
	_ = c.store.SetConfig("rules_convert_attempt", "")
	return c.store.SetConfig("rules_convert_error", err.Error())
}

// writeStep 写入当前转换步骤，用于前端展示进度。
func (c *Converter) writeStep(step string) error {
	if c.store == nil {
		return nil
	}
	return c.store.SetConfig("rules_convert_step", step)
}

// writeAttempt 写入当前重试次数，格式 "1/3"。
func (c *Converter) writeAttempt(current int, total int) error {
	if c.store == nil {
		return nil
	}
	return c.store.SetConfig("rules_convert_attempt", fmt.Sprintf("%d/%d", current, total))
}
