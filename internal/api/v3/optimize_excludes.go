/**
 * 智能调整排除已修改字段
 *
 * @author Anner
 * @since 12.0
 * Created on 2026/2/1
 */
package v3

import (
	"fmt"
	"math"

	"northstar/internal/model"
	"northstar/internal/store"
)

type fieldExcludes struct {
	wr map[int64]map[string]bool
	ac map[int64]map[string]bool
}

func newFieldExcludes() fieldExcludes {
	return fieldExcludes{
		wr: map[int64]map[string]bool{},
		ac: map[int64]map[string]bool{},
	}
}

func (e *fieldExcludes) AddWR(id int64, fields []string) {
	addExcludeFields(e.wr, id, fields)
}

func (e *fieldExcludes) AddAC(id int64, fields []string) {
	addExcludeFields(e.ac, id, fields)
}

func addExcludeFields(target map[int64]map[string]bool, id int64, fields []string) {
	if len(fields) == 0 {
		return
	}
	if target[id] == nil {
		target[id] = map[string]bool{}
	}
	for _, field := range fields {
		if field != "" {
			target[id][field] = true
		}
	}
}

func (e fieldExcludes) wrExcluded(id int64, field string) bool {
	m := e.wr[id]
	if m == nil {
		return false
	}
	return m[field]
}

func (e fieldExcludes) acExcluded(id int64, field string) bool {
	m := e.ac[id]
	if m == nil {
		return false
	}
	return m[field]
}

func (e fieldExcludes) acDerivedExcluded(id int64, foodField, goodsField string) bool {
	return e.acExcluded(id, foodField) || e.acExcluded(id, goodsField)
}

func applyIndicatorTargetsWithExcludes(st *store.Store, year, month int, targets map[string]float64, excludes fieldExcludes) error {
	ordered := orderTargets(targets)
	for _, item := range ordered {
		target := math.Round(item.Value)
		if err := applyIndicatorTargetWithExcludes(st, year, month, item.ID, target, excludes); err != nil {
			return err
		}
	}
	return recalcDerivedFields(st, year, month)
}

func applyIndicatorTargetWithExcludes(st *store.Store, year, month int, id string, target float64, excludes fieldExcludes) error {
	if math.IsNaN(target) || math.IsInf(target, 0) {
		return fmt.Errorf("无效目标值: %s", id)
	}
	if ok, err := applyLimitAboveWithExcludes(st, year, month, id, target, excludes); ok {
		return err
	}
	if ok, err := applySpecialWithExcludes(st, year, month, id, target, excludes); ok {
		return err
	}
	if ok, err := applyIndustryWithExcludes(st, year, month, id, target, excludes); ok {
		return err
	}
	if ok, err := applyTotalSocialWithExcludes(st, year, month, id, target, excludes); ok {
		return err
	}
	return fmt.Errorf("不支持的指标: %s", id)
}

func applyLimitAboveWithExcludes(st *store.Store, year, month int, id string, target float64, excludes fieldExcludes) (bool, error) {
	switch id {
	case "limitAbove_month_value":
		return true, adjustLimitAboveMonthValueWithExcludes(st, year, month, target, excludes)
	case "limitAbove_month_rate":
		return true, adjustLimitAboveMonthRateWithExcludes(st, year, month, target, excludes)
	case "limitAbove_cumulative_value":
		return true, adjustLimitAboveCumulativeValueWithExcludes(st, year, month, target, excludes)
	case "limitAbove_cumulative_rate":
		return true, adjustLimitAboveCumulativeRateWithExcludes(st, year, month, target, excludes)
	default:
		return false, nil
	}
}

func applySpecialWithExcludes(st *store.Store, year, month int, id string, target float64, excludes fieldExcludes) (bool, error) {
	switch id {
	case "eatWearUse_month_rate":
		return true, adjustWRSpecialRateWithExcludes(st, year, month, "is_eat_wear_use", target, excludes)
	case "microSmall_month_rate":
		return true, adjustWRSpecialRateWithExcludes(st, year, month, "is_small_micro", target, excludes)
	default:
		return false, nil
	}
}

func applyIndustryWithExcludes(st *store.Store, year, month int, id string, target float64, excludes fieldExcludes) (bool, error) {
	switch id {
	case "wholesale_month_rate":
		return true, adjustWRIndustryRateWithExcludes(st, year, month, "wholesale", "sales_current_month", "sales_last_year_month", target, excludes)
	case "wholesale_cumulative_rate":
		return true, adjustWRIndustryRateWithExcludes(st, year, month, "wholesale", "sales_current_cumulative", "sales_last_year_cumulative", target, excludes)
	case "retail_month_rate":
		return true, adjustWRIndustryRateWithExcludes(st, year, month, "retail", "sales_current_month", "sales_last_year_month", target, excludes)
	case "retail_cumulative_rate":
		return true, adjustWRIndustryRateWithExcludes(st, year, month, "retail", "sales_current_cumulative", "sales_last_year_cumulative", target, excludes)
	case "accommodation_month_rate":
		return true, adjustACIndustryRateWithExcludes(st, year, month, "accommodation", "revenue_current_month", "revenue_last_year_month", target, excludes)
	case "accommodation_cumulative_rate":
		return true, adjustACIndustryRateWithExcludes(st, year, month, "accommodation", "revenue_current_cumulative", "revenue_last_year_cumulative", target, excludes)
	case "catering_month_rate":
		return true, adjustACIndustryRateWithExcludes(st, year, month, "catering", "revenue_current_month", "revenue_last_year_month", target, excludes)
	case "catering_cumulative_rate":
		return true, adjustACIndustryRateWithExcludes(st, year, month, "catering", "revenue_current_cumulative", "revenue_last_year_cumulative", target, excludes)
	default:
		return false, nil
	}
}

func applyTotalSocialWithExcludes(st *store.Store, year, month int, id string, target float64, excludes fieldExcludes) (bool, error) {
	switch id {
	case "totalSocial_cumulative_value":
		return true, adjustTotalSocialCumulativeValueWithExcludes(st, year, month, target, excludes)
	case "totalSocial_cumulative_rate":
		return true, adjustTotalSocialCumulativeRateWithExcludes(st, year, month, target, excludes)
	default:
		return false, nil
	}
}

func adjustLimitAboveMonthValueWithExcludes(st *store.Store, year, month int, target float64, excludes fieldExcludes) error {
	if target < 0 {
		target = 0
	}
	return scaleAcrossWRAndACDerivedRetailWithExcludes(
		st, year, month, "retail_current_month", "food_current_month", "goods_current_month", target, excludes,
	)
}

func adjustLimitAboveMonthRateWithExcludes(st *store.Store, year, month int, targetRate float64, excludes fieldExcludes) error {
	lastYearSumWR, _, err := sumAndCountWR(st, year, month, "", "", "retail_last_year_month")
	if err != nil {
		return err
	}
	lastYearSumAC, _, err := sumAndCountACDerivedRetailMonth(st, year, month, "", true)
	if err != nil {
		return err
	}
	desired := (lastYearSumWR + lastYearSumAC) * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}
	return scaleAcrossWRAndACDerivedRetailWithExcludes(
		st, year, month, "retail_current_month", "food_current_month", "goods_current_month", desired, excludes,
	)
}

func adjustLimitAboveCumulativeValueWithExcludes(st *store.Store, year, month int, target float64, excludes fieldExcludes) error {
	if target < 0 {
		target = 0
	}
	return scaleAcrossWRAndACDerivedRetailWithExcludes(
		st, year, month, "retail_current_cumulative", "food_current_cumulative", "goods_current_cumulative", target, excludes,
	)
}

func adjustLimitAboveCumulativeRateWithExcludes(st *store.Store, year, month int, targetRate float64, excludes fieldExcludes) error {
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
	return scaleAcrossWRAndACDerivedRetailWithExcludes(
		st, year, month, "retail_current_cumulative", "food_current_cumulative", "goods_current_cumulative", desired, excludes,
	)
}

func adjustWRSpecialRateWithExcludes(st *store.Store, year, month int, flagField string, targetRate float64, excludes fieldExcludes) error {
	lastYearSum, _, err := sumAndCountWR(st, year, month, "", flagField, "retail_last_year_month")
	if err != nil {
		return err
	}
	desired := lastYearSum * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}
	return scaleWRFieldWithExcludes(st, year, month, "", flagField, "retail_current_month", desired, excludes)
}

func adjustWRIndustryRateWithExcludes(st *store.Store, year, month int, industryType, currentField, lastYearField string, targetRate float64, excludes fieldExcludes) error {
	lastYearSum, _, err := sumAndCountWR(st, year, month, industryType, "", lastYearField)
	if err != nil {
		return err
	}
	desired := lastYearSum * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}
	return scaleWRFieldWithExcludes(st, year, month, industryType, "", currentField, desired, excludes)
}

func adjustACIndustryRateWithExcludes(st *store.Store, year, month int, industryType, currentField, lastYearField string, targetRate float64, excludes fieldExcludes) error {
	lastYearSum, _, err := sumAndCountAC(st, year, month, industryType, lastYearField)
	if err != nil {
		return err
	}
	desired := lastYearSum * (1 + targetRate/100)
	if desired < 0 {
		desired = 0
	}
	return scaleACFieldWithExcludes(st, year, month, industryType, currentField, desired, excludes)
}

func adjustTotalSocialCumulativeValueWithExcludes(st *store.Store, year, month int, target float64, excludes fieldExcludes) error {
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
	return scaleAcrossWRAndACDerivedRetailWithExcludes(
		st, year, month, "retail_current_cumulative", "food_current_cumulative", "goods_current_cumulative", desiredLimitAbove, excludes,
	)
}

func adjustTotalSocialCumulativeRateWithExcludes(st *store.Store, year, month int, targetRate float64, excludes fieldExcludes) error {
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
	targetFraction := targetRate / 100
	desiredLimitAbove := (retailLastYearCumWR+retailLastYearCumAC)*(1+targetFraction) + limitBelowLastYear*(targetFraction-microRate/100)
	if desiredLimitAbove < 0 {
		desiredLimitAbove = 0
	}
	return scaleAcrossWRAndACDerivedRetailWithExcludes(
		st, year, month, "retail_current_cumulative", "food_current_cumulative", "goods_current_cumulative", desiredLimitAbove, excludes,
	)
}

func scaleWRFieldWithExcludes(st *store.Store, year, month int, industryType, flagField, field string, target float64, excludes fieldExcludes) error {
	rows, fixedSum, err := splitWRRowsForAdjust(st, year, month, industryType, flagField, field, excludes)
	if err != nil {
		return err
	}
	return scaleWRRowsWithExcludes(st, field, target, fixedSum, rows)
}

func scaleWRRowsWithExcludes(st *store.Store, field string, target, fixedSum float64, rows []*model.WholesaleRetail) error {
	adjustTarget := clampTarget(target - fixedSum)
	if err := ensureAdjustableRows(len(rows), target, fixedSum); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	values := randomizeAllocations(adjustTarget, wrFieldBases(rows, field))
	return updateWRFieldValues(st, field, rows, values)
}

func scaleACFieldWithExcludes(st *store.Store, year, month int, industryType, field string, target float64, excludes fieldExcludes) error {
	rows, fixedSum, err := splitACRowsForAdjust(st, year, month, industryType, field, excludes)
	if err != nil {
		return err
	}
	return scaleACRowsWithExcludes(st, field, target, fixedSum, rows)
}

func scaleACRowsWithExcludes(st *store.Store, field string, target, fixedSum float64, rows []*model.AccommodationCatering) error {
	adjustTarget := clampTarget(target - fixedSum)
	if err := ensureAdjustableRows(len(rows), target, fixedSum); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	values := randomizeAllocations(adjustTarget, acFieldBases(rows, field))
	return updateACFieldValues(st, field, rows, values)
}

func scaleAcrossWRAndACDerivedRetailWithExcludes(st *store.Store, year, month int, wrField, foodField, goodsField string, target float64, excludes fieldExcludes) error {
	wrRows, wrFixed, err := splitWRRowsForAdjust(st, year, month, "", "", wrField, excludes)
	if err != nil {
		return err
	}
	acRows, acFixed, err := splitACDerivedRetailRowsForAdjust(st, year, month, "", foodField, goodsField, excludes)
	if err != nil {
		return err
	}
	fixedSum := wrFixed + acFixed
	adjustTarget := clampTarget(target - fixedSum)
	if err := ensureAdjustableRows(len(wrRows)+len(acRows), target, fixedSum); err != nil {
		return err
	}
	if len(wrRows)+len(acRows) == 0 {
		return nil
	}
	bases := append(wrFieldBases(wrRows, wrField), acDerivedBases(acRows, foodField, goodsField)...)
	values := randomizeAllocations(adjustTarget, bases)
	if len(wrRows) > 0 {
		if err := updateWRFieldValues(st, wrField, wrRows, values[:len(wrRows)]); err != nil {
			return err
		}
	}
	if len(acRows) > 0 {
		return updateACDerivedRetailValues(st, foodField, goodsField, acRows, values[len(wrRows):])
	}
	return nil
}

func splitWRRowsForAdjust(st *store.Store, year, month int, industryType, flagField, field string, excludes fieldExcludes) ([]*model.WholesaleRetail, float64, error) {
	rows, err := loadWRRowsForAdjust(st, year, month, industryType, flagField)
	if err != nil {
		return nil, 0, err
	}
	adjustable := make([]*model.WholesaleRetail, 0, len(rows))
	fixedSum := 0.0
	for _, row := range rows {
		if excludes.wrExcluded(row.ID, field) {
			fixedSum += pickWRFieldValue(row, field)
			continue
		}
		adjustable = append(adjustable, row)
	}
	return adjustable, fixedSum, nil
}

func splitACRowsForAdjust(st *store.Store, year, month int, industryType, field string, excludes fieldExcludes) ([]*model.AccommodationCatering, float64, error) {
	rows, err := loadACRowsForAdjust(st, year, month, industryType)
	if err != nil {
		return nil, 0, err
	}
	adjustable := make([]*model.AccommodationCatering, 0, len(rows))
	fixedSum := 0.0
	for _, row := range rows {
		if excludes.acExcluded(row.ID, field) {
			fixedSum += pickACFieldValue(row, field)
			continue
		}
		adjustable = append(adjustable, row)
	}
	return adjustable, fixedSum, nil
}

func splitACDerivedRetailRowsForAdjust(st *store.Store, year, month int, industryType, foodField, goodsField string, excludes fieldExcludes) ([]*model.AccommodationCatering, float64, error) {
	rows, err := loadACRowsForAdjust(st, year, month, industryType)
	if err != nil {
		return nil, 0, err
	}
	adjustable := make([]*model.AccommodationCatering, 0, len(rows))
	fixedSum := 0.0
	for _, row := range rows {
		value := pickACFieldValue(row, foodField) + pickACFieldValue(row, goodsField)
		if excludes.acDerivedExcluded(row.ID, foodField, goodsField) {
			fixedSum += value
			continue
		}
		adjustable = append(adjustable, row)
	}
	return adjustable, fixedSum, nil
}

func wrFieldBases(rows []*model.WholesaleRetail, field string) []float64 {
	bases := make([]float64, len(rows))
	for i, row := range rows {
		bases[i] = pickWRFieldValue(row, field)
	}
	return bases
}

func acFieldBases(rows []*model.AccommodationCatering, field string) []float64 {
	bases := make([]float64, len(rows))
	for i, row := range rows {
		bases[i] = pickACFieldValue(row, field)
	}
	return bases
}

func acDerivedBases(rows []*model.AccommodationCatering, foodField, goodsField string) []float64 {
	bases := make([]float64, len(rows))
	for i, row := range rows {
		bases[i] = pickACFieldValue(row, foodField) + pickACFieldValue(row, goodsField)
	}
	return bases
}

func ensureAdjustableRows(count int, target, fixedSum float64) error {
	if count > 0 {
		return nil
	}
	if math.Round(target) == math.Round(fixedSum) {
		return nil
	}
	return fmt.Errorf("没有可调整数据")
}

func clampTarget(target float64) float64 {
	if target < 0 {
		return 0
	}
	return target
}
