/**
 * LLM 类型定义单元测试
 *
 * @author Anner
 * @since 12.0
 * Created on 2026/2/1
 */
package llm

import (
	"encoding/json"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

// TestParseToolCalls 测试工具调用解析
func TestParseToolCalls(t *testing.T) {
	tests := []struct {
		name           string
		calls          []llms.ToolCall
		expectErr      bool
		expectTargets  int
		expectUpdates  int
	}{
		{
			name:           "空调用列表",
			calls:          []llms.ToolCall{},
			expectErr:      false,
			expectTargets:  0,
			expectUpdates:  0,
		},
		{
			name: "仅设置指标目标",
			calls: []llms.ToolCall{
				{
					FunctionCall: &llms.FunctionCall{
						Name:      ToolSetIndicatorTargets,
						Arguments: `{"targets": {"limitAbove_month_rate": 7.5, "microSmall_month_rate": 30.0}}`,
					},
				},
			},
			expectErr:      false,
			expectTargets:  2,
			expectUpdates:  0,
		},
		{
			name: "仅更新企业",
			calls: []llms.ToolCall{
				{
					FunctionCall: &llms.FunctionCall{
						Name:      ToolUpdateCompanies,
						Arguments: `{"updates": [{"id": "wr:123", "patch": {"salesCurrentMonth": 1000}}]}`,
					},
				},
			},
			expectErr:      false,
			expectTargets:  0,
			expectUpdates:  1,
		},
		{
			name: "混合调用",
			calls: []llms.ToolCall{
				{
					FunctionCall: &llms.FunctionCall{
						Name:      ToolSetIndicatorTargets,
						Arguments: `{"targets": {"limitAbove_cumulative_rate": 8.0}}`,
					},
				},
				{
					FunctionCall: &llms.FunctionCall{
						Name:      ToolUpdateCompanies,
						Arguments: `{"updates": [{"id": "wr:1", "patch": {"retailCurrentMonth": 500}}, {"id": "ac:2", "patch": {"foodCurrentMonth": 300}}]}`,
					},
				},
			},
			expectErr:      false,
			expectTargets:  1,
			expectUpdates:  2,
		},
		{
			name: "无效 JSON",
			calls: []llms.ToolCall{
				{
					FunctionCall: &llms.FunctionCall{
						Name:      ToolSetIndicatorTargets,
						Arguments: `{"targets": invalid}`,
					},
				},
			},
			expectErr:      true,
			expectTargets:  0,
		},
		{
			name: "空 FunctionCall",
			calls: []llms.ToolCall{
				{FunctionCall: nil},
			},
			expectErr:      false,
			expectTargets:  0,
		},
		{
			name: "未知工具",
			calls: []llms.ToolCall{
				{
					FunctionCall: &llms.FunctionCall{
						Name:      "unknown_tool",
						Arguments: `{"data": "test"}`,
					},
				},
			},
			expectErr:      false,
			expectTargets:  0,
		},
		{
			name: "空目标",
			calls: []llms.ToolCall{
				{
					FunctionCall: &llms.FunctionCall{
						Name:      ToolSetIndicatorTargets,
						Arguments: `{"targets": {}}`,
					},
				},
			},
			expectErr:      false,
			expectTargets:  0,
		},
		{
			name: "空更新列表",
			calls: []llms.ToolCall{
				{
					FunctionCall: &llms.FunctionCall{
						Name:      ToolUpdateCompanies,
						Arguments: `{"updates": []}`,
					},
				},
			},
			expectErr:      false,
			expectUpdates:  0,
		},
		{
			name: "多个指标目标调用合并",
			calls: []llms.ToolCall{
				{
					FunctionCall: &llms.FunctionCall{
						Name:      ToolSetIndicatorTargets,
						Arguments: `{"targets": {"a": 1.0}}`,
					},
				},
				{
					FunctionCall: &llms.FunctionCall{
						Name:      ToolSetIndicatorTargets,
						Arguments: `{"targets": {"b": 2.0, "c": 3.0}}`,
					},
				},
			},
			expectErr:      false,
			expectTargets:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseToolCalls(tt.calls)

			if tt.expectErr {
				if err == nil {
					t.Error("期望错误但未返回")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误但返回: %v", err)
				}
				if len(parsed.IndicatorTargets) != tt.expectTargets {
					t.Errorf("指标目标数量期望 %d，实际 %d", tt.expectTargets, len(parsed.IndicatorTargets))
				}
				if len(parsed.CompanyUpdates) != tt.expectUpdates {
					t.Errorf("企业更新数量期望 %d，实际 %d", tt.expectUpdates, len(parsed.CompanyUpdates))
				}
			}
		})
	}
}

// TestParseToolCallsIndicatorValues 测试指标目标值解析
func TestParseToolCallsIndicatorValues(t *testing.T) {
	calls := []llms.ToolCall{
		{
			FunctionCall: &llms.FunctionCall{
				Name: ToolSetIndicatorTargets,
				Arguments: `{
					"targets": {
						"limitAbove_month_rate": 7.5,
						"limitAbove_cumulative_rate": 8.0,
						"microSmall_month_rate": 25.5,
						"eatWearUse_month_rate": 15.0
					}
				}`,
			},
		},
	}

	parsed, err := ParseToolCalls(calls)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	// 验证具体值
	expected := map[string]float64{
		"limitAbove_month_rate":      7.5,
		"limitAbove_cumulative_rate": 8.0,
		"microSmall_month_rate":      25.5,
		"eatWearUse_month_rate":      15.0,
	}

	for key, val := range expected {
		if parsed.IndicatorTargets[key] != val {
			t.Errorf("指标 %s 期望 %.1f，实际 %.1f", key, val, parsed.IndicatorTargets[key])
		}
	}
}

// TestParseToolCallsCompanyUpdates 测试企业更新解析
func TestParseToolCallsCompanyUpdates(t *testing.T) {
	calls := []llms.ToolCall{
		{
			FunctionCall: &llms.FunctionCall{
				Name: ToolUpdateCompanies,
				Arguments: `{
					"updates": [
						{"id": "wr:123", "patch": {"salesCurrentMonth": 1000.5, "retailCurrentMonth": 800}},
						{"id": "ac:456", "patch": {"revenueCurrentMonth": 500, "foodCurrentMonth": 300, "roomCurrentMonth": 200}},
						{"id": "wr:789", "patch": {"isSmallMicro": true, "isEatWearUse": false}}
					]
				}`,
			},
		},
	}

	parsed, err := ParseToolCalls(calls)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}

	if len(parsed.CompanyUpdates) != 3 {
		t.Fatalf("期望 3 个企业更新，实际 %d", len(parsed.CompanyUpdates))
	}

	// 验证第一个更新
	update1 := parsed.CompanyUpdates[0]
	if update1.ID != "wr:123" {
		t.Errorf("企业 ID 期望 'wr:123'，实际 '%s'", update1.ID)
	}
	if update1.Patch["salesCurrentMonth"] != 1000.5 {
		t.Errorf("salesCurrentMonth 期望 1000.5，实际 %v", update1.Patch["salesCurrentMonth"])
	}

	// 验证第三个更新（布尔值）
	update3 := parsed.CompanyUpdates[2]
	if update3.ID != "wr:789" {
		t.Errorf("企业 ID 期望 'wr:789'，实际 '%s'", update3.ID)
	}
	if update3.Patch["isSmallMicro"] != true {
		t.Errorf("isSmallMicro 期望 true，实际 %v", update3.Patch["isSmallMicro"])
	}
}

// TestChatMessageJSON 测试聊天消息的 JSON 序列化
func TestChatMessageJSON(t *testing.T) {
	msg := ChatMessage{
		Role:    "user",
		Content: "测试内容",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var decoded ChatMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if decoded.Role != msg.Role || decoded.Content != msg.Content {
		t.Error("反序列化后的消息不匹配")
	}
}

// TestChatRequestJSON 测试聊天请求的 JSON 序列化
func TestChatRequestJSON(t *testing.T) {
	req := ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "你好"},
			{Role: "assistant", Content: "你好！"},
		},
		Year:  2026,
		Month: 1,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("序列化失败: %v", err)
	}

	var decoded ChatRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if len(decoded.Messages) != len(req.Messages) {
		t.Error("消息数量不匹配")
	}
	if decoded.Year != req.Year || decoded.Month != req.Month {
		t.Error("年月不匹配")
	}
}

// TestParsedToolCallsEmpty 测试空解析结果
func TestParsedToolCallsEmpty(t *testing.T) {
	parsed := ParsedToolCalls{
		IndicatorTargets: map[string]float64{},
		CompanyUpdates:   []CompanyUpdate{},
	}

	if len(parsed.IndicatorTargets) != 0 {
		t.Error("指标目标应为空")
	}
	if len(parsed.CompanyUpdates) != 0 {
		t.Error("企业更新应为空")
	}
}

// TestCompanyUpdate 测试企业更新结构
func TestCompanyUpdate(t *testing.T) {
	update := CompanyUpdate{
		ID: "wr:123",
		Patch: map[string]interface{}{
			"salesCurrentMonth":  1000.0,
			"retailCurrentMonth": 800.0,
			"isSmallMicro":       true,
		},
	}

	// 验证 ID
	if update.ID != "wr:123" {
		t.Errorf("ID 不匹配: %s", update.ID)
	}

	// 验证 Patch 长度
	if len(update.Patch) != 3 {
		t.Errorf("Patch 长度期望 3，实际 %d", len(update.Patch))
	}

	// 验证具体值
	if update.Patch["salesCurrentMonth"] != 1000.0 {
		t.Errorf("salesCurrentMonth 不匹配")
	}
	if update.Patch["isSmallMicro"] != true {
		t.Errorf("isSmallMicro 不匹配")
	}
}

// TestChatResult 测试聊天结果结构
func TestChatResult(t *testing.T) {
	result := ChatResult{
		Content: "测试回复",
		ToolCalls: []llms.ToolCall{
			{
				FunctionCall: &llms.FunctionCall{
					Name:      ToolUpdateCompanies,
					Arguments: `{"updates": []}`,
				},
			},
		},
	}

	if result.Content != "测试回复" {
		t.Error("内容不匹配")
	}
	if len(result.ToolCalls) != 1 {
		t.Errorf("工具调用数量期望 1，实际 %d", len(result.ToolCalls))
	}
}
