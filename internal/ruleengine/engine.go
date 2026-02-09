/**
 * 可扩展规则引擎
 *
 * @author Anner
 * Created on 2026/2/6
 */

package ruleengine

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"northstar/internal/dagcalc"
	"northstar/internal/store"
)

const (
	StatusPass    = "pass"
	StatusFail    = "fail"
	StatusSkipped = "skipped"

	ruleNameIndustryGrowth = "规则P2-1 行业增速区间与差异约束"
	ruleNamePriority       = "规则P2-10 小微与吃穿用优先策略"
)

type RuleEvaluationIndicator struct {
	IndicatorCode string  `json:"indicatorCode"`
	IndicatorName string  `json:"indicatorName"`
	RelationLabel string  `json:"relationLabel,omitempty"`
	Weight        float64 `json:"weight,omitempty"`
	Value         float64 `json:"value"`
	Threshold     float64 `json:"threshold,omitempty"`
}

type RuleEvaluation struct {
	RuleCode          string                    `json:"ruleCode"`
	RuleName          string                    `json:"ruleName"`
	Description       string                    `json:"description"`
	Severity          string                    `json:"severity"`
	Expression        string                    `json:"expression"`
	Status            string                    `json:"status"`
	Message           string                    `json:"message"`
	Suggestion        string                    `json:"suggestion,omitempty"`
	FailedCount       int                       `json:"failedCount"`
	FailedIndicators  []RuleEvaluationIndicator `json:"failedIndicators"`
	SkippedReason     string                    `json:"skippedReason,omitempty"`
	EvaluatedBindings int                       `json:"evaluatedBindings"`
}

type evalContext struct {
	numbers    map[string]float64
	bools      map[string]bool
	indicators map[string]RuleEvaluationIndicator
}

func EvaluateRules(st *store.Store, year, month int, enabledOnly bool) ([]RuleEvaluation, error) {
	rules, err := st.ListRuleDefinitions(enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("list rule definitions failed: %w", err)
	}
	links, err := st.ListRuleIndicatorLinks("")
	if err != nil {
		return nil, fmt.Errorf("list rule links failed: %w", err)
	}

	ctx, err := buildEvalContext(st, year, month)
	if err != nil {
		return nil, err
	}

	linkMap := make(map[string][]store.RuleIndicatorLink)
	for _, link := range links {
		linkMap[link.RuleCode] = append(linkMap[link.RuleCode], link)
	}

	out := make([]RuleEvaluation, 0, len(rules))
	for _, rule := range rules {
		ruleLinks := linkMap[rule.RuleCode]
		evaluation, evalErr := evaluateRule(ctx, rule, ruleLinks)
		if evalErr != nil {
			return nil, evalErr
		}
		out = append(out, evaluation)
	}
	return out, nil
}

func buildEvalContext(st *store.Store, year, month int) (*evalContext, error) {
	groups, err := dagcalc.CalculateIndicators(st, year, month)
	if err != nil {
		return nil, fmt.Errorf("calculate indicators for rule context failed: %w", err)
	}

	numbers := make(map[string]float64, 128)
	bools := make(map[string]bool, 16)
	indicators := make(map[string]RuleEvaluationIndicator, 32)

	for _, group := range groups {
		for _, indicator := range group.Indicators {
			numbers[indicator.ID] = indicator.Value
			indicators[indicator.ID] = RuleEvaluationIndicator{
				IndicatorCode: indicator.ID,
				IndicatorName: indicator.Name,
				Value:         round2(indicator.Value),
			}
		}
	}

	configs, err := st.GetAllConfig()
	if err != nil {
		return nil, fmt.Errorf("load config for rule context failed: %w", err)
	}
	for key, raw := range configs {
		value := strings.TrimSpace(raw)
		lower := strings.ToLower(value)
		if lower == "true" {
			bools[key] = true
			numbers[key] = 1
			continue
		}
		if lower == "false" {
			bools[key] = false
			numbers[key] = 0
			continue
		}
		parsed, parseErr := strconv.ParseFloat(value, 64)
		if parseErr == nil {
			numbers[key] = parsed
		}
	}

	for key, alias := range configAliasMap {
		if value, ok := numbers[key]; ok {
			numbers[alias] = value
		}
		if value, ok := bools[key]; ok {
			bools[alias] = value
		}
	}

	return &evalContext{numbers: numbers, bools: bools, indicators: indicators}, nil
}

var configAliasMap = map[string]string{
	"rule_growth_abs_limit":                "行业增速绝对值上限",
	"rule_growth_jitter_limit":             "行业增速离散浮动阈值",
	"rule_wholesale_ratio_limit":           "批发零销比上限",
	"rule_wholesale_big_ratio_limit":       "批发大个体零销比上限",
	"rule_retail_gas_station_ratio_target": "乡镇加油站零销比目标",
	"rule_retail_big_growth_limit":         "零售大个体增速上限",
	"rule_room_food_delta_limit":           "住餐增速小数变动阈值",
	"rule_priority_target":                 "优先目标增速阈值",
}

func evaluateRule(ctx *evalContext, rule store.RuleDefinition, links []store.RuleIndicatorLink) (RuleEvaluation, error) {
	ruleKey := strings.TrimSpace(rule.Name)
	if ruleKey == "" {
		ruleKey = strings.TrimSpace(rule.RuleCode)
	}
	switch ruleKey {
	case ruleNameIndustryGrowth:
		return evaluateIndustryGrowthRule(ctx, rule, links), nil
	case ruleNamePriority:
		return evaluatePriorityRule(ctx, rule, links), nil
	default:
		return evaluateByExpression(ctx, rule, links), nil
	}
}

func evaluateIndustryGrowthRule(ctx *evalContext, rule store.RuleDefinition, links []store.RuleIndicatorLink) RuleEvaluation {
	limit := getNumberByKeys(ctx.numbers, 30, "行业增速绝对值上限", "rule_growth_abs_limit")
	if len(links) == 0 {
		return skippedRule(rule, "规则未配置联动指标")
	}

	failed := make([]RuleEvaluationIndicator, 0)
	evaluated := 0
	for _, link := range links {
		meta, ok := ctx.indicators[link.IndicatorCode]
		if !ok {
			continue
		}
		evaluated++
		if math.Abs(meta.Value) > limit {
			meta.Weight = link.Weight
			meta.RelationLabel = link.RelationLabel
			meta.Threshold = round2(limit)
			failed = append(failed, meta)
		}
	}

	if evaluated == 0 {
		return skippedRule(rule, "规则联动指标未命中当前指标上下文")
	}
	if len(failed) == 0 {
		return passedRule(rule, evaluated, fmt.Sprintf("所有联动指标绝对值均在 ±%.0f 范围内", limit))
	}
	return failedRule(rule, evaluated, failed, fmt.Sprintf("有 %d 个联动指标超过 ±%.0f", len(failed), limit))
}

func evaluatePriorityRule(ctx *evalContext, rule store.RuleDefinition, links []store.RuleIndicatorLink) RuleEvaluation {
	target := getNumberByKeys(ctx.numbers, 30, "优先目标增速阈值", "rule_priority_target")
	if len(links) == 0 {
		return skippedRule(rule, "规则未配置联动指标")
	}

	failed := make([]RuleEvaluationIndicator, 0)
	evaluated := 0
	for _, link := range links {
		meta, ok := ctx.indicators[link.IndicatorCode]
		if !ok {
			continue
		}
		evaluated++
		if meta.Value < target {
			meta.Weight = link.Weight
			meta.RelationLabel = link.RelationLabel
			meta.Threshold = round2(target)
			failed = append(failed, meta)
		}
	}

	if evaluated == 0 {
		return skippedRule(rule, "规则联动指标未命中当前指标上下文")
	}
	if len(failed) == 0 {
		return passedRule(rule, evaluated, fmt.Sprintf("联动指标均达到优先目标 %.0f", target))
	}
	return failedRule(rule, evaluated, failed, fmt.Sprintf("有 %d 个联动指标低于优先目标 %.0f", len(failed), target))
}

func evaluateByExpression(ctx *evalContext, rule store.RuleDefinition, links []store.RuleIndicatorLink) RuleEvaluation {
	expression := strings.TrimSpace(rule.Expression)
	if expression == "" {
		return skippedRule(rule, "规则表达式为空")
	}

	baseScope := expressionScope{
		baseNumbers: ctx.numbers,
		baseBools:   ctx.bools,
	}

	if len(links) == 0 {
		ok, err := evalRuleExpression(expression, baseScope)
		if err != nil {
			return skippedRule(rule, err.Error())
		}
		if ok {
			return passedRule(rule, 1, "规则表达式校验通过")
		}
		return failedRule(rule, 1, nil, "规则表达式校验未通过")
	}

	failed := make([]RuleEvaluationIndicator, 0)
	evaluated := 0
	skippedCount := 0
	lastError := ""
	for _, link := range links {
		meta, ok := ctx.indicators[link.IndicatorCode]
		if !ok {
			skippedCount++
			lastError = "联动指标未命中当前指标上下文"
			continue
		}

		scope := baseScope.withNumberOverrides(map[string]float64{
			"indicator_value":  meta.Value,
			"indicator_weight": link.Weight,
		})
		passed, err := evalRuleExpression(expression, scope)
		if err != nil {
			skippedCount++
			lastError = err.Error()
			continue
		}

		evaluated++
		if !passed {
			meta.Weight = link.Weight
			meta.RelationLabel = link.RelationLabel
			failed = append(failed, meta)
		}
	}

	if len(failed) > 0 {
		message := fmt.Sprintf("有 %d 个联动指标不满足规则表达式", len(failed))
		if skippedCount > 0 {
			message += fmt.Sprintf("（另有 %d 个联动项缺少上下文）", skippedCount)
		}
		return failedRule(rule, evaluated, failed, message)
	}
	if evaluated > 0 {
		message := "规则表达式校验通过"
		if skippedCount > 0 {
			message = fmt.Sprintf("%d 个联动指标通过，另有 %d 个联动项缺少上下文", evaluated, skippedCount)
		}
		return passedRule(rule, evaluated, message)
	}
	if lastError == "" {
		lastError = "没有可用于校验的上下文变量"
	}
	return skippedRule(rule, lastError)
}

func passedRule(rule store.RuleDefinition, evaluated int, message string) RuleEvaluation {
	return RuleEvaluation{
		RuleCode:          rule.RuleCode,
		RuleName:          rule.Name,
		Description:       rule.Description,
		Severity:          rule.Severity,
		Expression:        rule.Expression,
		Status:            StatusPass,
		Message:           message,
		Suggestion:        rule.Suggestion,
		FailedCount:       0,
		FailedIndicators:  []RuleEvaluationIndicator{},
		EvaluatedBindings: evaluated,
	}
}

func failedRule(rule store.RuleDefinition, evaluated int, indicators []RuleEvaluationIndicator, message string) RuleEvaluation {
	sorted := make([]RuleEvaluationIndicator, len(indicators))
	copy(sorted, indicators)
	sort.SliceStable(sorted, func(left, right int) bool {
		return sorted[left].IndicatorCode < sorted[right].IndicatorCode
	})
	return RuleEvaluation{
		RuleCode:          rule.RuleCode,
		RuleName:          rule.Name,
		Description:       rule.Description,
		Severity:          rule.Severity,
		Expression:        rule.Expression,
		Status:            StatusFail,
		Message:           message,
		Suggestion:        rule.Suggestion,
		FailedCount:       len(sorted),
		FailedIndicators:  sorted,
		EvaluatedBindings: evaluated,
	}
}

func skippedRule(rule store.RuleDefinition, reason string) RuleEvaluation {
	return RuleEvaluation{
		RuleCode:          rule.RuleCode,
		RuleName:          rule.Name,
		Description:       rule.Description,
		Severity:          rule.Severity,
		Expression:        rule.Expression,
		Status:            StatusSkipped,
		Message:           "规则校验已跳过",
		Suggestion:        rule.Suggestion,
		FailedCount:       0,
		FailedIndicators:  []RuleEvaluationIndicator{},
		SkippedReason:     reason,
		EvaluatedBindings: 0,
	}
}

func getNumberByKeys(values map[string]float64, fallback float64, keys ...string) float64 {
	for _, key := range keys {
		value, ok := values[key]
		if ok {
			return value
		}
	}
	return fallback
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
