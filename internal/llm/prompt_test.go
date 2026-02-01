/**
 * LLM 提示词单元测试
 *
 * @author Anner
 * @since 12.0
 * Created on 2026/2/1
 */
package llm

import (
	"strings"
	"testing"
)

// TestBuildSystemPrompt 测试系统提示词构建
func TestBuildSystemPrompt(t *testing.T) {
	tests := []struct {
		name    string
		ctx     PromptContext
		want    []string // 期望包含的子串
		notWant []string // 不期望包含的子串
	}{
		{
			name: "2026年1月",
			ctx: PromptContext{
				Year:  2026,
				Month: 1,
			},
			want: []string{
				"2026年1月",
				"Northstar",
				"set_indicator_targets",
				"update_companies",
				"limitAbove_month_value",
				"16",
			},
		},
		{
			name: "2025年12月",
			ctx: PromptContext{
				Year:  2025,
				Month: 12,
			},
			want: []string{
				"2025年12月",
			},
		},
		{
			name: "边界值测试",
			ctx: PromptContext{
				Year:  2000,
				Month: 6,
			},
			want: []string{
				"2000年6月",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := BuildSystemPrompt(tt.ctx)

			// 验证期望包含的内容
			for _, s := range tt.want {
				if !strings.Contains(prompt, s) {
					t.Errorf("提示词应包含 '%s'", s)
				}
			}

			// 验证不期望包含的内容
			for _, s := range tt.notWant {
				if strings.Contains(prompt, s) {
					t.Errorf("提示词不应包含 '%s'", s)
				}
			}

			// 验证提示词非空
			if len(prompt) == 0 {
				t.Error("提示词不应为空")
			}
		})
	}
}

// TestSystemPromptStructure 测试系统提示词结构
func TestSystemPromptStructure(t *testing.T) {
	ctx := PromptContext{Year: 2026, Month: 1}
	prompt := BuildSystemPrompt(ctx)

	// 验证关键章节存在
	sections := []string{
		"任务",
		"function call",
		"指标 ID",
		"16 项",
		"可修改的企业字段",
		"联动规则",
	}

	for _, section := range sections {
		if !strings.Contains(prompt, section) {
			t.Errorf("提示词应包含章节 '%s'", section)
		}
	}
}

// TestSystemPromptIndicatorIDs 测试指标ID完整性
func TestSystemPromptIndicatorIDs(t *testing.T) {
	ctx := PromptContext{Year: 2026, Month: 1}
	prompt := BuildSystemPrompt(ctx)

	// 所有16个指标ID都应在提示词中
	indicatorIDs := []string{
		"limitAbove_month_value",
		"limitAbove_month_rate",
		"limitAbove_cumulative_value",
		"limitAbove_cumulative_rate",
		"eatWearUse_month_rate",
		"microSmall_month_rate",
		"wholesale_month_rate",
		"wholesale_cumulative_rate",
		"retail_month_rate",
		"retail_cumulative_rate",
		"accommodation_month_rate",
		"accommodation_cumulative_rate",
		"catering_month_rate",
		"catering_cumulative_rate",
		"totalSocial_cumulative_value",
		"totalSocial_cumulative_rate",
	}

	for _, id := range indicatorIDs {
		if !strings.Contains(prompt, id) {
			t.Errorf("提示词应包含指标ID '%s'", id)
		}
	}
}

// TestSystemPromptFields 测试企业字段完整性
func TestSystemPromptFields(t *testing.T) {
	ctx := PromptContext{Year: 2026, Month: 1}
	prompt := BuildSystemPrompt(ctx)

	// 关键字段应在提示词中
	fields := []string{
		"salesCurrentMonth",
		"retailCurrentMonth",
		"revenueCurrentMonth",
		"foodCurrentMonth",
		"roomCurrentMonth",
		"goodsCurrentMonth",
		"isSmallMicro",
		"isEatWearUse",
	}

	for _, field := range fields {
		if !strings.Contains(prompt, field) {
			t.Errorf("提示词应包含字段 '%s'", field)
		}
	}
}

// TestPromptContext 测试提示词上下文结构
func TestPromptContext(t *testing.T) {
	ctx := PromptContext{
		Year:  2026,
		Month: 6,
	}

	if ctx.Year != 2026 {
		t.Errorf("Year 期望 2026，实际 %d", ctx.Year)
	}
	if ctx.Month != 6 {
		t.Errorf("Month 期望 6，实际 %d", ctx.Month)
	}
}

// TestSystemPromptNotTemplate 测试提示词不是模板字符串
func TestSystemPromptNotTemplate(t *testing.T) {
	ctx := PromptContext{Year: 2026, Month: 1}
	prompt := BuildSystemPrompt(ctx)

	// 验证 %d 和 %s 等格式化占位符已被替换
	if strings.Contains(prompt, "%d") {
		t.Errorf("提示词不应包含未替换的 %%d")
	}
	if strings.Contains(prompt, "%s") {
		t.Errorf("提示词不应包含未替换的 %%s")
	}
}

// TestSystemPromptFormat 测试提示词格式
func TestSystemPromptFormat(t *testing.T) {
	ctx := PromptContext{Year: 2026, Month: 1}
	prompt := BuildSystemPrompt(ctx)

	// 验证包含 Markdown 格式
	if !strings.Contains(prompt, "```") && !strings.Contains(prompt, "-") {
		t.Log("提示词可能缺少 Markdown 格式")
	}

	// 验证包含数字列表
	if !strings.Contains(prompt, "1)") {
		t.Log("提示词可能缺少编号列表")
	}

	// 验证提示词长度合理（应该超过 500 字符）
	if len(prompt) < 500 {
		t.Errorf("提示词长度 %d 过短，可能内容不完整", len(prompt))
	}
}
