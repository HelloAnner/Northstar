/**
 * 指标定义驱动计算
 *
 * @author Anner
 * Created on 2026/2/6
 */

package dagcalc

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"

	"northstar/internal/store"
)

func calculateByDefinitions(st *store.Store, year, month int, defs []store.IndicatorDefinition) ([]IndicatorGroup, error) {
	if len(defs) == 0 {
		return nil, nil
	}

	ctx, err := buildIndicatorContext(st, year, month)
	if err != nil {
		return nil, err
	}

	type groupBucket struct {
		Order int
		Name  string
		Items []Indicator
	}
	buckets := make([]groupBucket, 0, 8)
	groupIndex := make(map[string]int, 8)

	for _, def := range defs {
		value, evalErr := evalIndicatorFormula(def.Formula, ctx)
		if evalErr != nil {
			return nil, fmt.Errorf("eval indicator formula failed: code=%s formula=%s err=%w", def.Code, def.Formula, evalErr)
		}
		ctx[def.Code] = value

		idx, ok := groupIndex[def.GroupCode]
		if !ok {
			idx = len(buckets)
			groupIndex[def.GroupCode] = idx
			buckets = append(buckets, groupBucket{
				Order: def.GroupOrder,
				Name:  def.GroupName,
				Items: make([]Indicator, 0, 8),
			})
		}

		buckets[idx].Items = append(buckets[idx].Items, Indicator{
			ID:       def.Code,
			Name:     def.Name,
			Value:    value,
			Unit:     def.Unit,
			Formula:  def.Formula,
			FloatMin: def.FloatMin,
			FloatMax: def.FloatMax,
		})
	}

	// def 已按 group_order + display_order 排序，这里保持插入顺序
	out := make([]IndicatorGroup, 0, len(buckets))
	for _, bucket := range buckets {
		out = append(out, IndicatorGroup{
			Name:       bucket.Name,
			Indicators: bucket.Items,
		})
	}
	return out, nil
}

func buildIndicatorContext(st *store.Store, year, month int) (map[string]float64, error) {
	ctx := make(map[string]float64, 64)

	wrRetailCur, _, err := sumAndCountWR(st, year, month, "", "", "retail_current_month")
	if err != nil {
		return nil, err
	}
	wrRetailLast, _, err := sumAndCountWR(st, year, month, "", "", "retail_last_year_month")
	if err != nil {
		return nil, err
	}
	wrRetailCurCum, _, err := sumAndCountWR(st, year, month, "", "", "retail_current_cumulative")
	if err != nil {
		return nil, err
	}
	wrRetailLastCum, _, err := sumAndCountWR(st, year, month, "", "", "retail_last_year_cumulative")
	if err != nil {
		return nil, err
	}

	acRetailCur, _, err := sumAndCountACDerivedRetailMonth(st, year, month, "", true)
	if err != nil {
		return nil, err
	}
	acRetailLast, _, err := sumAndCountACDerivedRetailLastYearMonth(st, year, month, "")
	if err != nil {
		return nil, err
	}
	acRetailCurCum, _, err := sumAndCountACDerivedRetailCumulative(st, year, month, "", true)
	if err != nil {
		return nil, err
	}
	acRetailLastCum, _, err := sumAndCountACDerivedRetailLastYearCumulative(st, year, month, "")
	if err != nil {
		return nil, err
	}

	ctx["wr_retail_current_month_sum"] = wrRetailCur
	ctx["wr_retail_last_year_month_sum"] = wrRetailLast
	ctx["wr_retail_current_cumulative_sum"] = wrRetailCurCum
	ctx["wr_retail_last_year_cumulative_sum"] = wrRetailLastCum

	ctx["ac_derived_retail_current_month_sum"] = acRetailCur
	ctx["ac_derived_retail_last_year_month_sum"] = acRetailLast
	ctx["ac_derived_retail_current_cumulative_sum"] = acRetailCurCum
	ctx["ac_derived_retail_last_year_cumulative_sum"] = acRetailLastCum

	wrEatCur, _, err := sumAndCountWR(st, year, month, "", "is_eat_wear_use", "retail_current_month")
	if err != nil {
		return nil, err
	}
	wrEatLast, _, err := sumAndCountWR(st, year, month, "", "is_eat_wear_use", "retail_last_year_month")
	if err != nil {
		return nil, err
	}
	ctx["wr_eat_wear_use_current_month_sum"] = wrEatCur
	ctx["wr_eat_wear_use_last_year_month_sum"] = wrEatLast

	wrMicroCur, _, err := sumAndCountWR(st, year, month, "", "is_small_micro", "retail_current_month")
	if err != nil {
		return nil, err
	}
	wrMicroLast, _, err := sumAndCountWR(st, year, month, "", "is_small_micro", "retail_last_year_month")
	if err != nil {
		return nil, err
	}
	ctx["wr_micro_small_current_month_sum"] = wrMicroCur
	ctx["wr_micro_small_last_year_month_sum"] = wrMicroLast

	if err := putWRSalesIndustryMetrics(ctx, st, year, month, "wholesale"); err != nil {
		return nil, err
	}
	if err := putWRSalesIndustryMetrics(ctx, st, year, month, "retail"); err != nil {
		return nil, err
	}
	if err := putACRevenueIndustryMetrics(ctx, st, year, month, "accommodation"); err != nil {
		return nil, err
	}
	if err := putACRevenueIndustryMetrics(ctx, st, year, month, "catering"); err != nil {
		return nil, err
	}

	limitBelowEstimate, err := estimateLimitBelowCumulative(st, year, month)
	if err != nil {
		return nil, err
	}
	ctx["limit_below_last_cumulative"] = limitBelowEstimate.LastYearCumulative
	ctx["limit_below_current_cumulative"] = limitBelowEstimate.CurrentCumulative
	ctx["limit_below_month_rate"] = limitBelowEstimate.CurrentRate

	allConfig, cfgErr := st.GetAllConfig()
	if cfgErr == nil {
		for key, raw := range allConfig {
			value, parseErr := strconv.ParseFloat(raw, 64)
			if parseErr != nil {
				continue
			}
			ctx[key] = value
		}
	}

	applyIndicatorContextAliases(ctx)
	return ctx, nil
}

func putWRSalesIndustryMetrics(ctx map[string]float64, st *store.Store, year, month int, industry string) error {
	cur, _, err := sumAndCountWR(st, year, month, industry, "", "sales_current_month")
	if err != nil {
		return err
	}
	last, _, err := sumAndCountWR(st, year, month, industry, "", "sales_last_year_month")
	if err != nil {
		return err
	}
	curCum, _, err := sumAndCountWR(st, year, month, industry, "", "sales_current_cumulative")
	if err != nil {
		return err
	}
	lastCum, _, err := sumAndCountWR(st, year, month, industry, "", "sales_last_year_cumulative")
	if err != nil {
		return err
	}

	prefix := "wr_" + industry + "_sales_"
	ctx[prefix+"current_month_sum"] = cur
	ctx[prefix+"last_year_month_sum"] = last
	ctx[prefix+"current_cumulative_sum"] = curCum
	ctx[prefix+"last_year_cumulative_sum"] = lastCum
	return nil
}

func putACRevenueIndustryMetrics(ctx map[string]float64, st *store.Store, year, month int, industry string) error {
	cur, _, err := sumAndCountAC(st, year, month, industry, "revenue_current_month")
	if err != nil {
		return err
	}
	last, _, err := sumAndCountAC(st, year, month, industry, "revenue_last_year_month")
	if err != nil {
		return err
	}
	curCum, _, err := sumAndCountAC(st, year, month, industry, "revenue_current_cumulative")
	if err != nil {
		return err
	}
	lastCum, _, err := sumAndCountAC(st, year, month, industry, "revenue_last_year_cumulative")
	if err != nil {
		return err
	}

	prefix := "ac_" + industry + "_revenue_"
	ctx[prefix+"current_month_sum"] = cur
	ctx[prefix+"last_year_month_sum"] = last
	ctx[prefix+"current_cumulative_sum"] = curCum
	ctx[prefix+"last_year_cumulative_sum"] = lastCum
	return nil
}

func applyIndicatorContextAliases(ctx map[string]float64) {
	alias := map[string]string{
		"wr_retail_current_month_sum":                       "批零零售额_当月汇总",
		"wr_retail_last_year_month_sum":                     "批零零售额_上年当月汇总",
		"wr_retail_current_cumulative_sum":                  "批零零售额_累计汇总",
		"wr_retail_last_year_cumulative_sum":                "批零零售额_上年累计汇总",
		"ac_derived_retail_current_month_sum":               "住餐折算零售额_当月汇总",
		"ac_derived_retail_last_year_month_sum":             "住餐折算零售额_上年当月汇总",
		"ac_derived_retail_current_cumulative_sum":          "住餐折算零售额_累计汇总",
		"ac_derived_retail_last_year_cumulative_sum":        "住餐折算零售额_上年累计汇总",
		"wr_eat_wear_use_current_month_sum":                 "吃穿用零售额_当月汇总",
		"wr_eat_wear_use_last_year_month_sum":               "吃穿用零售额_上年当月汇总",
		"wr_micro_small_current_month_sum":                  "小微零售额_当月汇总",
		"wr_micro_small_last_year_month_sum":                "小微零售额_上年当月汇总",
		"wr_wholesale_sales_current_month_sum":              "批发销售额_当月汇总",
		"wr_wholesale_sales_last_year_month_sum":            "批发销售额_上年当月汇总",
		"wr_wholesale_sales_current_cumulative_sum":         "批发销售额_累计汇总",
		"wr_wholesale_sales_last_year_cumulative_sum":       "批发销售额_上年累计汇总",
		"wr_retail_sales_current_month_sum":                 "零售销售额_当月汇总",
		"wr_retail_sales_last_year_month_sum":               "零售销售额_上年当月汇总",
		"wr_retail_sales_current_cumulative_sum":            "零售销售额_累计汇总",
		"wr_retail_sales_last_year_cumulative_sum":          "零售销售额_上年累计汇总",
		"ac_accommodation_revenue_current_month_sum":        "住宿营业额_当月汇总",
		"ac_accommodation_revenue_last_year_month_sum":      "住宿营业额_上年当月汇总",
		"ac_accommodation_revenue_current_cumulative_sum":   "住宿营业额_累计汇总",
		"ac_accommodation_revenue_last_year_cumulative_sum": "住宿营业额_上年累计汇总",
		"ac_catering_revenue_current_month_sum":             "餐饮营业额_当月汇总",
		"ac_catering_revenue_last_year_month_sum":           "餐饮营业额_上年当月汇总",
		"ac_catering_revenue_current_cumulative_sum":        "餐饮营业额_累计汇总",
		"ac_catering_revenue_last_year_cumulative_sum":      "餐饮营业额_上年累计汇总",
		"small_micro_rate_prev":                             "小微增速_上月配置",
		"eat_wear_use_rate_prev":                            "吃穿用增速_上月配置",
		"sample_rate_prev":                                  "抽样增速_上月配置",
		"small_micro_rate_month":                            "小微增速_本月配置",
		"eat_wear_use_rate_month":                           "吃穿用增速_本月配置",
		"sample_rate_month":                                 "抽样增速_本月配置",
		"weight_small_micro":                                "小微权重_配置",
		"weight_eat_wear_use":                               "吃穿用权重_配置",
		"weight_sample":                                     "抽样权重_配置",
		"province_limit_below_rate_change":                  "全省限下增速变动量_配置",
		"limit_below_last_cumulative":                       "限下累计估算_上年值",
	}
	for source, target := range alias {
		value, ok := ctx[source]
		if !ok {
			continue
		}
		ctx[target] = value
	}
}

func evalIndicatorFormula(formula string, values map[string]float64) (float64, error) {
	expr, err := parser.ParseExpr(formula)
	if err != nil {
		return 0, err
	}
	return evalNumericExpr(expr, values)
}

func evalNumericExpr(expr ast.Expr, values map[string]float64) (float64, error) {
	switch node := expr.(type) {
	case *ast.BasicLit:
		if node.Kind != token.INT && node.Kind != token.FLOAT {
			return 0, fmt.Errorf("unsupported literal kind: %s", node.Kind.String())
		}
		return strconv.ParseFloat(node.Value, 64)
	case *ast.Ident:
		value, ok := values[node.Name]
		if !ok {
			return 0, fmt.Errorf("unknown identifier: %s", node.Name)
		}
		return value, nil
	case *ast.BinaryExpr:
		left, err := evalNumericExpr(node.X, values)
		if err != nil {
			return 0, err
		}
		right, err := evalNumericExpr(node.Y, values)
		if err != nil {
			return 0, err
		}
		switch node.Op {
		case token.ADD:
			return left + right, nil
		case token.SUB:
			return left - right, nil
		case token.MUL:
			return left * right, nil
		case token.QUO:
			if right == 0 {
				return 0, nil
			}
			return left / right, nil
		default:
			return 0, fmt.Errorf("unsupported binary operator: %s", node.Op.String())
		}
	case *ast.ParenExpr:
		return evalNumericExpr(node.X, values)
	case *ast.UnaryExpr:
		value, err := evalNumericExpr(node.X, values)
		if err != nil {
			return 0, err
		}
		switch node.Op {
		case token.ADD:
			return value, nil
		case token.SUB:
			return -value, nil
		default:
			return 0, fmt.Errorf("unsupported unary operator: %s", node.Op.String())
		}
	case *ast.CallExpr:
		fn, ok := node.Fun.(*ast.Ident)
		if !ok {
			return 0, fmt.Errorf("unsupported call expression")
		}
		return evalBuiltinFunc(fn.Name, node.Args, values)
	default:
		return 0, fmt.Errorf("unsupported expression node: %T", node)
	}
}

func evalBuiltinFunc(name string, args []ast.Expr, values map[string]float64) (float64, error) {
	switch name {
	case "percent_diff", "同比增速":
		if len(args) != 2 {
			return 0, fmt.Errorf("percent_diff args mismatch")
		}
		cur, err := evalNumericExpr(args[0], values)
		if err != nil {
			return 0, err
		}
		last, err := evalNumericExpr(args[1], values)
		if err != nil {
			return 0, err
		}
		if last == 0 {
			return 0, nil
		}
		return (cur - last) / last * 100, nil
	case "min", "最小值":
		if len(args) != 2 {
			return 0, fmt.Errorf("min args mismatch")
		}
		a, err := evalNumericExpr(args[0], values)
		if err != nil {
			return 0, err
		}
		b, err := evalNumericExpr(args[1], values)
		if err != nil {
			return 0, err
		}
		if a < b {
			return a, nil
		}
		return b, nil
	case "max", "最大值":
		if len(args) != 2 {
			return 0, fmt.Errorf("max args mismatch")
		}
		a, err := evalNumericExpr(args[0], values)
		if err != nil {
			return 0, err
		}
		b, err := evalNumericExpr(args[1], values)
		if err != nil {
			return 0, err
		}
		if a > b {
			return a, nil
		}
		return b, nil
	default:
		return 0, fmt.Errorf("unsupported function: %s", name)
	}
}
