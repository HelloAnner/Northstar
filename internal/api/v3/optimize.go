package v3

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"northstar/internal/calculator"
	"northstar/internal/model"
	"northstar/internal/store"
)

type OptimizeRequest struct {
	Targets map[string]float64 `json:"targets"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var randFloat64 = func() float64 {
	return rand.Float64()
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

	// 统一使用整数目标
	for k, v := range req.Targets {
		req.Targets[k] = math.Round(v)
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
	roundIndicatorGroupsInPlace(groups)

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

	return scaleAcrossWRAndACDerivedRetail(st, year, month, "retail_current_month", "food_current_month", "goods_current_month", 0, 0, 0, 0, target)
}

func adjustLimitAboveMonthRate(st *store.Store, year, month int, targetRate float64) error {
	lastYearSumWR, _, err := sumAndCountWR(st, year, month, "", "", "retail_last_year_month")
	if err != nil {
		return err
	}
	lastYearSumAC, _, err := sumAndCountACDerivedRetailMonth(st, year, month, "", true)
	if err != nil {
		return err
	}

	desired := lastYearSumWR + lastYearSumAC
	desired = desired * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}
	return scaleAcrossWRAndACDerivedRetail(st, year, month, "retail_current_month", "food_current_month", "goods_current_month", 0, 0, 0, 0, desired)
}

func adjustLimitAboveCumulativeValue(st *store.Store, year, month int, target float64) error {
	if target < 0 {
		target = 0
	}
	return scaleAcrossWRAndACDerivedRetail(st, year, month, "retail_current_cumulative", "food_current_cumulative", "goods_current_cumulative", 0, 0, 0, 0, target)
}

func adjustLimitAboveCumulativeRate(st *store.Store, year, month int, targetRate float64) error {
	lastYearSumWR, _, err := sumAndCountWR(st, year, month, "", "", "retail_last_year_cumulative")
	if err != nil {
		return err
	}
	lastYearSumAC, _, err := sumAndCountACDerivedRetailCumulative(st, year, month, "", true)
	if err != nil {
		return err
	}

	lastYearSum := lastYearSumWR + lastYearSumAC
	desired := lastYearSum * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}
	return scaleAcrossWRAndACDerivedRetail(st, year, month, "retail_current_cumulative", "food_current_cumulative", "goods_current_cumulative", 0, 0, 0, 0, desired)
}

func adjustWRSpecialRate(st *store.Store, year, month int, flagField string, targetRate float64) error {
	lastYearSum, _, err := sumAndCountWR(st, year, month, "", flagField, "retail_last_year_month")
	if err != nil {
		return err
	}

	desired := lastYearSum * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}

	return scaleWRField(st, year, month, "", flagField, "retail_current_month", 0, 0, desired)
}

func adjustWRIndustryRate(st *store.Store, year, month int, industryType, currentField, lastYearField string, targetRate float64) error {
	lastYearSum, _, err := sumAndCountWR(st, year, month, industryType, "", lastYearField)
	if err != nil {
		return err
	}

	desired := lastYearSum * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}

	return scaleWRField(st, year, month, industryType, "", currentField, 0, 0, desired)
}

func adjustACIndustryRate(st *store.Store, year, month int, industryType, currentField, lastYearField string, targetRate float64) error {
	lastYearSum, _, err := sumAndCountAC(st, year, month, industryType, lastYearField)
	if err != nil {
		return err
	}

	desired := lastYearSum * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}

	return scaleACField(st, year, month, industryType, currentField, 0, 0, desired)
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
	return scaleAcrossWRAndACDerivedRetail(st, year, month, "retail_current_cumulative", "food_current_cumulative", "goods_current_cumulative", 0, 0, 0, 0, desiredLimitAbove)
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

	retailLastYearCumWR, _, err := sumAndCountWR(st, year, month, "", "", "retail_last_year_cumulative")
	if err != nil {
		return err
	}
	retailLastYearCumAC, _, err := sumAndCountACDerivedRetailCumulative(st, year, month, "", true)
	if err != nil {
		return err
	}
	retailLastYearCumulativeSum := retailLastYearCumWR + retailLastYearCumAC

	targetFraction := targetRate / 100
	desiredLimitAbove := retailLastYearCumulativeSum*(1+targetFraction) + limitBelowLastYear*(targetFraction-microRate/100)
	if desiredLimitAbove < 0 {
		desiredLimitAbove = 0
	}
	return scaleAcrossWRAndACDerivedRetail(st, year, month, "retail_current_cumulative", "food_current_cumulative", "goods_current_cumulative", 0, 0, 0, 0, desiredLimitAbove)
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

func sumAndCountACDerivedRetailMonth(st *store.Store, year, month int, industryType string, lastYear bool) (float64, int, error) {
	where := "data_year = ? AND data_month = ?"
	args := []interface{}{year, month}
	if industryType != "" {
		where += " AND industry_type = ?"
		args = append(args, industryType)
	}

	expr := "food_current_month + goods_current_month"
	if lastYear {
		expr = "food_last_year_month + goods_last_year_month"
	}

	query := fmt.Sprintf("SELECT COALESCE(SUM(%s), 0), COUNT(1) FROM accommodation_catering WHERE %s", expr, where)
	var sum float64
	var count int
	if err := st.QueryRow(query, args...).Scan(&sum, &count); err != nil {
		return 0, 0, err
	}
	return sum, count, nil
}

func sumAndCountACDerivedRetailCumulative(st *store.Store, year, month int, industryType string, lastYear bool) (float64, int, error) {
	where := "data_year = ? AND data_month = ?"
	args := []interface{}{year, month}
	if industryType != "" {
		where += " AND industry_type = ?"
		args = append(args, industryType)
	}

	expr := "food_current_cumulative + goods_current_cumulative"
	if lastYear {
		expr = "food_last_year_cumulative + goods_last_year_cumulative"
	}

	query := fmt.Sprintf("SELECT COALESCE(SUM(%s), 0), COUNT(1) FROM accommodation_catering WHERE %s", expr, where)
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
			perRow = math.Round(target / float64(totalCount))
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
			fmt.Sprintf("UPDATE wholesale_retail SET %s = ROUND(%s * ?, 0) WHERE data_year = ? AND data_month = ?", field, field),
			factor, year, month,
		); err != nil {
			return err
		}
	}
	if acCount > 0 {
		if err := st.Exec(
			fmt.Sprintf("UPDATE accommodation_catering SET %s = ROUND(%s * ?, 0) WHERE data_year = ? AND data_month = ?", field, field),
			factor, year, month,
		); err != nil {
			return err
		}
	}
	return nil
}

func scaleAcrossWRAndACDerivedRetail(st *store.Store, year, month int, wrField string, acFoodField string, acGoodsField string, wrSum float64, wrCount int, acSum float64, acCount int, target float64) error {
	wrRows, err := loadWRRowsForAdjust(st, year, month, "", "")
	if err != nil {
		return err
	}
	acRows, err := loadACRowsForAdjust(st, year, month, "")
	if err != nil {
		return err
	}
	if len(wrRows)+len(acRows) == 0 {
		return fmt.Errorf("没有可调整数据")
	}

	bases := make([]float64, 0, len(wrRows)+len(acRows))
	for _, r := range wrRows {
		bases = append(bases, pickWRFieldValue(r, wrField))
	}
	for _, r := range acRows {
		bases = append(bases, pickACFieldValue(r, acFoodField)+pickACFieldValue(r, acGoodsField))
	}

	values := randomizeAllocations(target, bases)
	if len(wrRows) > 0 {
		if err := updateWRFieldValues(st, wrField, wrRows, values[:len(wrRows)]); err != nil {
			return err
		}
	}
	if len(acRows) > 0 {
		return updateACDerivedRetailValues(st, acFoodField, acGoodsField, acRows, values[len(wrRows):])
	}
	return nil
}

func scaleWRField(st *store.Store, year, month int, industryType string, flagField string, field string, currentSum float64, count int, target float64) error {
	rows, err := loadWRRowsForAdjust(st, year, month, industryType, flagField)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("没有可调整数据")
	}
	bases := make([]float64, len(rows))
	for i, r := range rows {
		bases[i] = pickWRFieldValue(r, field)
	}
	values := randomizeAllocations(target, bases)
	return updateWRFieldValues(st, field, rows, values)
}

func scaleACField(st *store.Store, year, month int, industryType string, field string, currentSum float64, count int, target float64) error {
	rows, err := loadACRowsForAdjust(st, year, month, industryType)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("没有可调整数据")
	}
	bases := make([]float64, len(rows))
	for i, r := range rows {
		bases[i] = pickACFieldValue(r, field)
	}
	values := randomizeAllocations(target, bases)
	return updateACFieldValues(st, field, rows, values)
}

func loadWRRowsForAdjust(st *store.Store, year, month int, industryType, flagField string) ([]*model.WholesaleRetail, error) {
	opts := store.WRQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	}
	if industryType != "" {
		opts.IndustryType = &industryType
	}
	if flagField == "is_small_micro" {
		v := 1
		opts.IsSmallMicro = &v
	}
	if flagField == "is_eat_wear_use" {
		v := 1
		opts.IsEatWearUse = &v
	}
	return st.GetWRByYearMonth(opts)
}

func loadACRowsForAdjust(st *store.Store, year, month int, industryType string) ([]*model.AccommodationCatering, error) {
	opts := store.ACQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	}
	if industryType != "" {
		opts.IndustryType = &industryType
	}
	return st.GetACByYearMonth(opts)
}

func pickWRFieldValue(r *model.WholesaleRetail, field string) float64 {
	switch field {
	case "retail_current_month":
		return r.RetailCurrentMonth
	case "retail_current_cumulative":
		return r.RetailCurrentCumulative
	case "sales_current_month":
		return r.SalesCurrentMonth
	case "sales_current_cumulative":
		return r.SalesCurrentCumulative
	default:
		return 0
	}
}

func pickACFieldValue(r *model.AccommodationCatering, field string) float64 {
	switch field {
	case "revenue_current_month":
		return r.RevenueCurrentMonth
	case "revenue_current_cumulative":
		return r.RevenueCurrentCumulative
	case "food_current_month":
		return r.FoodCurrentMonth
	case "food_current_cumulative":
		return r.FoodCurrentCumulative
	case "goods_current_month":
		return r.GoodsCurrentMonth
	case "goods_current_cumulative":
		return r.GoodsCurrentCumulative
	default:
		return 0
	}
}

func updateWRFieldValues(st *store.Store, field string, rows []*model.WholesaleRetail, values []float64) error {
	if len(rows) != len(values) {
		return fmt.Errorf("调整数据长度不一致")
	}
	tx, err := st.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := fmt.Sprintf("UPDATE wholesale_retail SET %s = ? WHERE id = ?", field)
	for i, r := range rows {
		if _, err := tx.Exec(query, values[i], r.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func updateACFieldValues(st *store.Store, field string, rows []*model.AccommodationCatering, values []float64) error {
	if len(rows) != len(values) {
		return fmt.Errorf("调整数据长度不一致")
	}
	tx, err := st.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := fmt.Sprintf("UPDATE accommodation_catering SET %s = ? WHERE id = ?", field)
	for i, r := range rows {
		if _, err := tx.Exec(query, values[i], r.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func updateACDerivedRetailValues(st *store.Store, foodField, goodsField string, rows []*model.AccommodationCatering, totals []float64) error {
	if len(rows) != len(totals) {
		return fmt.Errorf("调整数据长度不一致")
	}
	tx, err := st.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	retailField := acRetailFieldFor(foodField, goodsField)
	query := fmt.Sprintf("UPDATE accommodation_catering SET %s = ?, %s = ?", foodField, goodsField)
	if retailField != "" {
		query = fmt.Sprintf("%s, %s = ?", query, retailField)
	}
	query = fmt.Sprintf("%s WHERE id = ?", query)

	for i, r := range rows {
		foodBase := pickACFieldValue(r, foodField)
		goodsBase := pickACFieldValue(r, goodsField)
		newFood, newGoods := splitFoodGoods(totals[i], foodBase, goodsBase)
		args := []interface{}{newFood, newGoods}
		if retailField != "" {
			args = append(args, newFood+newGoods)
		}
		args = append(args, r.ID)
		if _, err := tx.Exec(query, args...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func acRetailFieldFor(foodField, goodsField string) string {
	if foodField == "food_current_month" && goodsField == "goods_current_month" {
		return "retail_current_month"
	}
	if foodField == "food_last_year_month" && goodsField == "goods_last_year_month" {
		return "retail_last_year_month"
	}
	return ""
}

func randomizeAllocations(target float64, bases []float64) []float64 {
	// 随机权重分配目标值，保证总和可回收
	if target < 0 {
		target = 0
	}
	weights := buildRandomWeights(bases)
	values := allocateByWeights(target, weights)
	adjustAllocationDiff(values, target)
	return values
}

func buildRandomWeights(bases []float64) []float64 {
	weights := make([]float64, len(bases))
	for i, base := range bases {
		w := base
		if w <= 0 {
			w = 1
		}
		weights[i] = w * randomJitter()
	}
	return weights
}

func allocateByWeights(target float64, weights []float64) []float64 {
	sum := sumFloat(weights)
	if sum == 0 {
		sum = float64(len(weights))
	}
	values := make([]float64, len(weights))
	for i, w := range weights {
		values[i] = math.Round(w / sum * target)
	}
	return values
}

func adjustAllocationDiff(values []float64, target float64) {
	diff := int(math.Round(target)) - int(sumFloat(values))
	if diff == 0 {
		return
	}
	step := 1
	if diff < 0 {
		step = -1
	}
	for diff != 0 {
		for i := range values {
			if diff == 0 {
				return
			}
			if step < 0 && values[i] <= 0 {
				continue
			}
			values[i] += float64(step)
			diff -= step
		}
	}
}

func randomJitter() float64 {
	return 0.7 + randFloat64()*0.6
}

func sumFloat(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

func splitFoodGoods(total, food, goods float64) (float64, float64) {
	if total < 0 {
		total = 0
	}
	baseTotal := food + goods
	ratio := 0.5
	if baseTotal > 0 {
		ratio = food / baseTotal
	}
	ratio = clampFloat(ratio+(randFloat64()-0.5)*0.2, 0.1, 0.9)
	newFood := math.Round(total * ratio)
	newGoods := total - newFood
	return newFood, newGoods
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
