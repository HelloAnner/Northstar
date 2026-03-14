/**
 * 规则约束测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

package rules

import "testing"

func TestClampTargetConstraintClamp(t *testing.T) {
	min := -10.0
	max := 15.0
	c := ClampTargetConstraint{
		ID:          "c1",
		IndicatorID: "retail_month_rate",
		Min:         &min,
		Max:         &max,
	}

	clamped, changed := c.Clamp("retail_month_rate", 20)
	if !changed {
		t.Fatalf("expected target to be clamped")
	}
	if clamped != 15 {
		t.Fatalf("unexpected clamped value: %.2f", clamped)
	}
}

func TestFilterAllocationConstraintApply(t *testing.T) {
	c := FilterAllocationConstraint{
		ID:          "f1",
		IndicatorID: "retail_month_rate",
		Filter:      "positive_current",
	}
	rows := []CompanyRow{
		{RowID: 1, CurrentValue: 5},
		{RowID: 2, CurrentValue: -2},
	}

	filtered, changed := c.Apply("retail_month_rate", rows)
	if !changed {
		t.Fatalf("expected rows to be filtered")
	}
	if len(filtered) != 1 || filtered[0].RowID != 1 {
		t.Fatalf("unexpected filtered rows: %+v", filtered)
	}
}

func TestCompensateConstraintCheck(t *testing.T) {
	c := CompensateConstraint{
		ID:        "p1",
		TriggerID: "retail_month_rate",
		EnsureID:  "wholesale_month_rate",
		Relation:  "gte",
		Tolerance: 1,
	}
	indicators := map[string]float64{
		"retail_month_rate":    10,
		"wholesale_month_rate": 7,
	}

	need, target := c.Check("retail_month_rate", indicators)
	if !need {
		t.Fatalf("expected compensation to be required")
	}
	if target != 9 {
		t.Fatalf("unexpected compensate target: %.2f", target)
	}
}
