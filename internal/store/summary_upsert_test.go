/**
 * 汇总覆盖导入测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

package store

import (
	"path/filepath"
	"testing"

	"northstar/internal/model"
)

func TestBatchInsertSummaryLimitAbove_Upsert(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	year, month := 2025, 12
	firstVal := 1.0
	secondVal := 2.5

	first := []model.SummaryLimitAboveRetail{{
		DataYear:     year,
		DataMonth:    month,
		RowKey:       "total",
		RowNo:        1,
		ValueCurrent: &firstVal,
	}}
	if err := st.BatchInsertSummaryLimitAbove(first); err != nil {
		t.Fatalf("insert summary failed: %v", err)
	}

	second := []model.SummaryLimitAboveRetail{{
		DataYear:     year,
		DataMonth:    month,
		RowKey:       "total",
		RowNo:        1,
		ValueCurrent: &secondVal,
	}}
	if err := st.BatchInsertSummaryLimitAbove(second); err != nil {
		t.Fatalf("upsert summary failed: %v", err)
	}

	var count int
	if err := st.QueryRow(
		`SELECT COUNT(*) FROM summary_limit_above_retail WHERE data_year = ? AND data_month = ? AND row_key = ?`,
		year, month, "total",
	).Scan(&count); err != nil {
		t.Fatalf("count summary failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("unexpected count: %d", count)
	}

	var current float64
	if err := st.QueryRow(
		`SELECT value_current FROM summary_limit_above_retail WHERE data_year = ? AND data_month = ? AND row_key = ?`,
		year, month, "total",
	).Scan(&current); err != nil {
		t.Fatalf("query summary failed: %v", err)
	}
	if current != secondVal {
		t.Fatalf("value not updated: %v", current)
	}
}
