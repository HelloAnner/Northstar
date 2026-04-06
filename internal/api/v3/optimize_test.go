package v3

import (
	"path/filepath"
	"testing"

	"northstar/internal/calculator"
	"northstar/internal/dagcalc"
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
	before := findIndicatorValue(groups, "限上社零额增速_累计")

	target := before + 0.5
	if err := applyIndicatorTarget(st, year, month, "限上社零额增速_累计", target); err != nil {
		t.Fatalf("apply target: %v", err)
	}
	if err := recalcDerivedFields(st, year, month); err != nil {
		t.Fatalf("recalc derived: %v", err)
	}

	afterGroups, err := calc.CalculateAll(year, month)
	if err != nil {
		t.Fatalf("calculate indicators after: %v", err)
	}
	after := findIndicatorValue(afterGroups, "限上社零额增速_累计")

	if diff := abs(after - target); diff > 0.05 {
		t.Fatalf("rate not reached: before=%.4f target=%.4f after=%.4f diff=%.4f", before, target, after, diff)
	}
}

func TestOptimize_RandomizeLimitAboveMonthValue(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := st.Exec(`
			INSERT INTO wholesale_retail (
				credit_code, name, industry_code, industry_type, company_scale, row_no,
				data_year, data_month,
				retail_current_month, retail_last_year_month,
				source_sheet, source_file
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "AAA-"+dagcalc.BuildRowID("wr", int64(i+1)), "企业A", "5101", "wholesale", 1, i+1, 2025, 12, 100, 100, "批发", "test.xlsx"); err != nil {
			t.Fatalf("insert wr: %v", err)
		}
	}

	seq := []float64{0.1, 0.9, 0.2, 0.8}
	seqIdx := 0
	restoreRand := dagcalc.SetRandFloat64ForTest(func() float64 {
		v := seq[seqIdx%len(seq)]
		seqIdx++
		return v
	})
	defer restoreRand()

	if err := applyIndicatorTarget(st, 2025, 12, "限上社零额_当月值", 300); err != nil {
		t.Fatalf("apply target: %v", err)
	}

	rows, err := st.Query("SELECT retail_current_month FROM wholesale_retail ORDER BY id")
	if err != nil {
		t.Fatalf("query values: %v", err)
	}
	defer rows.Close()

	var values []float64
	sum := 0.0
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan value: %v", err)
		}
		values = append(values, v)
		sum += v
	}
	if len(values) != 2 {
		t.Fatalf("unexpected row count: %d", len(values))
	}
	if sum != 300 {
		t.Fatalf("unexpected sum: %v", sum)
	}
	if values[0] == values[1] {
		t.Fatalf("expected randomized values, got same: %v", values)
	}
}

func TestRunOptimizeIncludesAppliedRules(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}
	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			sales_current_month, sales_last_year_month,
			sales_current_cumulative, sales_last_year_cumulative,
			retail_current_month, retail_last_year_month,
			retail_current_cumulative, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AAA", "企业A", "5101", "wholesale", 1, 1, 2025, 12, 100, 100, 100, 100, 100, 100, 100, 100, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	max := 10.0
	if _, err := st.CreateAdjustmentConstraint(store.AdjustmentConstraint{
		Type:        "clamp_target",
		IndicatorID: "wholesale_month_rate",
		MaxValue:    &max,
		Enabled:     true,
	}); err != nil {
		t.Fatalf("create constraint: %v", err)
	}

	eng := dagcalc.NewEngine(dagcalc.NewGraph(), st, 2025, 12)
	if err := eng.ReloadRules(); err != nil {
		t.Fatalf("reload rules: %v", err)
	}

	resp, err := runOptimize(eng, st, 2025, 12, map[string]float64{"wholesale_month_rate": 50})
	if err != nil {
		t.Fatalf("run optimize: %v", err)
	}
	if len(resp.AppliedRules) != 1 {
		t.Fatalf("unexpected applied rules: %+v", resp.AppliedRules)
	}
	if resp.AppliedRules[0].Type != "clamp_target" {
		t.Fatalf("unexpected applied rule: %+v", resp.AppliedRules[0])
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
