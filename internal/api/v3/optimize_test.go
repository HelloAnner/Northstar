package v3

import (
	"path/filepath"
	"testing"

	"northstar/internal/calculator"
	"northstar/internal/importer"
	"northstar/internal/store"
)

func TestOptimize_AdjustLimitAboveCumulativeRate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	coord := importer.NewCoordinator(st)
	ch := coord.Import(importer.ImportOptions{
		FilePath:         filepath.Join("..", "..", "..", "prd", "12月月报（预估）_补全企业名称社会代码_20260129.xlsx"),
		OriginalFilename: "12月月报（预估）_补全企业名称社会代码_20260129.xlsx",
		ClearExisting:    true,
		UpdateConfigYM:   true,
		CalculateFields:  true,
	})
	for evt := range ch {
		if evt.Type == "error" {
			t.Fatalf("import error: %s", evt.Message)
		}
	}

	year, month, err := st.GetCurrentYearMonth()
	if err != nil {
		t.Fatalf("get current ym: %v", err)
	}

	calc := calculator.NewCalculator(st)
	groups, err := calc.CalculateAll(year, month)
	if err != nil {
		t.Fatalf("calculate indicators: %v", err)
	}
	before := findIndicatorValue(groups, "limitAbove_cumulative_rate")

	target := before + 0.5
	if err := applyIndicatorTarget(st, year, month, "limitAbove_cumulative_rate", target); err != nil {
		t.Fatalf("apply target: %v", err)
	}
	if err := recalcDerivedFields(st, year, month); err != nil {
		t.Fatalf("recalc derived: %v", err)
	}

	afterGroups, err := calc.CalculateAll(year, month)
	if err != nil {
		t.Fatalf("calculate indicators after: %v", err)
	}
	after := findIndicatorValue(afterGroups, "limitAbove_cumulative_rate")

	if diff := abs(after - target); diff > 0.05 {
		t.Fatalf("rate not reached: before=%.4f target=%.4f after=%.4f diff=%.4f", before, target, after, diff)
	}
}

func findIndicatorValue(groups []calculator.IndicatorGroup, id string) float64 {
	for _, g := range groups {
		for _, it := range g.Indicators {
			if it.ID == id {
				return it.Value
			}
		}
	}
	return 0
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
