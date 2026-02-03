/**
 * 社零总额计算测试
 *
 * @author Anner
 * Created on 2026/2/3
 */

package calculator

import (
	"path/filepath"
	"testing"

	"northstar/internal/store"
)

func TestTotalSocialRateIncludesACLastYearCumulative(t *testing.T) {
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
			retail_current_cumulative, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "WR", "批发企业", "5101", "wholesale", 1, 1, 2025, 12, 100, 100, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	if err := st.Exec(`
		INSERT INTO accommodation_catering (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			food_current_cumulative, goods_current_cumulative,
			food_last_year_cumulative, goods_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AC", "住宿企业", "6101", "accommodation", 1, 1, 2025, 12, 60, 40, 60, 40, "住宿", "test.xlsx"); err != nil {
		t.Fatalf("insert ac: %v", err)
	}

	groups, err := NewCalculator(st).CalculateAll(2025, 12)
	if err != nil {
		t.Fatalf("calc: %v", err)
	}

	var totalRate float64
	found := false
	for _, g := range groups {
		for _, it := range g.Indicators {
			if it.ID == "totalSocial_cumulative_rate" {
				totalRate = it.Value
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("missing totalSocial_cumulative_rate")
	}
	if totalRate != 0 {
		t.Fatalf("unexpected total rate: %v", totalRate)
	}
}
