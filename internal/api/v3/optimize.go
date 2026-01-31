package v3

import (
	"fmt"
	"math"
	"net/http"
	"sort"

	"github.com/gin-gonic/gin"

	"northstar/internal/calculator"
	"northstar/internal/store"
)

type OptimizeRequest struct {
	Targets map[string]float64 `json:"targets"`
}

// Optimize 执行智能调整（按目标指标反推并写回企业数据）
// POST /api/optimize
func (h *Handler) Optimize(c *gin.Context) {
	var req OptimizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if len(req.Targets) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "targets 不能为空"})
		return
	}

	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取当前年月失败"})
		return
	}

	ordered := orderTargets(req.Targets)
	for _, item := range ordered {
		if err := applyIndicatorTarget(h.store, year, month, item.ID, item.Value); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "indicatorId": item.ID})
			return
		}
	}

	if err := recalcDerivedFields(h.store, year, month); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重算衍生字段失败"})
		return
	}

	calc := calculator.NewCalculator(h.store)
	groups, err := calc.CalculateAll(year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "计算指标失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"year":   year,
		"month":  month,
		"groups": groups,
	})
}

type orderedTarget struct {
	ID    string
	Value float64
}

func orderTargets(targets map[string]float64) []orderedTarget {
	knownOrder := []string{
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

	out := make([]orderedTarget, 0, len(targets))
	seen := map[string]bool{}
	for _, id := range knownOrder {
		if v, ok := targets[id]; ok {
			out = append(out, orderedTarget{ID: id, Value: v})
			seen[id] = true
		}
	}
	var rest []string
	for id := range targets {
		if !seen[id] {
			rest = append(rest, id)
		}
	}
	sort.Strings(rest)
	for _, id := range rest {
		out = append(out, orderedTarget{ID: id, Value: targets[id]})
	}
	return out
}

func applyIndicatorTarget(st *store.Store, year, month int, id string, target float64) error {
	if math.IsNaN(target) || math.IsInf(target, 0) {
		return fmt.Errorf("无效目标值: %s", id)
	}

	switch id {
	case "limitAbove_month_value":
		return adjustLimitAboveMonthValue(st, year, month, target)
	case "limitAbove_month_rate":
		return adjustLimitAboveMonthRate(st, year, month, target)
	case "limitAbove_cumulative_value":
		return adjustLimitAboveCumulativeValue(st, year, month, target)
	case "limitAbove_cumulative_rate":
		return adjustLimitAboveCumulativeRate(st, year, month, target)
	case "eatWearUse_month_rate":
		return adjustWRSpecialRate(st, year, month, "is_eat_wear_use", target)
	case "microSmall_month_rate":
		return adjustWRSpecialRate(st, year, month, "is_small_micro", target)
	case "wholesale_month_rate":
		return adjustWRIndustryRate(st, year, month, "wholesale", "sales_current_month", "sales_last_year_month", target)
	case "wholesale_cumulative_rate":
		return adjustWRIndustryRate(st, year, month, "wholesale", "sales_current_cumulative", "sales_last_year_cumulative", target)
	case "retail_month_rate":
		return adjustWRIndustryRate(st, year, month, "retail", "sales_current_month", "sales_last_year_month", target)
	case "retail_cumulative_rate":
		return adjustWRIndustryRate(st, year, month, "retail", "sales_current_cumulative", "sales_last_year_cumulative", target)
	case "accommodation_month_rate":
		return adjustACIndustryRate(st, year, month, "accommodation", "revenue_current_month", "revenue_last_year_month", target)
	case "accommodation_cumulative_rate":
		return adjustACIndustryRate(st, year, month, "accommodation", "revenue_current_cumulative", "revenue_last_year_cumulative", target)
	case "catering_month_rate":
		return adjustACIndustryRate(st, year, month, "catering", "revenue_current_month", "revenue_last_year_month", target)
	case "catering_cumulative_rate":
		return adjustACIndustryRate(st, year, month, "catering", "revenue_current_cumulative", "revenue_last_year_cumulative", target)
	case "totalSocial_cumulative_value":
		return adjustTotalSocialCumulativeValue(st, year, month, target)
	case "totalSocial_cumulative_rate":
		return adjustTotalSocialCumulativeRate(st, year, month, target)
	default:
		return fmt.Errorf("不支持的指标: %s", id)
	}
}

func adjustLimitAboveMonthValue(st *store.Store, year, month int, target float64) error {
	if target < 0 {
		target = 0
	}

	wrSum, wrCount, err := sumAndCountWR(st, year, month, "", "", "retail_current_month")
	if err != nil {
		return err
	}
	acSum, acCount, err := sumAndCountAC(st, year, month, "", "retail_current_month")
	if err != nil {
		return err
	}

	return scaleAcrossWRAndAC(st, year, month, "retail_current_month", wrSum, wrCount, acSum, acCount, target)
}

func adjustLimitAboveMonthRate(st *store.Store, year, month int, targetRate float64) error {
	lastYearSumWR, _, err := sumAndCountWR(st, year, month, "", "", "retail_last_year_month")
	if err != nil {
		return err
	}
	lastYearSumAC, _, err := sumAndCountAC(st, year, month, "", "retail_last_year_month")
	if err != nil {
		return err
	}

	desired := lastYearSumWR + lastYearSumAC
	desired = desired * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}

	wrSum, wrCount, err := sumAndCountWR(st, year, month, "", "", "retail_current_month")
	if err != nil {
		return err
	}
	acSum, acCount, err := sumAndCountAC(st, year, month, "", "retail_current_month")
	if err != nil {
		return err
	}

	return scaleAcrossWRAndAC(st, year, month, "retail_current_month", wrSum, wrCount, acSum, acCount, desired)
}

func adjustLimitAboveCumulativeValue(st *store.Store, year, month int, target float64) error {
	if target < 0 {
		target = 0
	}
	currentSum, count, err := sumAndCountWR(st, year, month, "", "", "retail_current_cumulative")
	if err != nil {
		return err
	}
	return scaleWRField(st, year, month, "", "", "retail_current_cumulative", currentSum, count, target)
}

func adjustLimitAboveCumulativeRate(st *store.Store, year, month int, targetRate float64) error {
	lastYearSum, _, err := sumAndCountWR(st, year, month, "", "", "retail_last_year_cumulative")
	if err != nil {
		return err
	}
	desired := lastYearSum * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}

	currentSum, count, err := sumAndCountWR(st, year, month, "", "", "retail_current_cumulative")
	if err != nil {
		return err
	}
	return scaleWRField(st, year, month, "", "", "retail_current_cumulative", currentSum, count, desired)
}

func adjustWRSpecialRate(st *store.Store, year, month int, flagField string, targetRate float64) error {
	currentSum, count, err := sumAndCountWR(st, year, month, "", flagField, "retail_current_month")
	if err != nil {
		return err
	}
	lastYearSum, _, err := sumAndCountWR(st, year, month, "", flagField, "retail_last_year_month")
	if err != nil {
		return err
	}

	desired := lastYearSum * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}

	return scaleWRField(st, year, month, "", flagField, "retail_current_month", currentSum, count, desired)
}

func adjustWRIndustryRate(st *store.Store, year, month int, industryType, currentField, lastYearField string, targetRate float64) error {
	currentSum, count, err := sumAndCountWR(st, year, month, industryType, "", currentField)
	if err != nil {
		return err
	}
	lastYearSum, _, err := sumAndCountWR(st, year, month, industryType, "", lastYearField)
	if err != nil {
		return err
	}

	desired := lastYearSum * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}

	return scaleWRField(st, year, month, industryType, "", currentField, currentSum, count, desired)
}

func adjustACIndustryRate(st *store.Store, year, month int, industryType, currentField, lastYearField string, targetRate float64) error {
	currentSum, count, err := sumAndCountAC(st, year, month, industryType, currentField)
	if err != nil {
		return err
	}

	lastYearSum, _, err := sumAndCountAC(st, year, month, industryType, lastYearField)
	if err != nil {
		return err
	}

	desired := lastYearSum * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}

	return scaleACField(st, year, month, industryType, currentField, currentSum, count, desired)
}

func adjustTotalSocialCumulativeValue(st *store.Store, year, month int, target float64) error {
	if target < 0 {
		target = 0
	}

	limitBelowLastYear, err := st.GetConfigFloat("last_year_limit_below_cumulative")
	if err != nil {
		limitBelowLastYear = 0
	}

	microRate, err := computeMicroSmallRate(st, year, month)
	if err != nil {
		return err
	}

	limitBelowEstimated := limitBelowLastYear * (1 + microRate/100)
	desiredLimitAbove := target - limitBelowEstimated
	if desiredLimitAbove < 0 {
		desiredLimitAbove = 0
	}

	currentSum, count, err := sumAndCountWR(st, year, month, "", "", "retail_current_cumulative")
	if err != nil {
		return err
	}
	return scaleWRField(st, year, month, "", "", "retail_current_cumulative", currentSum, count, desiredLimitAbove)
}

func adjustTotalSocialCumulativeRate(st *store.Store, year, month int, targetRate float64) error {
	limitBelowLastYear, err := st.GetConfigFloat("last_year_limit_below_cumulative")
	if err != nil {
		limitBelowLastYear = 0
	}

	microRate, err := computeMicroSmallRate(st, year, month)
	if err != nil {
		return err
	}

	retailLastYearCumulativeSum, _, err := sumAndCountWR(st, year, month, "", "", "retail_last_year_cumulative")
	if err != nil {
		return err
	}

	targetFraction := targetRate / 100
	desiredLimitAbove := retailLastYearCumulativeSum*(1+targetFraction) + limitBelowLastYear*(targetFraction-microRate/100)
	if desiredLimitAbove < 0 {
		desiredLimitAbove = 0
	}

	currentSum, count, err := sumAndCountWR(st, year, month, "", "", "retail_current_cumulative")
	if err != nil {
		return err
	}
	return scaleWRField(st, year, month, "", "", "retail_current_cumulative", currentSum, count, desiredLimitAbove)
}

func computeMicroSmallRate(st *store.Store, year, month int) (float64, error) {
	currentSum, _, err := sumAndCountWR(st, year, month, "", "is_small_micro", "retail_current_month")
	if err != nil {
		return 0, err
	}
	lastYearSum, _, err := sumAndCountWR(st, year, month, "", "is_small_micro", "retail_last_year_month")
	if err != nil {
		return 0, err
	}
	if lastYearSum == 0 {
		return 0, nil
	}
	return (currentSum - lastYearSum) / lastYearSum * 100, nil
}

func sumAndCountWR(st *store.Store, year, month int, industryType string, flagField string, field string) (float64, int, error) {
	where := "data_year = ? AND data_month = ?"
	args := []interface{}{year, month}

	if industryType != "" {
		where += " AND industry_type = ?"
		args = append(args, industryType)
	}
	if flagField != "" {
		where += fmt.Sprintf(" AND %s = 1", flagField)
	}

	query := fmt.Sprintf("SELECT COALESCE(SUM(%s), 0), COUNT(1) FROM wholesale_retail WHERE %s", field, where)
	var sum float64
	var count int
	if err := st.QueryRow(query, args...).Scan(&sum, &count); err != nil {
		return 0, 0, err
	}
	return sum, count, nil
}

func sumAndCountAC(st *store.Store, year, month int, industryType string, field string) (float64, int, error) {
	where := "data_year = ? AND data_month = ?"
	args := []interface{}{year, month}
	if industryType != "" {
		where += " AND industry_type = ?"
		args = append(args, industryType)
	}
	query := fmt.Sprintf("SELECT COALESCE(SUM(%s), 0), COUNT(1) FROM accommodation_catering WHERE %s", field, where)
	var sum float64
	var count int
	if err := st.QueryRow(query, args...).Scan(&sum, &count); err != nil {
		return 0, 0, err
	}
	return sum, count, nil
}

func scaleAcrossWRAndAC(st *store.Store, year, month int, field string, wrSum float64, wrCount int, acSum float64, acCount int, target float64) error {
	totalSum := wrSum + acSum
	totalCount := wrCount + acCount
	if totalCount == 0 {
		return fmt.Errorf("没有可调整数据")
	}

	if totalSum == 0 {
		perRow := 0.0
		if target > 0 {
			perRow = target / float64(totalCount)
		}
		if wrCount > 0 {
			if err := st.Exec(
				fmt.Sprintf("UPDATE wholesale_retail SET %s = ? WHERE data_year = ? AND data_month = ?", field),
				perRow, year, month,
			); err != nil {
				return err
			}
		}
		if acCount > 0 {
			if err := st.Exec(
				fmt.Sprintf("UPDATE accommodation_catering SET %s = ? WHERE data_year = ? AND data_month = ?", field),
				perRow, year, month,
			); err != nil {
				return err
			}
		}
		return nil
	}

	factor := target / totalSum
	if wrCount > 0 {
		if err := st.Exec(
			fmt.Sprintf("UPDATE wholesale_retail SET %s = %s * ? WHERE data_year = ? AND data_month = ?", field, field),
			factor, year, month,
		); err != nil {
			return err
		}
	}
	if acCount > 0 {
		if err := st.Exec(
			fmt.Sprintf("UPDATE accommodation_catering SET %s = %s * ? WHERE data_year = ? AND data_month = ?", field, field),
			factor, year, month,
		); err != nil {
			return err
		}
	}
	return nil
}

func scaleWRField(st *store.Store, year, month int, industryType string, flagField string, field string, currentSum float64, count int, target float64) error {
	if count == 0 {
		return fmt.Errorf("没有可调整数据")
	}
	where := "data_year = ? AND data_month = ?"
	args := []interface{}{year, month}

	if industryType != "" {
		where += " AND industry_type = ?"
		args = append(args, industryType)
	}
	if flagField != "" {
		where += fmt.Sprintf(" AND %s = 1", flagField)
	}

	if currentSum == 0 {
		perRow := 0.0
		if target > 0 {
			perRow = target / float64(count)
		}
		argsWithValue := append([]interface{}{perRow}, args...)
		return st.Exec(
			fmt.Sprintf("UPDATE wholesale_retail SET %s = ? WHERE %s", field, where),
			argsWithValue...,
		)
	}

	factor := target / currentSum
	argsWithFactor := append([]interface{}{factor}, args...)
	return st.Exec(
		fmt.Sprintf("UPDATE wholesale_retail SET %s = %s * ? WHERE %s", field, field, where),
		argsWithFactor...,
	)
}

func scaleACField(st *store.Store, year, month int, industryType string, field string, currentSum float64, count int, target float64) error {
	if count == 0 {
		return fmt.Errorf("没有可调整数据")
	}

	where := "data_year = ? AND data_month = ?"
	args := []interface{}{year, month}

	if industryType != "" {
		where += " AND industry_type = ?"
		args = append(args, industryType)
	}

	if currentSum == 0 {
		perRow := 0.0
		if target > 0 {
			perRow = target / float64(count)
		}
		argsWithValue := append([]interface{}{perRow}, args...)
		return st.Exec(
			fmt.Sprintf("UPDATE accommodation_catering SET %s = ? WHERE %s", field, where),
			argsWithValue...,
		)
	}

	factor := target / currentSum
	argsWithFactor := append([]interface{}{factor}, args...)
	return st.Exec(
		fmt.Sprintf("UPDATE accommodation_catering SET %s = %s * ? WHERE %s", field, field, where),
		argsWithFactor...,
	)
}

