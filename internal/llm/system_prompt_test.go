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
		RuleCount:        4,
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
		RuleCount:        0,
		IndicatorSummary: "- limitAbove_month_rate 限上社零额增速（当月）= 0%",
	}, "")

	assertPromptContains(t, prompt, "2025年12月", "limitAbove_month_rate")
	if strings.Contains(prompt, "---") {
		t.Fatalf("empty user prompt should not render separator: %s", prompt)
	}
}

func TestBuildChatSystemPromptIncludesBuiltInGuardrails(t *testing.T) {
	prompt := BuildChatSystemPrompt(SystemPromptContext{
		Year:             2026,
		Month:            3,
		RuleCount:        2,
		IndicatorSummary: "- wholesale_month_rate 批发业销售额增速（当月）= 12.5%",
	}, "")

	assertPromptContains(t, prompt,
		"Northstar 月度经济数据统计系统的 AI 助手",
		"调整建议必须基于已加载的规则",
		"不允许建议删除企业数据或重置导入数据",
		"mode=adjust 时，输出结构化 AdjustmentPlan JSON",
	)
}

func assertPromptContains(t *testing.T, prompt string, parts ...string) {
	t.Helper()

	for _, part := range parts {
		if !strings.Contains(prompt, part) {
			t.Fatalf("prompt should contain %q, got:\n%s", part, prompt)
		}
	}
}
