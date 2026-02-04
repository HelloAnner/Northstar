/**
 * 指标反推调整测试
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

func TestApplyIndicatorTargetRandomizesValues(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := st.Exec(`
			INSERT INTO wholesale_retail (
				credit_code, name, industry_code, industry_type, company_scale, row_no,
				data_year, data_month,
				retail_current_month, retail_last_year_month,
				source_sheet, source_file
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "AAA", "企业A", "5101", "wholesale", 1, i+1, 2025, 12, 100, 100, "批发", "test.xlsx"); err != nil {
			t.Fatalf("insert wr: %v", err)
		}
	}

	originRand := randFloat64
	seq := []float64{0.1, 0.9, 0.2, 0.8}
	seqIdx := 0
	randFloat64 = func() float64 {
		v := seq[seqIdx%len(seq)]
		seqIdx++
		return v
	}
	defer func() { randFloat64 = originRand }()

	if err := ApplyIndicatorTarget(st, 2025, 12, "limitAbove_month_value", 300); err != nil {
		t.Fatalf("apply target: %v", err)
	}

	rows, err := st.Query("SELECT retail_current_month FROM wholesale_retail ORDER BY id")
	if err != nil {
		t.Fatalf("query values: %v", err)
	}
	defer rows.Close()

	var values []float64
	sum := 0.0
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan value: %v", err)
		}
		values = append(values, v)
		sum += v
	}
	if len(values) != 2 {
		t.Fatalf("unexpected row count: %d", len(values))
	}
	if sum != 300 {
		t.Fatalf("unexpected sum: %v", sum)
	}
	if values[0] == values[1] {
		t.Fatalf("expected randomized values, got same: %v", values)
	}
}

func TestApplyIndicatorTarget_NoAdjustDataNoError(t *testing.T) {
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
			retail_current_month, retail_last_year_month,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "BBB", "企业B", "5101", "wholesale", 1, 1, 2025, 12, 120, 110, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	if err := ApplyIndicatorTarget(st, 2025, 12, "microSmall_month_rate", 5); err != nil {
		t.Fatalf("apply target should ignore empty dataset: %v", err)
	}
}
