package importer

import (
	"path/filepath"
	"testing"

	"northstar/internal/store"
)

func TestBackfillWRFromSnapshot_FillsPrevMonthAndCumulative(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	year := 2025
	month := 12

	// 主表：缺少上月字段
	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			sales_prev_month, sales_prev_cumulative,
			sales_current_cumulative, sales_current_month
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AAA", "企业A", "5101", "wholesale", 1, 2, year, month, 0, 0, 150, 0); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	// 快照：提供 2025-11 的本年-本月与本年累计（用于填充上月）
	if err := st.Exec(`
		INSERT INTO wr_snapshot (
			snapshot_year, snapshot_month, snapshot_name,
			credit_code, name, industry_code, company_scale,
			sales_current_month, sales_current_cumulative,
			retail_current_month, retail_current_cumulative
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 2025, 11, "2025年11月批零", "AAA", "企业A", "5101", 1, 123, 456, 7, 89); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	c := NewCoordinator(st)
	if err := c.backfillCalculableFields(year, month); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	var salesPrevMonth float64
	var salesPrevCum float64
	var retailPrevMonth float64
	var retailPrevCum float64
	if err := st.QueryRow(`
		SELECT sales_prev_month, sales_prev_cumulative, retail_prev_month, retail_prev_cumulative
		FROM wholesale_retail WHERE credit_code = ?
	`, "AAA").Scan(&salesPrevMonth, &salesPrevCum, &retailPrevMonth, &retailPrevCum); err != nil {
		t.Fatalf("query: %v", err)
	}

	if salesPrevMonth != 123 || salesPrevCum != 456 {
		t.Fatalf("unexpected sales prev: month=%v cum=%v", salesPrevMonth, salesPrevCum)
	}
	if retailPrevMonth != 7 || retailPrevCum != 89 {
		t.Fatalf("unexpected retail prev: month=%v cum=%v", retailPrevMonth, retailPrevCum)
	}
}

