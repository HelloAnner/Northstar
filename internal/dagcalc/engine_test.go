package dagcalc

import (
	"os"
	"path/filepath"
	"testing"

	"northstar/internal/store"
)

func TestEngineForwardRecalcReturnsPlan(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	eng := NewEngine(NewGraph(), st, 2025, 12, "")
	plan, err := eng.ForwardRecalc("a")
	if err != nil {
		t.Fatalf("forward recalc: %v", err)
	}
	if plan == nil {
		t.Fatalf("expected plan")
	}
}

func TestEngineReverseAdjustReturnsImpact(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			retail_current_month, retail_last_year_month,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AAA", "企业A", "5101", "wholesale", 1, 1, 2025, 12, 100, 100, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	g := NewGraph()
	g.AddIndicatorID("indicator:limitAbove_month_value", "limitAbove_month_value")
	eng := NewEngine(g, st, 2025, 12, "")
	plan, err := eng.ReverseAdjust("indicator:limitAbove_month_value", 0)
	if err != nil {
		t.Fatalf("reverse adjust: %v", err)
	}
	if plan == nil {
		t.Fatalf("expected plan")
	}
}

func TestEngineOptimizeReturnsAppliedRules(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

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

	rolePath := filepath.Join(t.TempDir(), "role.json")
	if err := os.WriteFile(rolePath, []byte(`{
  "version": "1.0",
  "rules": [
    {"id":"c1","name":"限上","type":"clamp_target","indicator":"wholesale_month_rate","max":10}
  ]
}`), 0644); err != nil {
		t.Fatalf("write role.json: %v", err)
	}

	eng := NewEngine(NewGraph(), st, 2025, 12, rolePath)
	if err := eng.ReloadRules(); err != nil {
		t.Fatalf("reload rules: %v", err)
	}

	result, err := eng.Optimize(map[string]float64{"wholesale_month_rate": 50})
	if err != nil {
		t.Fatalf("optimize: %v", err)
	}
	if result == nil {
		t.Fatalf("expected optimize result")
	}
	if len(result.AppliedRules) != 1 {
		t.Fatalf("unexpected applied rule count: %d", len(result.AppliedRules))
	}
	if result.AppliedRules[0].Type != "clamp_target" {
		t.Fatalf("unexpected applied rule: %+v", result.AppliedRules[0])
	}
}
