/**
 * LLM 工具定义
 *
 * @author Anner
 * @since 12.0
 * Created on 2026/2/1
 */
package llm

import "github.com/tmc/langchaingo/llms"

const (
	// ToolSetIndicatorTargets 指标目标工具
	ToolSetIndicatorTargets = "set_indicator_targets"
	// ToolUpdateCompanies 企业修改工具
	ToolUpdateCompanies = "update_companies"
	// ToolAddRule 添加持久规则工具
	ToolAddRule = "add_rule"
)

var indicatorTargetsSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"targets": map[string]any{
			"type":                 "object",
			"additionalProperties": map[string]any{"type": "number"},
			"description":          "指标目标值，key 为指标 ID",
		},
	},
	"required": []string{"targets"},
}

var companyPatchProperties = map[string]any{
	"salesCurrentMonth":       map[string]any{"type": "number"},
	"salesLastYearMonth":      map[string]any{"type": "number"},
	"salesCurrentCumulative":  map[string]any{"type": "number"},
	"salesLastYearCumulative": map[string]any{"type": "number"},
	"salesMonthRate":          map[string]any{"type": "number"},
	"salesCumulativeRate":     map[string]any{"type": "number"},

	"retailCurrentMonth":       map[string]any{"type": "number"},
	"retailLastYearMonth":      map[string]any{"type": "number"},
	"retailCurrentCumulative":  map[string]any{"type": "number"},
	"retailLastYearCumulative": map[string]any{"type": "number"},
	"retailMonthRate":          map[string]any{"type": "number"},
	"retailCumulativeRate":     map[string]any{"type": "number"},

	"revenueCurrentMonth":       map[string]any{"type": "number"},
	"revenueLastYearMonth":      map[string]any{"type": "number"},
	"revenueCurrentCumulative":  map[string]any{"type": "number"},
	"revenueLastYearCumulative": map[string]any{"type": "number"},
	"revenueMonthRate":          map[string]any{"type": "number"},
	"revenueCumulativeRate":     map[string]any{"type": "number"},

	"roomCurrentMonth":  map[string]any{"type": "number"},
	"foodCurrentMonth":  map[string]any{"type": "number"},
	"goodsCurrentMonth": map[string]any{"type": "number"},

	"isSmallMicro": map[string]any{"type": "boolean"},
	"isEatWearUse": map[string]any{"type": "boolean"},
}

var updateCompaniesSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"updates": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type":        "string",
						"description": "企业 ID，如 wr:123 或 ac:456",
					},
					"patch": map[string]any{
						"type":                 "object",
						"properties":           companyPatchProperties,
						"additionalProperties": true,
					},
				},
				"required": []string{"id", "patch"},
			},
		},
	},
	"required": []string{"updates"},
}

var addRuleSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"ruleText": map[string]any{
			"type":        "string",
			"description": "自然语言规则描述，如：限制批发当月增速不超过15%",
		},
	},
	"required": []string{"ruleText"},
}

// Tools 返回工具定义
func Tools() []llms.Tool {
	return []llms.Tool{indicatorTargetsTool(), updateCompaniesTool(), addRuleTool()}
}

func indicatorTargetsTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolSetIndicatorTargets,
			Description: "设置指标目标值，用于触发智能调整",
			Parameters:  indicatorTargetsSchema,
		},
	}
}

func updateCompaniesTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolUpdateCompanies,
			Description: "批量修改企业明细字段（必须提供 id 与 patch）",
			Parameters:  updateCompaniesSchema,
		},
	}
}

func addRuleTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        ToolAddRule,
			Description: "添加一条持久生效的调整规则，系统会将自然语言转换为结构化 JSON 规则并自动生效",
			Parameters:  addRuleSchema,
		},
	}
}
