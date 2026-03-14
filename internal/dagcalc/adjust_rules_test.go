/**
 * 规则驱动调整测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

package dagcalc

import (
	"path/filepath"
	"testing"

	"northstar/internal/rules"
	"northstar/internal/store"
)

func TestApplyIndicatorTargetWithRules_ClampTarget(t *testing.T) {
	st := newRuleTestStore(t)
	insertWRRow(t, st, "W1", "wholesale", 1, 100, 100)

	max := 10.0
	rs := &rules.RuleSet{
		Clamps: []*rules.ClampTargetConstraint{
			{ID: "c1", IndicatorID: "wholesale_month_rate", Max: &max},
		},
	}

	applied, err := ApplyIndicatorTarget(st, 2025, 12, "wholesale_month_rate", 50, rs, 0)
	if err != nil {
		t.Fatalf("apply target: %v", err)
	}
	after := mustIndicatorValue(t, st, "wholesale_month_rate")
	if after != 10 {
		t.Fatalf("unexpected indicator value: %.2f", after)
	}
	if len(applied) != 1 || applied[0].Type != "clamp_target" {
		t.Fatalf("unexpected applied rules: %+v", applied)
	}
	if applied[0].BeforeValue != 50 || applied[0].AfterValue != 10 {
		t.Fatalf("unexpected clamp values: %+v", applied[0])
	}
}

func TestApplyIndicatorTargetWithRules_FilterAllocation(t *testing.T) {
	st := newRuleTestStore(t)
	insertWRRow(t, st, "R1", "retail", 1, 110, 100)
	insertWRRow(t, st, "R2", "retail", 1, 80, 100)

	rs := &rules.RuleSet{
		Filters: []*rules.FilterAllocationConstraint{
			{ID: "f1", IndicatorID: "retail_month_rate", Filter: "positive_current"},
		},
	}

	applied, err := ApplyIndicatorTarget(st, 2025, 12, "retail_month_rate", 10, rs, 0)
	if err != nil {
		t.Fatalf("apply target: %v", err)
	}
	if len(applied) != 1 || applied[0].Type != "filter_allocation" {
		t.Fatalf("unexpected applied rules: %+v", applied)
	}
	if applied[0].BeforeCount != 2 || applied[0].AfterCount != 1 {
		t.Fatalf("unexpected filter counts: %+v", applied[0])
	}
	if sales := mustSalesCurrentMonth(t, st, "R2", "retail"); sales != 80 {
		t.Fatalf("expected negative row unchanged, got %.2f", sales)
	}
}

func TestApplyIndicatorTargetWithRules_FilterFallbackToFullSet(t *testing.T) {
	st := newRuleTestStore(t)
	insertWRRow(t, st, "R1", "retail", 1, 80, 100)
	insertWRRow(t, st, "R2", "retail", 1, 70, 100)

	rs := &rules.RuleSet{
		Filters: []*rules.FilterAllocationConstraint{
			{ID: "f1", IndicatorID: "retail_month_rate", Filter: "positive_current"},
		},
	}

	applied, err := ApplyIndicatorTarget(st, 2025, 12, "retail_month_rate", 10, rs, 0)
	if err != nil {
		t.Fatalf("apply target: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("expected no applied filter when fallback happens: %+v", applied)
	}
	if sales := mustSalesCurrentMonth(t, st, "R1", "retail"); sales == 80 {
		t.Fatalf("expected fallback to full allocation")
	}
}

func TestApplyIndicatorTargetWithRules_Compensate(t *testing.T) {
	st := newRuleTestStore(t)
	insertWRRow(t, st, "W1", "wholesale", 1, 100, 100)
	insertWRRow(t, st, "R1", "retail", 1, 100, 100)

	rs := &rules.RuleSet{
		Compensates: []*rules.CompensateConstraint{
			{
				ID:        "p1",
				TriggerID: "retail_month_rate",
				EnsureID:  "wholesale_month_rate",
				Relation:  "gte",
			},
		},
	}

	applied, err := ApplyIndicatorTarget(st, 2025, 12, "retail_month_rate", 30, rs, 0)
	if err != nil {
		t.Fatalf("apply target: %v", err)
	}
	if !hasAppliedRuleType(applied, "compensate") {
		t.Fatalf("expected compensate rule: %+v", applied)
	}
	if value := mustIndicatorValue(t, st, "wholesale_month_rate"); value < 30 {
		t.Fatalf("expected wholesale compensated, got %.2f", value)
	}
}

func TestApplyIndicatorTargetWithRules_DepthOneSkipsCompensate(t *testing.T) {
	st := newRuleTestStore(t)
	insertWRRow(t, st, "W1", "wholesale", 1, 100, 100)
	insertWRRow(t, st, "R1", "retail", 1, 100, 100)

	rs := &rules.RuleSet{
		Compensates: []*rules.CompensateConstraint{
			{
				ID:        "p1",
				TriggerID: "retail_month_rate",
				EnsureID:  "wholesale_month_rate",
				Relation:  "gte",
			},
		},
	}

	applied, err := ApplyIndicatorTarget(st, 2025, 12, "retail_month_rate", 30, rs, 1)
	if err != nil {
		t.Fatalf("apply target: %v", err)
	}
	if hasAppliedRuleType(applied, "compensate") {
		t.Fatalf("did not expect compensate rule: %+v", applied)
	}
	if value := mustIndicatorValue(t, st, "wholesale_month_rate"); value != 0 {
		t.Fatalf("expected wholesale unchanged, got %.2f", value)
	}
}

func newRuleTestStore(t *testing.T) *store.Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}
	return st
}

func insertWRRow(t *testing.T, st *store.Store, code, industry string, scale int, current, lastYear float64) {
	t.Helper()

	err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			sales_current_month, sales_last_year_month,
			sales_current_cumulative, sales_last_year_cumulative,
			retail_current_month, retail_last_year_month,
			retail_current_cumulative, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, code, code, "5101", industry, scale, 1, 2025, 12, current, lastYear, current, lastYear, current, lastYear, current, lastYear, "test", "test.xlsx")
	if err != nil {
		t.Fatalf("insert wr row: %v", err)
	}
}

func mustIndicatorValue(t *testing.T, st *store.Store, id string) float64 {
	t.Helper()

	groups, err := RecalcAll(st, 2025, 12)
	if err != nil {
		t.Fatalf("recalc all: %v", err)
	}
	for _, group := range groups {
		for _, indicator := range group.Indicators {
			if indicator.ID == id {
				return indicator.Value
			}
		}
	}
	t.Fatalf("indicator not found: %s", id)
	return 0
}

func mustSalesCurrentMonth(t *testing.T, st *store.Store, code, industry string) float64 {
	t.Helper()

	var value float64
	err := st.QueryRow(
		"SELECT sales_current_month FROM wholesale_retail WHERE credit_code = ? AND industry_type = ?",
		code,
		industry,
	).Scan(&value)
	if err != nil {
		t.Fatalf("query sales_current_month: %v", err)
	}
	return value
}

func hasAppliedRuleType(applied []AppliedRule, want string) bool {
	for _, item := range applied {
		if item.Type == want {
			return true
		}
	}
	return false
}
