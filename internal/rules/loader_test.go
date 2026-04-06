/**
 * 规则加载测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

package rules

import (
	"path/filepath"
	"testing"

	"northstar/internal/store"
)

func TestLoadFromStoreDispatchesConstraintsByType(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	min := -5.0
	max := 10.0
	if _, err := st.CreateAdjustmentConstraint(store.AdjustmentConstraint{
		Type: "clamp_target", IndicatorID: "retail_month_rate",
		MinValue: &min, MaxValue: &max, Enabled: true,
	}); err != nil {
		t.Fatalf("create clamp: %v", err)
	}
	if _, err := st.CreateAdjustmentConstraint(store.AdjustmentConstraint{
		Type: "filter_allocation", IndicatorID: "retail_month_rate",
		FilterMode: "positive_current", Enabled: true,
	}); err != nil {
		t.Fatalf("create filter: %v", err)
	}
	if _, err := st.CreateAdjustmentConstraint(store.AdjustmentConstraint{
		Type: "compensate", TriggerID: "retail_month_rate",
		EnsureID: "wholesale_month_rate", Relation: "gte", Tolerance: 1, Enabled: true,
	}); err != nil {
		t.Fatalf("create compensate: %v", err)
	}

	rs, err := LoadFromStore(st)
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}
	if len(rs.Clamps) != 1 {
		t.Fatalf("unexpected clamp count: %d", len(rs.Clamps))
	}
	if len(rs.Filters) != 1 {
		t.Fatalf("unexpected filter count: %d", len(rs.Filters))
	}
	if len(rs.Compensates) != 1 {
		t.Fatalf("unexpected compensate count: %d", len(rs.Compensates))
	}
	if rs.Clamps[0].IndicatorID != "retail_month_rate" {
		t.Fatalf("unexpected clamp indicator: %s", rs.Clamps[0].IndicatorID)
	}
	if rs.Filters[0].Filter != "positive_current" {
		t.Fatalf("unexpected filter mode: %s", rs.Filters[0].Filter)
	}
	if rs.Compensates[0].EnsureID != "wholesale_month_rate" {
		t.Fatalf("unexpected ensure indicator: %s", rs.Compensates[0].EnsureID)
	}
}

func TestLoadFromStoreNilReturnsEmptyRuleSet(t *testing.T) {
	rs, err := LoadFromStore(nil)
	if err != nil {
		t.Fatalf("load nil store: %v", err)
	}
	if len(rs.Clamps) != 0 || len(rs.Filters) != 0 || len(rs.Compensates) != 0 {
		t.Fatalf("expected empty ruleset: %+v", rs)
	}
}
