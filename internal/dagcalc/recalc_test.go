/**
 * 统一重算入口测试
 *
 * @author Anner
 * Created on 2026/2/4
 */

package dagcalc

import (
	"path/filepath"
	"testing"

	"northstar/internal/store"
)

func TestRecalcAllUpdatesRatesAndIndicators(t *testing.T) {
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
	`, "AAA", "企业A", "5101", "wholesale", 1, 1, 2025, 12, 0, 0, 0, 0, 0, 0, 0, 0, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	groups, err := RecalcAll(st, 2025, 12)
	if err != nil {
		t.Fatalf("recalc all: %v", err)
	}
	if len(groups) != 4 {
		t.Fatalf("unexpected group size: %d", len(groups))
	}

	var salesRate float64
	if err := st.QueryRow(`
		SELECT sales_month_rate FROM wholesale_retail WHERE credit_code = ?
	`, "AAA").Scan(&salesRate); err != nil {
		t.Fatalf("query wr: %v", err)
	}
	if salesRate != -100 {
		t.Fatalf("unexpected sales rate: %v", salesRate)
	}
}
