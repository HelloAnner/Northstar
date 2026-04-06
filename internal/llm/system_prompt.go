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
	ConstraintCount  int
	NaturalRules     []string
	IndicatorSummary string
	SystemPromptBody string // 用户可编辑的系统提示词正文（业务背景+回答规则）
}

// DefaultSystemPromptBody 是系统提示词的默认正文，首次写入数据库。
const DefaultSystemPromptBody = `# 业务背景

本系统是月度经济数据构造工具，用于生成和调整批发、零售、住宿、餐饮四大行业的企业级月度数据，使汇总后的宏观指标（增速、总额）达到预期目标。所有数据均为构造数据，允许自由调整。

社会消费品零售总额（社零总额）是衡量消费市场的核心指标，由"限额以上"和"限额以下"两部分构成。

限额以上（限上）企业按月逐户上报数据，是本系统管理的主体。限额以下（限下）通过抽样推算，不逐户管理。社零总额 = 限上 + 限下。

## 四大行业

| 行业 | 核心字段 | 说明 |
|------|----------|------|
| 批发业 | 商品销售额 | 大宗商品流通，销售额大但零售额占比小 |
| 零售业 | 商品销售额、零售额 | 直接面向消费者，是限上社零额的主力 |
| 住宿业 | 营业额（客房+餐费+商品） | 以客房收入为主 |
| 餐饮业 | 营业额（餐费+商品） | 以餐费收入为主 |

批发和零售合称"批零"（wholesale_retail），住宿和餐饮合称"住餐"（accommodation_catering）。

## 指标体系

**限上社零额**（limitAbove）：所有限上企业的零售额汇总，是最核心的上报指标。
- 当月值 = 批零零售额当月汇总 + 住餐折算零售额当月汇总
- 当月增速 = (当月值 - 上年同期) / 上年同期 × 100
- 累计值 = 本年 1 月到本月的累加
- 累计增速 = (累计值 - 上年累计) / 上年累计 × 100

**四大行业增速**：每个行业有当月增速和累计增速。
- 批发/零售按"销售额"算增速
- 住宿/餐饮按"营业额"算增速

**特殊指标**：
- 吃穿用增速（eatWearUse）：粮油食品 + 饮料 + 烟酒 + 服装 + 日用品的零售额增速
- 小微企业增速（microSmall）：规模为 3/4 的企业零售额增速
- 社零总额（totalSocial）：限上 + 限下，仅有累计值和累计增速

## 调整机制

系统通过修改企业的"本月值"（销售额/营业额）来反向达成目标增速。调整时会：
1. 将目标增速换算为目标汇总值
2. 计算目标值与当前值的差额
3. 按企业上年同期的占比，将差额分配到各企业
4. 硬约束在分配过程中自动生效（clamp 裁剪目标值、filter 过滤参与企业、compensate 联动补偿）

# 回答规则
- 简洁回答，先结论后依据
- 用户打招呼时简短回应并说明你能做什么
- 具体数值用加粗，不用"大约"等模糊表述
- 调整完成后如实报告执行结果和规则生效情况
- 主动提出合理的调整建议，帮助用户构造符合预期的数据
- 禁止使用 emoji 或表情符号
- 使用 Markdown，要点用列表，数值对比用加粗`

const systemPromptHeader = `你是 Northstar 数据构造助手，帮助用户构造和调整批发、零售、住宿、餐饮四大行业的月度经济数据，使各项宏观指标达到预期目标。`

const systemPromptRuntimeTemplate = `
---

数据期间：%d年%d月 | 硬约束：%d条
指标快照：
%s`

// BuildChatSystemPrompt 构建 AI 对话系统提示词。
func BuildChatSystemPrompt(ctx SystemPromptContext, userPrompt string) string {
	indicatorSummary := strings.TrimSpace(ctx.IndicatorSummary)
	if indicatorSummary == "" {
		indicatorSummary = "- 暂无指标快照"
	}

	// 角色定义（固定）
	prompt := systemPromptHeader

	// 用户可编辑的正文（业务背景 + 回答规则）
	body := strings.TrimSpace(ctx.SystemPromptBody)
	if body == "" {
		body = DefaultSystemPromptBody
	}
	prompt += "\n\n" + body

	// 运行时数据（固定格式，动态填充）
	prompt += fmt.Sprintf(systemPromptRuntimeTemplate,
		ctx.Year,
		ctx.Month,
		ctx.ConstraintCount,
		indicatorSummary,
	)

	// 注入自然语言规则
	if len(ctx.NaturalRules) > 0 {
		prompt += "\n\n---\n用户定义的调整规则（请在调整时遵守）：\n"
		for i, rule := range ctx.NaturalRules {
			prompt += fmt.Sprintf("%d. %s\n", i+1, rule)
		}
	}

	custom := strings.TrimSpace(userPrompt)
	if custom == "" {
		return prompt
	}
	return prompt + "\n\n---\n用户偏好提示词：\n" + custom
}
