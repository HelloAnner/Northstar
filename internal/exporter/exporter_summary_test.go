/**
 * 导出汇总取数测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

package exporter

import (
	"path/filepath"
	"testing"

	"northstar/internal/store"
)

func TestExport_UsesSummaryTables(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	// 写入汇总表数据
	if err := st.Exec(`
		INSERT INTO summary_micro_small (data_year, data_month, row_key, row_no, rate)
		VALUES (2025, 12, '', 6, 12)
	`); err != nil {
		t.Fatalf("insert summary_micro_small failed: %v", err)
	}
	if err := st.Exec(`
		INSERT INTO summary_eat_wear_use (data_year, data_month, row_key, row_no, rate)
		VALUES (2025, 12, '', 5, 34)
	`); err != nil {
		t.Fatalf("insert summary_eat_wear_use failed: %v", err)
	}

	indicators, err := calculateIndicatorIndex(st, 2025, 12)
	if err != nil {
		t.Fatalf("calculateIndicatorIndex failed: %v", err)
	}
	if indicators["microSmall_month_rate"].Value != 12 {
		t.Fatalf("microSmall rate mismatch: %v", indicators["microSmall_month_rate"].Value)
	}
	if indicators["eatWearUse_month_rate"].Value != 34 {
		t.Fatalf("eatWearUse rate mismatch: %v", indicators["eatWearUse_month_rate"].Value)
	}
}
