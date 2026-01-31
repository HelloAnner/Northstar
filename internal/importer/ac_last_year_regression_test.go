package importer

import (
	"path/filepath"
	"testing"

	"northstar/internal/store"
)

func TestImport_PRD_Accommodation_LastYearRevenueNotMissing(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	coord := NewCoordinator(st)
	ch := coord.Import(ImportOptions{
		FilePath:         filepath.Join("..", "..", "prd", "12月月报（预估）_补全企业名称社会代码_20260129.xlsx"),
		OriginalFilename: "12月月报（预估）_补全企业名称社会代码_20260129.xlsx",
		ClearExisting:    true,
		UpdateConfigYM:   true,
		CalculateFields:  false,
	})
	for evt := range ch {
		if evt.Type == "error" {
			t.Fatalf("import error: %s", evt.Message)
		}
	}

	creditCode := "91320500TCE5UYN5HX"
	expectedLastYearMonth := 327.0
	expectedLastYearCumulative := 3620.0

	var gotMonth float64
	var gotCum float64
	if err := st.QueryRow(
		"SELECT revenue_last_year_month, revenue_last_year_cumulative FROM accommodation_catering WHERE credit_code = ?",
		creditCode,
	).Scan(&gotMonth, &gotCum); err != nil {
		t.Fatalf("query accommodation_catering: %v", err)
	}

	if gotMonth != expectedLastYearMonth {
		t.Fatalf("unexpected revenue_last_year_month: got=%v want=%v", gotMonth, expectedLastYearMonth)
	}
	if gotCum != expectedLastYearCumulative {
		t.Fatalf("unexpected revenue_last_year_cumulative: got=%v want=%v", gotCum, expectedLastYearCumulative)
	}
}
