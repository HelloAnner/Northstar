package v3

import (
	"path/filepath"
	"testing"

	"northstar/internal/store"
)

func TestRecalcDerivedFields_LastYearZero_UseMinus100(t *testing.T) {
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
	`, "ZZZ", "企业Z", "5101", "retail", 1, 1, 2025, 12, 0, 0, 0, 0, 0, 0, 0, 0, "零售", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	if err := st.Exec(`
		INSERT INTO accommodation_catering (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			revenue_current_month, revenue_last_year_month,
			revenue_current_cumulative, revenue_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "YYY", "企业Y", "6101", "accommodation", 1, 1, 2025, 12, 0, 0, 0, 0, "住宿", "test.xlsx"); err != nil {
		t.Fatalf("insert ac: %v", err)
	}

	if err := recalcDerivedFields(st, 2025, 12); err != nil {
		t.Fatalf("recalc derived: %v", err)
	}

	var salesRate float64
	var salesCumRate float64
	var retailRate float64
	var retailCumRate float64
	if err := st.QueryRow(`
		SELECT sales_month_rate, sales_cumulative_rate, retail_month_rate, retail_cumulative_rate
		FROM wholesale_retail WHERE credit_code = ?
	`, "ZZZ").Scan(&salesRate, &salesCumRate, &retailRate, &retailCumRate); err != nil {
		t.Fatalf("query wr: %v", err)
	}
	if salesRate != -100 || salesCumRate != -100 || retailRate != -100 || retailCumRate != -100 {
		t.Fatalf("unexpected wr rates: sales=%v salesCum=%v retail=%v retailCum=%v", salesRate, salesCumRate, retailRate, retailCumRate)
	}

	var revRate float64
	var revCumRate float64
	if err := st.QueryRow(`
		SELECT revenue_month_rate, revenue_cumulative_rate
		FROM accommodation_catering WHERE credit_code = ?
	`, "YYY").Scan(&revRate, &revCumRate); err != nil {
		t.Fatalf("query ac: %v", err)
	}
	if revRate != -100 || revCumRate != -100 {
		t.Fatalf("unexpected ac rates: rev=%v revCum=%v", revRate, revCumRate)
	}
}

