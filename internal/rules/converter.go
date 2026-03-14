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
	go func() {
		if err := c.convertAndApply(mdContent); err != nil {
			_ = c.writeError(err)
		}
	}()
}

func (c *Converter) convertAndApply(mdContent string) error {
	jsonStr, err := c.convert(mdContent)
	if err != nil {
		return err
	}
	if err := c.writeRoleJSON(jsonStr); err != nil {
		return err
	}
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
	for attempt := 0; attempt < 3; attempt++ {
		result, err := c.llm.Chat(context.Background(), llm.ChatRequest{Messages: messages}, nil)
		if err != nil {
			return "", err
		}
		jsonStr, err := extractJSON(result.Content)
		if err != nil {
			messages = appendRetryMessages(messages, result.Content,
				fmt.Sprintf("输出无法提取为 JSON，错误：%v，请只输出纯 JSON。", err))
			continue
		}
		errs := validateRoleJSON(jsonStr)
		if len(errs) == 0 {
			return jsonStr, nil
		}
		messages = appendRetryMessages(messages, result.Content, buildValidationErrorMessage(errs))
	}
	return "", fmt.Errorf("3次重试仍失败")
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

任务：
1. 读取用户提供的 rules.md 内容
2. 将自然语言规则转换为如下 JSON：
{
  "version": "1.0",
  "rules": [
    {
      "id": "rule_1",
      "name": "规则摘要",
      "type": "clamp_target|filter_allocation|compensate"
    }
  ]
}

约束：
- 只能输出纯 JSON，不要输出 Markdown 或解释
- indicator / trigger / ensure 只能使用系统允许的 16 个指标 ID
- filter 只能使用 positive_current、negative_current、large_scale_only、exclude_small_micro
- relation 只能使用 gte 或 lte
`)
}

func buildConvertUserMessage(mdContent string) string {
	return "请把以下 rules.md 转成 role.json：\n\n" + mdContent
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
	return c.store.SetConfig("rules_convert_error", "")
}

func (c *Converter) writeError(err error) error {
	if c.store == nil {
		return nil
	}
	if setErr := c.store.SetConfig("rules_convert_status", "error"); setErr != nil {
		return setErr
	}
	return c.store.SetConfig("rules_convert_error", err.Error())
}
