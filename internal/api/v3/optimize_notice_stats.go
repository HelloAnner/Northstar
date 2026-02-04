/**
 * 智能调整提示计算
 *
 * @author Anner
 * Created on 2026/2/4
 */

package v3

import (
	"fmt"
	"math"

	"northstar/internal/dagcalc"
	"northstar/internal/store"
)

type indicatorStats struct {
	Eligible   int
	LastYear   float64
	PrevSum    float64
	HasPrevSum bool
}

func precheckIndicatorTarget(st *store.Store, year, month int, id string, target float64, meta indicatorSnapshot) (*OptimizeNotice, error) {
	if math.IsNaN(target) || math.IsInf(target, 0) {
		return noticeInvalidTarget(id, target), nil
	}
	stats, ok, err := loadIndicatorStats(st, year, month, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return noticeUnsupported(id, target), nil
	}
	if needsData(id) && stats.Eligible == 0 {
		return noticeNoData(meta, id, target), nil
	}
	if isRateIndicator(id) && stats.LastYear == 0 {
		return noticeLastYearZero(meta, id, target), nil
	}
	return nil, nil
}

func buildPostApplyNotice(st *store.Store, year, month int, id string, target float64, before indicatorSnapshot, after indicatorSnapshot) (*OptimizeNotice, error) {
	stats, ok, err := loadIndicatorStats(st, year, month, id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return noticeUnsupported(id, target), nil
	}
	if targetReached(target, after.Value) {
		return nil, nil
	}
	if notice := noChangeNotice(before, after, id, target); notice != nil {
		return notice, nil
	}
	if notice := belowMinNotice(stats, after, id, target); notice != nil {
		return notice, nil
	}
	if roundValue(before.Value) == roundValue(after.Value) {
		return noticeNoChange(after, id, target), nil
	}
	return noticeNotReached(after, id, target), nil
}

func targetReached(target float64, after float64) bool {
	return roundValue(target) == roundValue(after)
}

func noChangeNotice(before indicatorSnapshot, after indicatorSnapshot, id string, target float64) *OptimizeNotice {
	if roundValue(before.Value) != roundValue(after.Value) {
		return nil
	}
	if roundValue(target) == roundValue(before.Value) {
		return noticeTargetSame(after, id, target)
	}
	if math.Abs(target-before.Value) < 1 {
		return noticeSmallDelta(after, id, target)
	}
	return nil
}

func belowMinNotice(stats indicatorStats, after indicatorSnapshot, id string, target float64) *OptimizeNotice {
	targetRounded := roundValue(target)
	if minValue, ok := minValueForIndicator(stats, id); ok && targetRounded < roundValue(minValue) {
		return noticeBelowMin(after, id, target, minValue)
	}
	if minRate, ok := minRateForIndicator(stats, id); ok && targetRounded < roundValue(minRate) {
		return noticeBelowMinRate(after, id, target, minRate)
	}
	return nil
}

func loadIndicatorStats(st *store.Store, year, month int, id string) (indicatorStats, bool, error) {
	if stats, ok, err := loadLimitAboveStats(st, year, month, id); ok || err != nil {
		return stats, ok, err
	}
	if stats, ok, err := loadSpecialStats(st, year, month, id); ok || err != nil {
		return stats, ok, err
	}
	if stats, ok, err := loadWRIndustryStats(st, year, month, id); ok || err != nil {
		return stats, ok, err
	}
	if stats, ok, err := loadACIndustryStats(st, year, month, id); ok || err != nil {
		return stats, ok, err
	}
	return loadTotalSocialStats(st, year, month, id)
}

func loadLimitAboveStats(st *store.Store, year, month int, id string) (indicatorStats, bool, error) {
	switch id {
	case "limitAbove_month_value":
		count, err := limitAboveRowCount(st, year, month)
		return indicatorStats{Eligible: count}, true, err
	case "limitAbove_month_rate":
		sum, count, err := sumLimitAboveLastYearMonth(st, year, month)
		return indicatorStats{Eligible: count, LastYear: sum}, true, err
	case "limitAbove_cumulative_value":
		return loadLimitAboveCumulativeValueStats(st, year, month)
	case "limitAbove_cumulative_rate":
		return loadLimitAboveCumulativeRateStats(st, year, month)
	default:
		return indicatorStats{}, false, nil
	}
}

func loadLimitAboveCumulativeValueStats(st *store.Store, year, month int) (indicatorStats, bool, error) {
	count, err := limitAboveRowCount(st, year, month)
	if err != nil {
		return indicatorStats{Eligible: count}, true, err
	}
	prev, err := sumLimitAbovePrevCumulative(st, year, month)
	if err != nil {
		return indicatorStats{Eligible: count}, true, err
	}
	return indicatorStats{Eligible: count, PrevSum: prev, HasPrevSum: true}, true, nil
}

func loadLimitAboveCumulativeRateStats(st *store.Store, year, month int) (indicatorStats, bool, error) {
	sum, count, err := sumLimitAboveLastYearCumulative(st, year, month)
	if err != nil {
		return indicatorStats{Eligible: count}, true, err
	}
	prev, err := sumLimitAbovePrevCumulative(st, year, month)
	if err != nil {
		return indicatorStats{Eligible: count, LastYear: sum}, true, err
	}
	return indicatorStats{Eligible: count, LastYear: sum, PrevSum: prev, HasPrevSum: true}, true, nil
}

func loadSpecialStats(st *store.Store, year, month int, id string) (indicatorStats, bool, error) {
	flag, ok := specialFlagField(id)
	if !ok {
		return indicatorStats{}, false, nil
	}
	sum, count, err := dagcalc.SumAndCountWR(st, year, month, "", flag, "retail_last_year_month")
	return indicatorStats{Eligible: count, LastYear: sum}, true, err
}

func loadWRIndustryStats(st *store.Store, year, month int, id string) (indicatorStats, bool, error) {
	industry, lastYearField, isCum := wrRateSpec(id)
	if industry == "" {
		return indicatorStats{}, false, nil
	}
	sum, count, err := dagcalc.SumAndCountWR(st, year, month, industry, "", lastYearField)
	if err != nil {
		return indicatorStats{}, true, err
	}
	stats := indicatorStats{Eligible: count, LastYear: sum}
	if isCum {
		prev, _, err := sumWRPrevCumulative(st, year, month, industry, "sales_current_cumulative", "sales_current_month")
		stats.PrevSum = prev
		stats.HasPrevSum = true
		return stats, true, err
	}
	return stats, true, nil
}

func loadACIndustryStats(st *store.Store, year, month int, id string) (indicatorStats, bool, error) {
	industry, lastYearField, isCum := acRateSpec(id)
	if industry == "" {
		return indicatorStats{}, false, nil
	}
	sum, count, err := dagcalc.SumAndCountAC(st, year, month, industry, lastYearField)
	if err != nil {
		return indicatorStats{}, true, err
	}
	stats := indicatorStats{Eligible: count, LastYear: sum}
	if isCum {
		prev, _, err := sumACPrevCumulative(st, year, month, industry, "revenue_current_cumulative", "revenue_current_month")
		stats.PrevSum = prev
		stats.HasPrevSum = true
		return stats, true, err
	}
	return stats, true, nil
}

func loadTotalSocialStats(st *store.Store, year, month int, id string) (indicatorStats, bool, error) {
	if id != "totalSocial_cumulative_value" && id != "totalSocial_cumulative_rate" {
		return indicatorStats{}, false, nil
	}
	prev, err := sumLimitAbovePrevCumulative(st, year, month)
	if err != nil {
		return indicatorStats{}, true, err
	}
	stats := indicatorStats{PrevSum: prev, HasPrevSum: true}
	if id == "totalSocial_cumulative_rate" {
		limitAboveLastYear, _, err := sumLimitAboveLastYearCumulative(st, year, month)
		if err != nil {
			return indicatorStats{}, true, err
		}
		limitBelowLastYear, err := st.GetConfigFloat("last_year_limit_below_cumulative")
		if err != nil {
			limitBelowLastYear = 0
		}
		total := limitAboveLastYear + limitBelowLastYear
		stats.LastYear = total
	}
	return stats, true, nil
}

func minValueForIndicator(stats indicatorStats, id string) (float64, bool) {
	switch id {
	case "limitAbove_month_value":
		return 0, true
	case "limitAbove_cumulative_value", "totalSocial_cumulative_value":
		if stats.PrevSum < 0 {
			return 0, true
		}
		return stats.PrevSum, true
	default:
		return 0, false
	}
}

func minRateForIndicator(stats indicatorStats, id string) (float64, bool) {
	if !isRateIndicator(id) || stats.LastYear == 0 {
		return 0, false
	}
	minRate := -100.0
	if stats.HasPrevSum {
		candidate := (stats.PrevSum - stats.LastYear) / stats.LastYear * 100
		if candidate > minRate {
			minRate = candidate
		}
	}
	return minRate, true
}

func sumLimitAboveLastYearMonth(st *store.Store, year, month int) (float64, int, error) {
	wrSum, wrCount, err := dagcalc.SumAndCountWR(st, year, month, "", "", "retail_last_year_month")
	if err != nil {
		return 0, 0, err
	}
	acSum, acCount, err := dagcalc.SumAndCountACDerivedRetailLastYearMonth(st, year, month, "")
	if err != nil {
		return 0, 0, err
	}
	return wrSum + acSum, wrCount + acCount, nil
}

func sumLimitAboveLastYearCumulative(st *store.Store, year, month int) (float64, int, error) {
	wrSum, wrCount, err := dagcalc.SumAndCountWR(st, year, month, "", "", "retail_last_year_cumulative")
	if err != nil {
		return 0, 0, err
	}
	acSum, acCount, err := dagcalc.SumAndCountACDerivedRetailLastYearCumulative(st, year, month, "")
	if err != nil {
		return 0, 0, err
	}
	return wrSum + acSum, wrCount + acCount, nil
}

func limitAboveRowCount(st *store.Store, year, month int) (int, error) {
	_, wrCount, err := dagcalc.SumAndCountWR(st, year, month, "", "", "retail_current_month")
	if err != nil {
		return 0, err
	}
	_, acCount, err := dagcalc.SumAndCountAC(st, year, month, "", "revenue_current_month")
	if err != nil {
		return 0, err
	}
	return wrCount + acCount, nil
}

func sumLimitAbovePrevCumulative(st *store.Store, year, month int) (float64, error) {
	wrSum, _, err := sumWRPrevCumulative(st, year, month, "", "retail_current_cumulative", "retail_current_month")
	if err != nil {
		return 0, err
	}
	acSum, _, err := sumACDerivedPrevCumulative(st, year, month)
	if err != nil {
		return 0, err
	}
	return wrSum + acSum, nil
}

func sumWRPrevCumulative(st *store.Store, year, month int, industryType, cumField, currentField string) (float64, int, error) {
	where := "data_year = ? AND data_month = ?"
	args := []interface{}{year, month}
	if industryType != "" {
		where += " AND industry_type = ?"
		args = append(args, industryType)
	}
	query := fmt.Sprintf("SELECT COALESCE(SUM(%s - %s), 0), COUNT(1) FROM wholesale_retail WHERE %s", cumField, currentField, where)
	var sum float64
	var count int
	if err := st.QueryRow(query, args...).Scan(&sum, &count); err != nil {
		return 0, 0, err
	}
	return sum, count, nil
}

func sumACPrevCumulative(st *store.Store, year, month int, industryType, cumField, currentField string) (float64, int, error) {
	where := "data_year = ? AND data_month = ?"
	args := []interface{}{year, month}
	if industryType != "" {
		where += " AND industry_type = ?"
		args = append(args, industryType)
	}
	query := fmt.Sprintf("SELECT COALESCE(SUM(%s - %s), 0), COUNT(1) FROM accommodation_catering WHERE %s", cumField, currentField, where)
	var sum float64
	var count int
	if err := st.QueryRow(query, args...).Scan(&sum, &count); err != nil {
		return 0, 0, err
	}
	return sum, count, nil
}

func sumACDerivedPrevCumulative(st *store.Store, year, month int) (float64, int, error) {
	query := "SELECT COALESCE(SUM((food_current_cumulative - food_current_month) + (goods_current_cumulative - goods_current_month)), 0), COUNT(1) FROM accommodation_catering WHERE data_year = ? AND data_month = ?"
	var sum float64
	var count int
	if err := st.QueryRow(query, year, month).Scan(&sum, &count); err != nil {
		return 0, 0, err
	}
	return sum, count, nil
}

func specialFlagField(id string) (string, bool) {
	switch id {
	case "eatWearUse_month_rate":
		return "is_eat_wear_use", true
	case "microSmall_month_rate":
		return "is_small_micro", true
	default:
		return "", false
	}
}

func wrRateSpec(id string) (string, string, bool) {
	switch id {
	case "wholesale_month_rate":
		return "wholesale", "sales_last_year_month", false
	case "wholesale_cumulative_rate":
		return "wholesale", "sales_last_year_cumulative", true
	case "retail_month_rate":
		return "retail", "sales_last_year_month", false
	case "retail_cumulative_rate":
		return "retail", "sales_last_year_cumulative", true
	default:
		return "", "", false
	}
}

func acRateSpec(id string) (string, string, bool) {
	switch id {
	case "accommodation_month_rate":
		return "accommodation", "revenue_last_year_month", false
	case "accommodation_cumulative_rate":
		return "accommodation", "revenue_last_year_cumulative", true
	case "catering_month_rate":
		return "catering", "revenue_last_year_month", false
	case "catering_cumulative_rate":
		return "catering", "revenue_last_year_cumulative", true
	default:
		return "", "", false
	}
}

func isRateIndicator(id string) bool {
	switch id {
	case "limitAbove_month_rate",
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
		"totalSocial_cumulative_rate":
		return true
	default:
		return false
	}
}

func needsData(id string) bool {
	switch id {
	case "totalSocial_cumulative_value", "totalSocial_cumulative_rate":
		return false
	default:
		return true
	}
}
