package dagcalc

import (
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

	eng := NewEngine(NewGraph(), st, 2025, 12)
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
	g.AddIndicatorID("indicator:限上社零额_当月值", "限上社零额_当月值")
	eng := NewEngine(g, st, 2025, 12)
	plan, err := eng.ReverseAdjust("indicator:限上社零额_当月值", 0)
	if err != nil {
		t.Fatalf("reverse adjust: %v", err)
	}
	if plan == nil {
		t.Fatalf("expected plan")
	}
}
