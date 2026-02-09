/**
 * 指标公式计算测试
 *
 * @author Anner
 * Created on 2026/2/6
 */

package dagcalc

import (
	"math"
	"path/filepath"
	"testing"

	"northstar/internal/store"
)

func TestCalculateIndicators_FromDefinitions(t *testing.T) {
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
		t.Fatalf("set config failed: %v", err)
	}

	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			sales_current_month, sales_last_year_month,
			sales_current_cumulative, sales_last_year_cumulative,
			retail_current_month, retail_last_year_month,
			retail_current_cumulative, retail_last_year_cumulative,
			is_small_micro, is_eat_wear_use,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "WR1", "零售样本", "5201", "retail", 3, 1, 2025, 12, 300, 200, 1200, 1000, 200, 100, 800, 700, 1, 1, "零售", "test.xlsx"); err != nil {
		t.Fatalf("insert wr failed: %v", err)
	}

	if err := st.Exec(`
		INSERT INTO accommodation_catering (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			revenue_current_month, revenue_last_year_month,
			revenue_current_cumulative, revenue_last_year_cumulative,
			food_current_month, goods_current_month,
			food_last_year_month, goods_last_year_month,
			food_current_cumulative, goods_current_cumulative,
			food_last_year_cumulative, goods_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AC1", "住宿样本", "6101", "accommodation", 2, 1, 2025, 12, 150, 100, 600, 400, 60, 40, 30, 20, 240, 160, 120, 80, "住宿", "test.xlsx"); err != nil {
		t.Fatalf("insert ac failed: %v", err)
	}

	groups, err := CalculateIndicators(st, 2025, 12)
	if err != nil {
		t.Fatalf("calculate indicators failed: %v", err)
	}

	valueMap := map[string]float64{}
	for _, g := range groups {
		for _, it := range g.Indicators {
			valueMap[it.ID] = it.Value
		}
	}

	if got := valueMap["限上社零额_当月值"]; math.Abs(got-300) > 0.01 {
		t.Fatalf("限上社零额_当月值 mismatch: got %.4f", got)
	}
	if got := valueMap["限上社零额_累计值"]; math.Abs(got-1200) > 0.01 {
		t.Fatalf("限上社零额_累计值 mismatch: got %.4f", got)
	}
	if got := valueMap["社零总额_累计值"]; got <= valueMap["限上社零额_累计值"] {
		t.Fatalf("社零总额_累计值 should include limit below estimate: got %.4f", got)
	}
	if _, ok := valueMap["小微企业增速_当月"]; !ok {
		t.Fatalf("missing 小微企业增速_当月")
	}
}
