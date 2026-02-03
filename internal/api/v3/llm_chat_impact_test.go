/**
 * LLM 工具影响范围构建测试
 *
 * @author Anner
 * Created on 2026/2/3
 */

package v3

import (
	"path/filepath"
	"strconv"
	"testing"

	"northstar/internal/linkage"
	"northstar/internal/store"
)

func TestBuildLLMToolImpact_WRAnchor(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}

	index, err := linkage.LoadTemplateIndex()
	if err != nil {
		t.Fatalf("load template: %v", err)
	}
	code := index.FirstCode("批发")
	if code == "" {
		t.Fatalf("missing code")
	}

	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			sales_current_month, sales_last_year_month,
			retail_current_month, retail_last_year_month,
			sales_current_cumulative, sales_last_year_cumulative,
			retail_current_cumulative, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AAA", "企业A", code, "wholesale", 1, 1, 2025, 12, 120, 100, 80, 70, 1000, 900, 600, 550, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	var id int64
	if err := st.QueryRow("SELECT id FROM wholesale_retail WHERE credit_code = ?", "AAA").Scan(&id); err != nil {
		t.Fatalf("query id: %v", err)
	}

	impact, err := buildLLMToolImpact(st, 2025, 12, []llmAppliedUpdate{{
		Kind:   "wr",
		ID:     id,
		Fields: []string{"sales_current_month"},
	}}, map[string]float64{})
	if err != nil {
		t.Fatalf("build impact: %v", err)
	}
	if impact.ToolPositionCount != 1 {
		t.Fatalf("tool position count mismatch: %d", impact.ToolPositionCount)
	}
	rowID := "wr:" + strconv.FormatInt(id, 10)
	if !hasUICell(impact.ImpactCells, rowID, "salesCurrentMonth") {
		t.Fatalf("missing ui cell highlight for salesCurrentMonth")
	}
	if !containsString(impact.ImpactIndicators, "wholesale_month_rate") {
		t.Fatalf("missing indicator highlight for wholesale_month_rate")
	}
}

func TestBuildLLMToolImpact_ACRevenueMap(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}

	index, err := linkage.LoadTemplateIndex()
	if err != nil {
		t.Fatalf("load template: %v", err)
	}
	code := index.FirstCode("住宿")
	if code == "" {
		t.Fatalf("missing code")
	}

	if err := st.Exec(`
		INSERT INTO accommodation_catering (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			revenue_current_month, revenue_last_year_month,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "BBB", "企业B", code, "accommodation", 1, 1, 2025, 12, 50, 40, "住宿", "test.xlsx"); err != nil {
		t.Fatalf("insert ac: %v", err)
	}

	var id int64
	if err := st.QueryRow("SELECT id FROM accommodation_catering WHERE credit_code = ?", "BBB").Scan(&id); err != nil {
		t.Fatalf("query id: %v", err)
	}

	impact, err := buildLLMToolImpact(st, 2025, 12, []llmAppliedUpdate{{
		Kind:   "ac",
		ID:     id,
		Fields: []string{"revenue_current_month"},
	}}, map[string]float64{})
	if err != nil {
		t.Fatalf("build impact: %v", err)
	}
	if impact.ToolPositionCount != 1 {
		t.Fatalf("tool position count mismatch: %d", impact.ToolPositionCount)
	}
	rowID := "ac:" + strconv.FormatInt(id, 10)
	if !hasUICell(impact.ImpactCells, rowID, "salesCurrentMonth") {
		t.Fatalf("missing ui cell highlight for ac salesCurrentMonth")
	}
}

func TestBuildLLMToolImpact_IndicatorAnchor(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}

	index, err := linkage.LoadTemplateIndex()
	if err != nil {
		t.Fatalf("load template: %v", err)
	}
	code := index.FirstCode("批发")
	if code == "" {
		t.Fatalf("missing code")
	}

	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			sales_current_month, sales_last_year_month,
			retail_current_month, retail_last_year_month,
			sales_current_cumulative, sales_last_year_cumulative,
			retail_current_cumulative, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "CCC", "企业C", code, "wholesale", 1, 1, 2025, 12, 120, 100, 80, 70, 1000, 900, 600, 550, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	impact, err := buildLLMToolImpact(st, 2025, 12, nil, map[string]float64{
		"limitAbove_month_value": 123.0,
	})
	if err != nil {
		t.Fatalf("build impact: %v", err)
	}
	if impact.ToolPositionCount != 1 {
		t.Fatalf("tool position count mismatch: %d", impact.ToolPositionCount)
	}
	if !containsString(impact.ImpactIndicators, "limitAbove_month_value") {
		t.Fatalf("missing indicator highlight for limitAbove_month_value")
	}
}

func hasUICell(cells []linkage.UICoord, rowID, columnKey string) bool {
	for _, c := range cells {
		if c.RowID == rowID && c.ColumnKey == columnKey {
			return true
		}
	}
	return false
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
