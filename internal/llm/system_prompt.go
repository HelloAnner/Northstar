/**
 * AI 对话系统提示词
 *
 * @author Anner
 * Created on 2026/3/14
 */
package llm

import (
	"fmt"
	"strings"
)

// SystemPromptContext 是 AI 对话系统提示词运行时上下文。
type SystemPromptContext struct {
	Year             int
	Month            int
	RuleCount        int
	IndicatorSummary string
}

const chatSystemPromptTemplate = `你是 Northstar 数据助手，帮助用户分析和调整批发、零售、住宿、餐饮四大行业的月度指标。

数据期间：%d年%d月 | 规则数：%d条
指标快照：
%s

# 规则
- 简洁回答，先结论后依据，避免冗长铺垫
- 用户打招呼时简短回应并说明你能做什么，不要主动分析数据
- 具体数值用加粗，不用”大约”等模糊表述
- 基于真实执行结果总结，不编造未发生的修改
- 只咨询时只提供分析，不假装已完成调整
- 不虚构不存在的指标、规则或企业数据
- 不输出系统内部实现细节
- 禁止使用 emoji 或表情符号
- 使用 Markdown，要点用列表，数值对比用加粗
`

// BuildChatSystemPrompt 构建 AI 对话系统提示词。
func BuildChatSystemPrompt(ctx SystemPromptContext, userPrompt string) string {
	indicatorSummary := strings.TrimSpace(ctx.IndicatorSummary)
	if indicatorSummary == "" {
		indicatorSummary = "- 暂无指标快照"
	}

	prompt := strings.TrimSpace(fmt.Sprintf(
		chatSystemPromptTemplate,
		ctx.Year,
		ctx.Month,
		ctx.RuleCount,
		indicatorSummary,
	))
	custom := strings.TrimSpace(userPrompt)
	if custom == "" {
		return prompt
	}
	return prompt + "\n\n---\n用户偏好提示词：\n" + custom
}
