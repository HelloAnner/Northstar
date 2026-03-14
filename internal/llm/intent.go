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
	IndicatorID string  `json:"indicatorId"`
	Value       float64 `json:"value"`
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
	return strings.TrimSpace(`
你是 Northstar 的调整意图解析器。

任务：
1. 从用户输入中识别“要把哪个指标调整到多少”
2. 只输出纯 JSON，不要输出解释、Markdown 或额外文本
3. 若用户只是咨询，不要求系统执行调整，则返回 {"actions":[]}

输出格式：
{
  "actions": [
    {
      "type": "set_target",
      "indicatorId": "wholesale_month_rate",
      "value": 15
    }
  ]
}

约束：
- type 只能是 set_target
- indicatorId 只能使用系统提供的合法指标 ID
- value 必须是数字
`)
}

// ParseIntent 将用户输入解析为结构化调整计划。
func ParseIntent(client IntentClient, userMsg string, indicators map[string]float64) (*AdjustmentPlan, error) {
	if client == nil {
		return nil, fmt.Errorf("intent client is nil")
	}

	result, err := client.Chat(context.Background(), ChatRequest{
		Messages: []ChatMessage{{
			Role:    "user",
			Content: buildIntentUserMessage(userMsg, indicators),
		}},
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
	if action.Type != "set_target" {
		return AdjustmentAction{}, fmt.Errorf("action[%d] type must be set_target", idx)
	}
	if _, ok := allowedAdjustmentIndicatorIDs[action.IndicatorID]; !ok {
		return AdjustmentAction{}, fmt.Errorf("action[%d] indicatorId %q is invalid", idx, action.IndicatorID)
	}
	return action, nil
}
