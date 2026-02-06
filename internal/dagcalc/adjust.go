package dagcalc

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"northstar/internal/model"
	"northstar/internal/store"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var randFloat64 = func() float64 {
	return rand.Float64()
}

// ErrNoAdjustData 没有可调整数据
var ErrNoAdjustData = errors.New("没有可调整数据")

// SetRandFloat64ForTest 设置随机函数（仅用于测试）
func SetRandFloat64ForTest(fn func() float64) func() {
	prev := randFloat64
	randFloat64 = fn
	return func() {
		randFloat64 = prev
	}
}

// ApplyIndicatorTarget 反向调整指标目标值
func ApplyIndicatorTarget(st *store.Store, year, month int, id string, target float64) error {
	if math.IsNaN(target) || math.IsInf(target, 0) {
		return fmt.Errorf("无效目标值: %s", id)
	}

	var err error
	switch id {
	case "limitAbove_month_value":
		err = adjustLimitAboveMonthValue(st, year, month, target)
	case "limitAbove_month_rate":
		err = adjustLimitAboveMonthRate(st, year, month, target)
	case "limitAbove_cumulative_value":
		err = adjustLimitAboveCumulativeValue(st, year, month, target)
	case "limitAbove_cumulative_rate":
		err = adjustLimitAboveCumulativeRate(st, year, month, target)
	case "eatWearUse_month_rate":
		err = adjustWRSpecialRate(st, year, month, "is_eat_wear_use", target)
	case "microSmall_month_rate":
		err = adjustWRSpecialRate(st, year, month, "is_small_micro", target)
	case "wholesale_month_rate":
		err = adjustWRIndustryRate(st, year, month, "wholesale", "sales_current_month", "sales_last_year_month", target)
	case "wholesale_cumulative_rate":
		err = adjustWRCumulativeRate(st, year, month, "wholesale", "sales_current_month", "sales_prev_cumulative", "sales_last_year_cumulative", target)
	case "retail_month_rate":
		err = adjustWRIndustryRate(st, year, month, "retail", "sales_current_month", "sales_last_year_month", target)
	case "retail_cumulative_rate":
		err = adjustWRCumulativeRate(st, year, month, "retail", "sales_current_month", "sales_prev_cumulative", "sales_last_year_cumulative", target)
	case "accommodation_month_rate":
		err = adjustACIndustryRate(st, year, month, "accommodation", "revenue_current_month", "revenue_last_year_month", target)
	case "accommodation_cumulative_rate":
		err = adjustACCumulativeRate(st, year, month, "accommodation", "revenue_current_month", "revenue_prev_cumulative", "revenue_last_year_cumulative", target)
	case "catering_month_rate":
		err = adjustACIndustryRate(st, year, month, "catering", "revenue_current_month", "revenue_last_year_month", target)
	case "catering_cumulative_rate":
		err = adjustACCumulativeRate(st, year, month, "catering", "revenue_current_month", "revenue_prev_cumulative", "revenue_last_year_cumulative", target)
	case "totalSocial_cumulative_value":
		err = adjustTotalSocialCumulativeValue(st, year, month, target)
	case "totalSocial_cumulative_rate":
		err = adjustTotalSocialCumulativeRate(st, year, month, target)
	default:
		return fmt.Errorf("不支持的指标: %s", id)
	}

	if err != nil && errors.Is(err, ErrNoAdjustData) {
		return nil
	}
	return err
}

func adjustLimitAboveMonthValue(st *store.Store, year, month int, target float64) error {
	if target < 0 {
		target = 0
	}

	return scaleAcrossWRAndACDerivedRetail(st, year, month, "retail_current_month", "food_current_month", "goods_current_month", target)
}

func adjustLimitAboveMonthRate(st *store.Store, year, month int, targetRate float64) error {
	lastYearSumWR, _, err := sumAndCountWR(st, year, month, "", "", "retail_last_year_month")
	if err != nil {
		return err
	}
	lastYearSumAC, _, err := sumAndCountACDerivedRetailLastYearMonth(st, year, month, "")
	if err != nil {
		return err
	}

	desired := (lastYearSumWR + lastYearSumAC) * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}
	return scaleAcrossWRAndACDerivedRetail(st, year, month, "retail_current_month", "food_current_month", "goods_current_month", desired)
}

func adjustLimitAboveCumulativeValue(st *store.Store, year, month int, target float64) error {
	if target < 0 {
		target = 0
	}

	return scaleAcrossWRAndACDerivedRetailByCumulative(st, year, month, "retail_current_month", "retail_current_cumulative", "food_current_month", "food_current_cumulative", "goods_current_month", "goods_current_cumulative", target)
}

func adjustLimitAboveCumulativeRate(st *store.Store, year, month int, targetRate float64) error {
	lastYearSumWR, _, err := sumAndCountWR(st, year, month, "", "", "retail_last_year_cumulative")
	if err != nil {
		return err
	}
	lastYearSumAC, _, err := sumAndCountACDerivedRetailLastYearCumulative(st, year, month, "")
	if err != nil {
		return err
	}

	lastYearSum := lastYearSumWR + lastYearSumAC
	desired := lastYearSum * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}
	return scaleAcrossWRAndACDerivedRetailByCumulative(st, year, month, "retail_current_month", "retail_current_cumulative", "food_current_month", "food_current_cumulative", "goods_current_month", "goods_current_cumulative", desired)
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

	return scaleWRField(st, year, month, "", flagField, "retail_current_month", desired)
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

	return scaleWRField(st, year, month, industryType, "", currentField, desired)
}

func adjustWRCumulativeRate(st *store.Store, year, month int, industryType, currentField, prevCumField, lastYearCumField string, targetRate float64) error {
	lastYearSum, _, err := sumAndCountWR(st, year, month, industryType, "", lastYearCumField)
	if err != nil {
		return err
	}

	desiredCum := lastYearSum * (1 + targetRate/100)
	if desiredCum < 0 {
		desiredCum = 0
	}

	return scaleWRFieldByCumulative(st, year, month, industryType, currentField, prevCumField, cumulativeFieldFor(currentField), desiredCum)
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

	return scaleACField(st, year, month, industryType, currentField, desired)
}

func adjustACCumulativeRate(st *store.Store, year, month int, industryType, currentField, prevCumField, lastYearCumField string, targetRate float64) error {
	lastYearSum, _, err := sumAndCountAC(st, year, month, industryType, lastYearCumField)
	if err != nil {
		return err
	}

	desiredCum := lastYearSum * (1 + targetRate/100)
	if desiredCum < 0 {
		desiredCum = 0
	}

	return scaleACFieldByCumulative(st, year, month, industryType, currentField, prevCumField, cumulativeFieldFor(currentField), desiredCum)
}

func adjustTotalSocialCumulativeValue(st *store.Store, year, month int, target float64) error {
	if target < 0 {
		target = 0
	}

	currentLimitAbove, err := sumLimitAboveCumulative(st, year, month)
	if err != nil {
		return err
	}
	limitBelowEstimate, err := estimateLimitBelowCumulative(st, year, month)
	if err != nil {
		return err
	}
	limitBelowEstimated := limitBelowEstimate.CurrentCumulative
	currentTotal := currentLimitAbove + limitBelowEstimated
	delta := target - currentTotal

	limitAboveTarget, limitBelowTarget := splitDelta(delta, currentLimitAbove, limitBelowEstimated)
	if limitBelowTarget < 0 {
		limitBelowTarget = 0
	}

	if err := updateLimitBelowConfig(st, limitBelowEstimate.GrowthFactor, limitBelowTarget); err != nil {
		return err
	}

	return scaleAcrossWRAndACDerivedRetailByCumulative(st, year, month, "retail_current_month", "retail_current_cumulative", "food_current_month", "food_current_cumulative", "goods_current_month", "goods_current_cumulative", limitAboveTarget)
}

func adjustTotalSocialCumulativeRate(st *store.Store, year, month int, targetRate float64) error {
	currentLimitAbove, err := sumLimitAboveCumulative(st, year, month)
	if err != nil {
		return err
	}
	limitBelowEstimate, err := estimateLimitBelowCumulative(st, year, month)
	if err != nil {
		return err
	}

	retailLastYearCumWR, _, err := sumAndCountWR(st, year, month, "", "", "retail_last_year_cumulative")
	if err != nil {
		return err
	}
	retailLastYearCumAC, _, err := sumAndCountACDerivedRetailLastYearCumulative(st, year, month, "")
	if err != nil {
		return err
	}
	retailLastYearCumulativeSum := retailLastYearCumWR + retailLastYearCumAC

	targetFraction := targetRate / 100
	desiredTotal := (retailLastYearCumulativeSum + limitBelowEstimate.LastYearCumulative) * (1 + targetFraction)
	limitBelowEstimated := limitBelowEstimate.CurrentCumulative
	currentTotal := currentLimitAbove + limitBelowEstimated
	delta := desiredTotal - currentTotal

	limitAboveTarget, limitBelowTarget := splitDelta(delta, currentLimitAbove, limitBelowEstimated)
	if limitBelowTarget < 0 {
		limitBelowTarget = 0
	}

	if err := updateLimitBelowConfig(st, limitBelowEstimate.GrowthFactor, limitBelowTarget); err != nil {
		return err
	}

	return scaleAcrossWRAndACDerivedRetailByCumulative(st, year, month, "retail_current_month", "retail_current_cumulative", "food_current_month", "food_current_cumulative", "goods_current_month", "goods_current_cumulative", limitAboveTarget)
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

func sumLimitAboveCumulative(st *store.Store, year, month int) (float64, error) {
	wrSum, _, err := sumAndCountWR(st, year, month, "", "", "retail_current_cumulative")
	if err != nil {
		return 0, err
	}
	acSum, _, err := sumAndCountACDerivedRetailCumulative(st, year, month, "", true)
	if err != nil {
		return 0, err
	}
	return wrSum + acSum, nil
}

func splitDelta(delta, currentLimitAbove, currentLimitBelow float64) (float64, float64) {
	half := delta / 2
	return clampTarget(currentLimitAbove + half), clampTarget(currentLimitBelow + half)
}

func updateLimitBelowConfig(st *store.Store, growthFactor float64, targetLimitBelow float64) error {
	denom := growthFactor
	if denom <= 0 {
		return st.SetConfigFloat("last_year_limit_below_cumulative", 0)
	}
	value := targetLimitBelow / denom
	if value < 0 {
		value = 0
	}
	return st.SetConfigFloat("last_year_limit_below_cumulative", value)
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

func sumAndCountACDerivedRetailMonth(st *store.Store, year, month int, industryType string, hasRows bool) (float64, int, error) {
	rows, err := loadACRowsForAdjust(st, year, month, industryType)
	if err != nil {
		return 0, 0, err
	}
	if !hasRows {
		return 0, len(rows), nil
	}

	sum := 0.0
	for _, r := range rows {
		sum += r.FoodCurrentMonth + r.GoodsCurrentMonth
	}
	return sum, len(rows), nil
}

func sumAndCountACDerivedRetailLastYearMonth(st *store.Store, year, month int, industryType string) (float64, int, error) {
	rows, err := loadACRowsForAdjust(st, year, month, industryType)
	if err != nil {
		return 0, 0, err
	}

	sum := 0.0
	for _, r := range rows {
		sum += r.FoodLastYearMonth + r.GoodsLastYearMonth
	}
	return sum, len(rows), nil
}

func sumAndCountACDerivedRetailCumulative(st *store.Store, year, month int, industryType string, hasRows bool) (float64, int, error) {
	rows, err := loadACRowsForAdjust(st, year, month, industryType)
	if err != nil {
		return 0, 0, err
	}
	if !hasRows {
		return 0, len(rows), nil
	}

	sum := 0.0
	for _, r := range rows {
		sum += r.FoodCurrentCumulative + r.GoodsCurrentCumulative
	}
	return sum, len(rows), nil
}

func sumAndCountACDerivedRetailLastYearCumulative(st *store.Store, year, month int, industryType string) (float64, int, error) {
	rows, err := loadACRowsForAdjust(st, year, month, industryType)
	if err != nil {
		return 0, 0, err
	}

	sum := 0.0
	for _, r := range rows {
		sum += r.FoodLastYearCumulative + r.GoodsLastYearCumulative
	}
	return sum, len(rows), nil
}

func scaleAcrossWRAndACDerivedRetail(st *store.Store, year, month int, wrField string, acFoodField string, acGoodsField string, target float64) error {
	wrRows, err := loadWRRowsForAdjust(st, year, month, "", "")
	if err != nil {
		return err
	}
	acRows, err := loadACRowsForAdjust(st, year, month, "")
	if err != nil {
		return err
	}
	if len(wrRows)+len(acRows) == 0 {
		return ErrNoAdjustData
	}

	bases := make([]float64, 0, len(wrRows)+len(acRows))
	scales := make([]int, 0, len(wrRows)+len(acRows))
	for _, r := range wrRows {
		bases = append(bases, pickWRFieldValue(r, wrField))
		scales = append(scales, r.CompanyScale)
	}
	for _, r := range acRows {
		bases = append(bases, pickACFieldValue(r, acFoodField)+pickACFieldValue(r, acGoodsField))
		scales = append(scales, r.CompanyScale)
	}

	values := randomizeAllocations(target, bases, scales)
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

func scaleAcrossWRAndACDerivedRetailByCumulative(
	st *store.Store,
	year int,
	month int,
	wrCurrentField string,
	wrCumField string,
	acFoodCurrent string,
	acFoodCum string,
	acGoodsCurrent string,
	acGoodsCum string,
	targetCum float64,
) error {
	wrRows, err := loadWRRowsForAdjust(st, year, month, "", "")
	if err != nil {
		return err
	}
	acRows, err := loadACRowsForAdjust(st, year, month, "")
	if err != nil {
		return err
	}
	if len(wrRows)+len(acRows) == 0 {
		return ErrNoAdjustData
	}

	prevSum := 0.0
	bases := make([]float64, 0, len(wrRows)+len(acRows))
	scales := make([]int, 0, len(wrRows)+len(acRows))
	for _, r := range wrRows {
		prevSum += pickWRFieldValue(r, wrCumField) - pickWRFieldValue(r, wrCurrentField)
		bases = append(bases, pickWRFieldValue(r, wrCurrentField))
		scales = append(scales, r.CompanyScale)
	}
	for _, r := range acRows {
		prevSum += pickACFieldValue(r, acFoodCum) - pickACFieldValue(r, acFoodCurrent)
		prevSum += pickACFieldValue(r, acGoodsCum) - pickACFieldValue(r, acGoodsCurrent)
		bases = append(bases, pickACFieldValue(r, acFoodCurrent)+pickACFieldValue(r, acGoodsCurrent))
		scales = append(scales, r.CompanyScale)
	}

	desiredCurrentSum := targetCum - prevSum
	if desiredCurrentSum < 0 {
		desiredCurrentSum = 0
	}

	values := randomizeAllocations(desiredCurrentSum, bases, scales)
	if len(wrRows) > 0 {
		if err := updateWRFieldValuesWithCumulative(st, wrCurrentField, wrCumField, wrRows, values[:len(wrRows)]); err != nil {
			return err
		}
	}
	if len(acRows) > 0 {
		if err := updateACDerivedRetailValuesWithCumulative(st, acFoodCurrent, acGoodsCurrent, acFoodCum, acGoodsCum, acRows, values[len(wrRows):]); err != nil {
			return err
		}
	}
	return nil
}

func scaleWRField(st *store.Store, year, month int, industryType string, flagField string, field string, target float64) error {
	rows, err := loadWRRowsForAdjust(st, year, month, industryType, flagField)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrNoAdjustData
	}
	bases := make([]float64, len(rows))
	scales := make([]int, len(rows))
	for i, r := range rows {
		bases[i] = pickWRFieldValue(r, field)
		scales[i] = r.CompanyScale
	}
	values := randomizeAllocations(target, bases, scales)
	return updateWRFieldValues(st, field, rows, values)
}

func scaleWRFieldByCumulative(st *store.Store, year, month int, industryType string, currentField string, prevCumField string, cumField string, targetCum float64) error {
	rows, err := loadWRRowsForAdjust(st, year, month, industryType, "")
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrNoAdjustData
	}

	prevSum := 0.0
	bases := make([]float64, len(rows))
	scales := make([]int, len(rows))
	for i, r := range rows {
		prevSum += pickWRFieldValue(r, cumField) - pickWRFieldValue(r, currentField)
		bases[i] = pickWRFieldValue(r, currentField)
		scales[i] = r.CompanyScale
	}

	desiredCurrentSum := targetCum - prevSum
	if desiredCurrentSum < 0 {
		desiredCurrentSum = 0
	}
	values := randomizeAllocations(desiredCurrentSum, bases, scales)
	return updateWRFieldValuesWithCumulative(st, currentField, cumField, rows, values)
}

func scaleACField(st *store.Store, year, month int, industryType string, field string, target float64) error {
	rows, err := loadACRowsForAdjust(st, year, month, industryType)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrNoAdjustData
	}
	bases := make([]float64, len(rows))
	scales := make([]int, len(rows))
	for i, r := range rows {
		bases[i] = pickACFieldValue(r, field)
		scales[i] = r.CompanyScale
	}
	values := randomizeAllocations(target, bases, scales)
	return updateACFieldValues(st, field, rows, values)
}

func scaleACFieldByCumulative(st *store.Store, year, month int, industryType string, currentField string, prevCumField string, cumField string, targetCum float64) error {
	rows, err := loadACRowsForAdjust(st, year, month, industryType)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrNoAdjustData
	}

	prevSum := 0.0
	bases := make([]float64, len(rows))
	scales := make([]int, len(rows))
	for i, r := range rows {
		prevSum += pickACFieldValue(r, cumField) - pickACFieldValue(r, currentField)
		bases[i] = pickACFieldValue(r, currentField)
		scales[i] = r.CompanyScale
	}

	desiredCurrentSum := targetCum - prevSum
	if desiredCurrentSum < 0 {
		desiredCurrentSum = 0
	}
	values := randomizeAllocations(desiredCurrentSum, bases, scales)
	return updateACFieldValuesWithCumulative(st, currentField, cumField, rows, values)
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
	case "retail_prev_cumulative":
		return r.RetailPrevCumulative
	case "retail_last_year_cumulative":
		return r.RetailLastYearCumulative
	case "sales_current_month":
		return r.SalesCurrentMonth
	case "sales_current_cumulative":
		return r.SalesCurrentCumulative
	case "sales_prev_cumulative":
		return r.SalesPrevCumulative
	case "sales_last_year_cumulative":
		return r.SalesLastYearCumulative
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
	case "revenue_prev_cumulative":
		return r.RevenuePrevCumulative
	case "revenue_last_year_cumulative":
		return r.RevenueLastYearCumulative
	case "food_current_month":
		return r.FoodCurrentMonth
	case "food_current_cumulative":
		return r.FoodCurrentCumulative
	case "food_prev_cumulative":
		return r.FoodPrevCumulative
	case "food_last_year_cumulative":
		return r.FoodLastYearCumulative
	case "goods_current_month":
		return r.GoodsCurrentMonth
	case "goods_current_cumulative":
		return r.GoodsCurrentCumulative
	case "goods_prev_cumulative":
		return r.GoodsPrevCumulative
	case "goods_last_year_cumulative":
		return r.GoodsLastYearCumulative
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

func updateWRFieldValuesWithCumulative(st *store.Store, currentField, cumulativeField string, rows []*model.WholesaleRetail, values []float64) error {
	if len(rows) != len(values) {
		return fmt.Errorf("调整数据长度不一致")
	}
	tx, err := st.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := fmt.Sprintf("UPDATE wholesale_retail SET %s = ?, %s = ? WHERE id = ?", currentField, cumulativeField)
	for i, r := range rows {
		prev := pickWRFieldValue(r, cumulativeField) - pickWRFieldValue(r, currentField)
		newCurrent := values[i]
		newCumulative := prev + newCurrent
		if _, err := tx.Exec(query, newCurrent, newCumulative, r.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func updateACFieldValuesWithCumulative(st *store.Store, currentField, cumulativeField string, rows []*model.AccommodationCatering, values []float64) error {
	if len(rows) != len(values) {
		return fmt.Errorf("调整数据长度不一致")
	}
	tx, err := st.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := fmt.Sprintf("UPDATE accommodation_catering SET %s = ?, %s = ? WHERE id = ?", currentField, cumulativeField)
	for i, r := range rows {
		prev := pickACFieldValue(r, cumulativeField) - pickACFieldValue(r, currentField)
		newCurrent := values[i]
		newCumulative := prev + newCurrent
		if _, err := tx.Exec(query, newCurrent, newCumulative, r.ID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func updateACDerivedRetailValuesWithCumulative(st *store.Store, foodField, goodsField, foodCumField, goodsCumField string, rows []*model.AccommodationCatering, totals []float64) error {
	if len(rows) != len(totals) {
		return fmt.Errorf("调整数据长度不一致")
	}
	tx, err := st.BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	retailField := acRetailFieldFor(foodField, goodsField)
	query := fmt.Sprintf("UPDATE accommodation_catering SET %s = ?, %s = ?, %s = ?, %s = ?", foodField, goodsField, foodCumField, goodsCumField)
	if retailField != "" {
		query = fmt.Sprintf("%s, %s = ?", query, retailField)
	}
	query = fmt.Sprintf("%s WHERE id = ?", query)

	for i, r := range rows {
		foodBase := pickACFieldValue(r, foodField)
		goodsBase := pickACFieldValue(r, goodsField)
		foodPrev := pickACFieldValue(r, foodCumField) - foodBase
		goodsPrev := pickACFieldValue(r, goodsCumField) - goodsBase
		newFood, newGoods := splitFoodGoods(totals[i], foodBase, goodsBase)
		newFoodCum := foodPrev + newFood
		newGoodsCum := goodsPrev + newGoods
		args := []interface{}{newFood, newGoods, newFoodCum, newGoodsCum}
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

func randomizeAllocations(target float64, bases []float64, scales []int) []float64 {
	if target < 0 {
		target = 0
	}
	weights := buildRandomWeights(bases, scales)
	values := allocateByWeights(target, weights)
	adjustAllocationDiff(values, target)
	return values
}

func buildRandomWeights(bases []float64, scales []int) []float64 {
	weights := make([]float64, len(bases))
	baseSum := sumFloat(bases)
	scaleSum := sumScale(scales)
	fallback := 0.0
	if len(bases) > 0 {
		fallback = 1 / float64(len(bases))
	}
	for i, base := range bases {
		industryShare := fallback
		if baseSum > 0 {
			industryShare = base / baseSum
		}
		scaleShare := fallback
		if scaleSum > 0 {
			scaleShare = float64(normalizeScale(scales, i)) / scaleSum
		}
		w := 0.5*industryShare + 0.5*scaleShare
		if w <= 0 {
			w = fallback
		}
		weights[i] = w * randomJitter()
	}
	return weights
}

func sumScale(scales []int) float64 {
	sum := 0.0
	for _, v := range scales {
		sum += float64(normalizeScaleValue(v))
	}
	return sum
}

func normalizeScale(scales []int, idx int) int {
	if idx < 0 || idx >= len(scales) {
		return 1
	}
	return normalizeScaleValue(scales[idx])
}

func normalizeScaleValue(v int) int {
	if v <= 0 {
		return 1
	}
	return v
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

func clampTarget(target float64) float64 {
	if target < 0 {
		return 0
	}
	return target
}

func cumulativeFieldFor(currentField string) string {
	switch currentField {
	case "sales_current_month":
		return "sales_current_cumulative"
	case "retail_current_month":
		return "retail_current_cumulative"
	case "revenue_current_month":
		return "revenue_current_cumulative"
	case "food_current_month":
		return "food_current_cumulative"
	case "goods_current_month":
		return "goods_current_cumulative"
	default:
		return ""
	}
}
