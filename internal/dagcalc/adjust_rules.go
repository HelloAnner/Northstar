/**
 * 规则驱动调整辅助
 *
 * @author Anner
 * Created on 2026/3/14
 */

package dagcalc

import (
	"fmt"

	"northstar/internal/model"
	"northstar/internal/rules"
	"northstar/internal/store"
)

// AppliedRule 表示一次实际生效的规则。
type AppliedRule struct {
	RuleID      string  `json:"ruleId"`
	Type        string  `json:"type"`
	IndicatorID string  `json:"indicatorId,omitempty"`
	TriggerID   string  `json:"triggerId,omitempty"`
	EnsureID    string  `json:"ensureId,omitempty"`
	BeforeValue float64 `json:"beforeValue,omitempty"`
	AfterValue  float64 `json:"afterValue,omitempty"`
	BeforeCount int     `json:"beforeCount,omitempty"`
	AfterCount  int     `json:"afterCount,omitempty"`
	TargetValue float64 `json:"targetValue,omitempty"`
}

type adjustRuleScope struct {
	indicatorID string
	ruleSet     *rules.RuleSet
	applied     []AppliedRule
}

func newAdjustRuleScope(indicatorID string, ruleSet *rules.RuleSet) *adjustRuleScope {
	return &adjustRuleScope{
		indicatorID: indicatorID,
		ruleSet:     ruleSet,
		applied:     []AppliedRule{},
	}
}

func (s *adjustRuleScope) applyClamps(target float64) float64 {
	if s == nil || s.ruleSet == nil {
		return target
	}

	next := target
	for _, rule := range s.ruleSet.Clamps {
		clamped, changed := rule.Clamp(s.indicatorID, next)
		if !changed {
			continue
		}
		s.applied = append(s.applied, AppliedRule{
			RuleID:      rule.ID,
			Type:        "clamp_target",
			IndicatorID: s.indicatorID,
			BeforeValue: next,
			AfterValue:  clamped,
		})
		next = clamped
	}
	return next
}

func (s *adjustRuleScope) applyCompensates(st *store.Store, year, month, depth int) error {
	if s == nil || s.ruleSet == nil || depth >= 1 {
		return nil
	}

	for _, rule := range s.ruleSet.Compensates {
		indicators, err := loadIndicatorValueMap(st, year, month)
		if err != nil {
			return err
		}
		need, target := rule.Check(s.indicatorID, indicators)
		if !need {
			continue
		}
		s.applied = append(s.applied, AppliedRule{
			RuleID:      rule.ID,
			Type:        "compensate",
			TriggerID:   rule.TriggerID,
			EnsureID:    rule.EnsureID,
			TargetValue: target,
		})
		applied, err := ApplyIndicatorTarget(st, year, month, rule.EnsureID, target, s.ruleSet, depth+1)
		if err != nil {
			return err
		}
		s.applied = append(s.applied, applied...)
	}
	return nil
}

func (s *adjustRuleScope) filterWRRows(rows []*model.WholesaleRetail) []*model.WholesaleRetail {
	if s == nil || len(rows) == 0 || s.ruleSet == nil {
		return rows
	}

	current := rows
	for _, rule := range s.ruleSet.Filters {
		if rule.IndicatorID != s.indicatorID {
			continue
		}
		companyRows := buildWRCompanyRows(current, s.indicatorID)
		filtered, changed := rule.Apply(s.indicatorID, companyRows)
		if !changed || len(filtered) == 0 {
			continue
		}
		current = selectWRRows(current, filtered)
		s.applied = append(s.applied, AppliedRule{
			RuleID:      rule.ID,
			Type:        "filter_allocation",
			IndicatorID: s.indicatorID,
			BeforeCount: len(companyRows),
			AfterCount:  len(filtered),
		})
	}
	return current
}

func (s *adjustRuleScope) filterACRows(rows []*model.AccommodationCatering) []*model.AccommodationCatering {
	if s == nil || len(rows) == 0 || s.ruleSet == nil {
		return rows
	}

	current := rows
	for _, rule := range s.ruleSet.Filters {
		if rule.IndicatorID != s.indicatorID {
			continue
		}
		companyRows := buildACCompanyRows(current, s.indicatorID)
		filtered, changed := rule.Apply(s.indicatorID, companyRows)
		if !changed || len(filtered) == 0 {
			continue
		}
		current = selectACRows(current, filtered)
		s.applied = append(s.applied, AppliedRule{
			RuleID:      rule.ID,
			Type:        "filter_allocation",
			IndicatorID: s.indicatorID,
			BeforeCount: len(companyRows),
			AfterCount:  len(filtered),
		})
	}
	return current
}

func (s *adjustRuleScope) filterMixedRows(
	wrRows []*model.WholesaleRetail,
	acRows []*model.AccommodationCatering,
) ([]*model.WholesaleRetail, []*model.AccommodationCatering) {
	if s == nil || s.ruleSet == nil {
		return wrRows, acRows
	}

	currentWR := wrRows
	currentAC := acRows
	for _, rule := range s.ruleSet.Filters {
		if rule.IndicatorID != s.indicatorID {
			continue
		}
		companyRows := buildMixedCompanyRows(currentWR, currentAC, s.indicatorID)
		filtered, changed := rule.Apply(s.indicatorID, companyRows)
		if !changed || len(filtered) == 0 {
			continue
		}
		currentWR, currentAC = splitMixedRows(currentWR, currentAC, filtered)
		s.applied = append(s.applied, AppliedRule{
			RuleID:      rule.ID,
			Type:        "filter_allocation",
			IndicatorID: s.indicatorID,
			BeforeCount: len(companyRows),
			AfterCount:  len(filtered),
		})
	}
	return currentWR, currentAC
}

func (s *adjustRuleScope) AppliedRules() []AppliedRule {
	if s == nil {
		return nil
	}
	out := make([]AppliedRule, len(s.applied))
	copy(out, s.applied)
	return out
}

func loadIndicatorValueMap(st *store.Store, year, month int) (map[string]float64, error) {
	groups, err := CalculateIndicators(st, year, month)
	if err != nil {
		return nil, err
	}

	out := make(map[string]float64, 16)
	for _, group := range groups {
		for _, indicator := range group.Indicators {
			out[indicator.ID] = indicator.Value
		}
	}
	return out, nil
}

func buildWRCompanyRows(rows []*model.WholesaleRetail, indicatorID string) []rules.CompanyRow {
	companies := make([]rules.CompanyRow, 0, len(rows))
	for _, row := range rows {
		companies = append(companies, rules.CompanyRow{
			RowID:        row.ID,
			CompanyScale: row.CompanyScale,
			IsSmallMicro: isSmallMicro(row.CompanyScale, row.IsSmallMicro),
			CurrentValue: wrCurrentValueForIndicator(row, indicatorID),
		})
	}
	return companies
}

func buildACCompanyRows(rows []*model.AccommodationCatering, indicatorID string) []rules.CompanyRow {
	companies := make([]rules.CompanyRow, 0, len(rows))
	for _, row := range rows {
		companies = append(companies, rules.CompanyRow{
			RowID:        row.ID,
			CompanyScale: row.CompanyScale,
			IsSmallMicro: isSmallMicro(row.CompanyScale, row.IsSmallMicro),
			CurrentValue: acCurrentValueForIndicator(row, indicatorID),
		})
	}
	return companies
}

func buildMixedCompanyRows(
	wrRows []*model.WholesaleRetail,
	acRows []*model.AccommodationCatering,
	indicatorID string,
) []rules.CompanyRow {
	companies := buildWRCompanyRows(wrRows, indicatorID)
	for _, row := range acRows {
		companies = append(companies, rules.CompanyRow{
			RowID:        -row.ID - 1,
			CompanyScale: row.CompanyScale,
			IsSmallMicro: isSmallMicro(row.CompanyScale, row.IsSmallMicro),
			CurrentValue: acCurrentValueForIndicator(row, indicatorID),
		})
	}
	return companies
}

func selectWRRows(rows []*model.WholesaleRetail, companies []rules.CompanyRow) []*model.WholesaleRetail {
	index := make(map[int64]struct{}, len(companies))
	for _, company := range companies {
		index[company.RowID] = struct{}{}
	}
	filtered := make([]*model.WholesaleRetail, 0, len(companies))
	for _, row := range rows {
		if _, ok := index[row.ID]; ok {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func selectACRows(rows []*model.AccommodationCatering, companies []rules.CompanyRow) []*model.AccommodationCatering {
	index := make(map[int64]struct{}, len(companies))
	for _, company := range companies {
		index[company.RowID] = struct{}{}
	}
	filtered := make([]*model.AccommodationCatering, 0, len(companies))
	for _, row := range rows {
		if _, ok := index[row.ID]; ok {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func splitMixedRows(
	wrRows []*model.WholesaleRetail,
	acRows []*model.AccommodationCatering,
	companies []rules.CompanyRow,
) ([]*model.WholesaleRetail, []*model.AccommodationCatering) {
	wrIndex := map[int64]struct{}{}
	acIndex := map[int64]struct{}{}
	for _, company := range companies {
		if company.RowID >= 0 {
			wrIndex[company.RowID] = struct{}{}
			continue
		}
		acIndex[-company.RowID-1] = struct{}{}
	}

	nextWR := make([]*model.WholesaleRetail, 0, len(wrIndex))
	for _, row := range wrRows {
		if _, ok := wrIndex[row.ID]; ok {
			nextWR = append(nextWR, row)
		}
	}

	nextAC := make([]*model.AccommodationCatering, 0, len(acIndex))
	for _, row := range acRows {
		if _, ok := acIndex[row.ID]; ok {
			nextAC = append(nextAC, row)
		}
	}
	return nextWR, nextAC
}

func isSmallMicro(companyScale int, field int) bool {
	if field == 1 {
		return true
	}
	return companyScale == 3 || companyScale == 4
}

func wrCurrentValueForIndicator(row *model.WholesaleRetail, indicatorID string) float64 {
	if indicatorID == "wholesale_month_rate" || indicatorID == "retail_month_rate" {
		return growthRate(row.SalesCurrentMonth, row.SalesLastYearMonth)
	}
	if indicatorID == "wholesale_cumulative_rate" || indicatorID == "retail_cumulative_rate" {
		return growthRate(row.SalesCurrentCumulative, row.SalesLastYearCumulative)
	}
	if indicatorID == "limitAbove_month_rate" || indicatorID == "eatWearUse_month_rate" ||
		indicatorID == "microSmall_month_rate" {
		return growthRate(row.RetailCurrentMonth, row.RetailLastYearMonth)
	}
	if indicatorID == "limitAbove_cumulative_rate" || indicatorID == "totalSocial_cumulative_rate" {
		return growthRate(row.RetailCurrentCumulative, row.RetailLastYearCumulative)
	}
	if indicatorID == "limitAbove_month_value" {
		return row.RetailCurrentMonth
	}
	if indicatorID == "limitAbove_cumulative_value" || indicatorID == "totalSocial_cumulative_value" {
		return row.RetailCurrentCumulative
	}
	return 0
}

func acCurrentValueForIndicator(row *model.AccommodationCatering, indicatorID string) float64 {
	retailMonth := row.FoodCurrentMonth + row.GoodsCurrentMonth
	retailLastYearMonth := row.FoodLastYearMonth + row.GoodsLastYearMonth
	retailCumulative := row.FoodCurrentCumulative + row.GoodsCurrentCumulative
	retailLastYearCumulative := row.FoodLastYearCumulative + row.GoodsLastYearCumulative
	if indicatorID == "accommodation_month_rate" || indicatorID == "catering_month_rate" {
		return growthRate(row.RevenueCurrentMonth, row.RevenueLastYearMonth)
	}
	if indicatorID == "accommodation_cumulative_rate" || indicatorID == "catering_cumulative_rate" {
		return growthRate(row.RevenueCurrentCumulative, row.RevenueLastYearCumulative)
	}
	if indicatorID == "limitAbove_month_rate" || indicatorID == "eatWearUse_month_rate" ||
		indicatorID == "microSmall_month_rate" {
		return growthRate(retailMonth, retailLastYearMonth)
	}
	if indicatorID == "limitAbove_cumulative_rate" || indicatorID == "totalSocial_cumulative_rate" {
		return growthRate(retailCumulative, retailLastYearCumulative)
	}
	if indicatorID == "limitAbove_month_value" {
		return retailMonth
	}
	if indicatorID == "limitAbove_cumulative_value" || indicatorID == "totalSocial_cumulative_value" {
		return retailCumulative
	}
	return 0
}

func growthRate(current, lastYear float64) float64 {
	if lastYear == 0 {
		return 0
	}
	return (current - lastYear) / lastYear * 100
}

func effectiveTargetForIndicator(applied []AppliedRule, indicatorID string, target float64) float64 {
	next := target
	for _, item := range applied {
		if item.Type != "clamp_target" {
			continue
		}
		if item.IndicatorID != indicatorID {
			continue
		}
		next = item.AfterValue
	}
	return next
}

func formatAppliedRule(item AppliedRule) string {
	return fmt.Sprintf("%s:%s", item.Type, item.RuleID)
}
