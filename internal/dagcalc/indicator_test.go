/**
 * 统一指标计算测试
 *
 * @author Anner
 * Created on 2026/2/4
 */

package dagcalc

import (
	"math"
	"path/filepath"
	"testing"

	"northstar/internal/store"
)

func TestCalculateIndicatorsTotalSocialRate(t *testing.T) {
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

	groups, err := CalculateIndicators(st, 2025, 12)
	if err != nil {
		t.Fatalf("calc: %v", err)
	}

	var totalRate float64
	found := false
	for _, g := range groups {
		for _, it := range g.Indicators {
			if it.ID == "社零总额增速_累计" {
				totalRate = it.Value
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("missing 社零总额增速_累计")
	}
	if totalRate != 0 {
		t.Fatalf("unexpected total rate: %v", totalRate)
	}
}

func TestCalculateIndicatorsTotalSocialRate_WithLimitBelowCompositeRule(t *testing.T) {
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
		t.Fatalf("set last_year_limit_below_cumulative: %v", err)
	}
	if err := st.SetConfigFloat("small_micro_rate_prev", 5); err != nil {
		t.Fatalf("set small_micro_rate_prev: %v", err)
	}
	if err := st.SetConfigFloat("eat_wear_use_rate_prev", 3); err != nil {
		t.Fatalf("set eat_wear_use_rate_prev: %v", err)
	}
	if err := st.SetConfigFloat("sample_rate_prev", 2); err != nil {
		t.Fatalf("set sample_rate_prev: %v", err)
	}
	if err := st.SetConfigFloat("sample_rate_month", 4); err != nil {
		t.Fatalf("set sample_rate_month: %v", err)
	}
	if err := st.SetConfigFloat("weight_small_micro", 0.5); err != nil {
		t.Fatalf("set weight_small_micro: %v", err)
	}
	if err := st.SetConfigFloat("weight_eat_wear_use", 0.3); err != nil {
		t.Fatalf("set weight_eat_wear_use: %v", err)
	}
	if err := st.SetConfigFloat("weight_sample", 0.2); err != nil {
		t.Fatalf("set weight_sample: %v", err)
	}
	if err := st.SetConfigFloat("province_limit_below_rate_change", 1); err != nil {
		t.Fatalf("set province_limit_below_rate_change: %v", err)
	}

	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			retail_current_month, retail_last_year_month,
			retail_current_cumulative, retail_last_year_cumulative,
			is_small_micro, is_eat_wear_use,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "SM", "小微样本", "5101", "retail", 3, 1, 2025, 12, 120, 100, 120, 100, 1, 0, "零售", "test.xlsx"); err != nil {
		t.Fatalf("insert micro sample: %v", err)
	}

	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			retail_current_month, retail_last_year_month,
			retail_current_cumulative, retail_last_year_cumulative,
			is_small_micro, is_eat_wear_use,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "EWU", "吃穿用样本", "5201", "retail", 2, 2, 2025, 12, 110, 100, 110, 100, 0, 1, "零售", "test.xlsx"); err != nil {
		t.Fatalf("insert eat-wear-use sample: %v", err)
	}

	groups, err := CalculateIndicators(st, 2025, 12)
	if err != nil {
		t.Fatalf("calc: %v", err)
	}

	totalValue := 0.0
	totalRate := 0.0
	foundValue := false
	foundRate := false
	for _, g := range groups {
		for _, it := range g.Indicators {
			if it.ID == "社零总额_累计值" {
				totalValue = it.Value
				foundValue = true
			}
			if it.ID == "社零总额增速_累计" {
				totalRate = it.Value
				foundRate = true
			}
		}
	}
	if !foundValue || !foundRate {
		t.Fatalf("missing total social indicators: value=%v rate=%v", foundValue, foundRate)
	}

	// 期望值：
	// 小微增速=20，吃穿用增速=10
	// 限下增速变动量=(20-5)*0.5 + (10-3)*0.3 + (4-2)*0.2 + 1 = 11
	// 上月限下增速=5*0.5 + 3*0.3 + 2*0.2 = 3.8
	// 本月限下增速=14.8，限下累计=100*(1+14.8%)=114.8
	// 限上累计=230，总社零累计=344.8
	wantValue := 344.8
	if math.Abs(totalValue-wantValue) > 0.01 {
		t.Fatalf("unexpected total value: got=%.4f want=%.4f", totalValue, wantValue)
	}

	// 上年累计总额=200+100=300，增速=(344.8-300)/300*100
	wantRate := (wantValue - 300) / 300 * 100
	if math.Abs(totalRate-wantRate) > 0.01 {
		t.Fatalf("unexpected total rate: got=%.4f want=%.4f", totalRate, wantRate)
	}
}
