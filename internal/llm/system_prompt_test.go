/**
 * AI 对话系统提示词测试
 *
 * @author Anner
 * Created on 2026/3/14
 */
package llm

import (
	"strings"
	"testing"
)

func TestBuildChatSystemPromptIncludesRuntimeContext(t *testing.T) {
	prompt := BuildChatSystemPrompt(SystemPromptContext{
		Year:             2026,
		Month:            3,
		ConstraintCount:        4,
		IndicatorSummary: "- wholesale_month_rate 批发业销售额增速（当月）= 12%\n- retail_month_rate 零售业销售额增速（当月）= 8%",
	}, "回答尽量直接，先给结论再解释原因。")

	assertPromptContains(t, prompt,
		"2026年3月",
		"4",
		"wholesale_month_rate",
		"retail_month_rate",
		"回答尽量直接",
		"---",
	)
}

func TestBuildChatSystemPromptSkipsUserSectionWhenEmpty(t *testing.T) {
	prompt := BuildChatSystemPrompt(SystemPromptContext{
		Year:             2025,
		Month:            12,
		ConstraintCount:        0,
		IndicatorSummary: "- limitAbove_month_rate 限上社零额增速（当月）= 0%",
	}, "")

	assertPromptContains(t, prompt, "2025年12月", "limitAbove_month_rate")
	if strings.Contains(prompt, "用户偏好提示词") {
		t.Fatalf("empty user prompt should not render user prompt section: %s", prompt)
	}
}

func TestBuildChatSystemPromptIncludesBackgroundKnowledge(t *testing.T) {
	prompt := BuildChatSystemPrompt(SystemPromptContext{
		Year:             2026,
		Month:            3,
		ConstraintCount:  0,
		IndicatorSummary: "- test = 0",
	}, "")

	assertPromptContains(t, prompt,
		"社会消费品零售总额",
		"限额以上",
		"批发业",
		"零售业",
		"住宿业",
		"餐饮业",
		"吃穿用",
		"小微企业",
		"调整机制",
	)
}

func TestBuildChatSystemPromptIncludesBuiltInGuardrails(t *testing.T) {
	prompt := BuildChatSystemPrompt(SystemPromptContext{
		Year:             2026,
		Month:            3,
		ConstraintCount:        2,
		IndicatorSummary: "- wholesale_month_rate 批发业销售额增速（当月）= 12.5%",
	}, "")

	assertPromptContains(t, prompt,
		"Northstar",
		"简洁回答",
		"不虚构不存在的指标",
		"禁止使用 emoji",
	)
}

func TestBuildChatSystemPromptUsesCustomBody(t *testing.T) {
	customBody := "这是自定义的业务背景。\n\n# 自定义回答规则\n- 用英文回答"
	prompt := BuildChatSystemPrompt(SystemPromptContext{
		Year:             2026,
		Month:            3,
		ConstraintCount:  0,
		IndicatorSummary: "- test = 0",
		SystemPromptBody: customBody,
	}, "")

	assertPromptContains(t, prompt,
		"Northstar",         // 角色定义始终存在
		"自定义的业务背景",  // 自定义正文生效
		"用英文回答",
		"2026年3月",         // 运行时数据仍注入
	)
	if strings.Contains(prompt, "社会消费品零售总额") {
		t.Fatalf("custom body should replace default, but default content still present")
	}
}

func TestBuildChatSystemPromptFallsBackToDefault(t *testing.T) {
	prompt := BuildChatSystemPrompt(SystemPromptContext{
		Year:             2026,
		Month:            3,
		ConstraintCount:  0,
		IndicatorSummary: "- test = 0",
		SystemPromptBody: "", // 空 → 回退默认
	}, "")

	assertPromptContains(t, prompt, "社会消费品零售总额", "调整机制")
}

func assertPromptContains(t *testing.T, prompt string, parts ...string) {
	t.Helper()

	for _, part := range parts {
		if !strings.Contains(prompt, part) {
			t.Fatalf("prompt should contain %q, got:\n%s", part, prompt)
		}
	}
}
