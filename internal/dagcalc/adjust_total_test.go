/**
 * 社零总额反推测试
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

func TestApplyTotalSocialCumulativeValueAdjustsConfig(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}
	if err := st.SetConfigFloat("last_year_limit_below_cumulative", 100); err != nil {
		t.Fatalf("set config: %v", err)
	}

	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			retail_current_month, retail_current_cumulative,
			retail_last_year_month, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AAA", "企业A", "5101", "wholesale", 1, 1, 2025, 12, 100, 100, 100, 100, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	if _, err := ApplyIndicatorTarget(st, 2025, 12, "totalSocial_cumulative_value", 300, nil, 0); err != nil {
		t.Fatalf("apply target: %v", err)
	}

	value, err := st.GetConfigFloat("last_year_limit_below_cumulative")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if value <= 100 {
		t.Fatalf("expected config updated, got %.2f", value)
	}
}
