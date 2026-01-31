package exporter

import (
	"path/filepath"
	"testing"

	"northstar/internal/calculator"
	"northstar/internal/store"
)

func TestExport_PreserveTemplateFormulas(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}

	f, err := openEmbeddedMonthReportTemplate()
	if err != nil {
		t.Fatalf("open template: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	// 回归：导出流程在写入关键输入值时，不能把模板公式“物化”为数值（否则定稿表失真）
	// 这里只验证“可能清公式”的两个关键步骤：社零额（定）输入写入、汇总表（定）输入写入。
	idx := indicatorIndex{
		"microSmall_month_rate":  calculator.Indicator{ID: "microSmall_month_rate", Value: 0},
		"eatWearUse_month_rate":  calculator.Indicator{ID: "eatWearUse_month_rate", Value: 0},
		"totalSocial_cumulative_value": calculator.Indicator{ID: "totalSocial_cumulative_value", Value: 0},
		"totalSocial_cumulative_rate":  calculator.Indicator{ID: "totalSocial_cumulative_rate", Value: 0},
	}
	if err := fillSocialRetailSheetAndMaterialize(f, st, 2025, 12, idx); err != nil {
		t.Fatalf("fill social retail: %v", err)
	}
	if err := rewriteFixedSummarySheet(f, 2025, 12, wrSums{}, wrSums{}, wrSums{}, wrSums{}, idx, nil, nil); err != nil {
		t.Fatalf("rewrite summary: %v", err)
	}

	for _, tc := range []struct {
		sheet string
		cell  string
	}{
		{sheet: "社零额（定）", cell: "K3"},
		{sheet: "社零额（定）", cell: "K7"},
		{sheet: "社零额（定）", cell: "K9"},
		{sheet: "汇总表（定）", cell: "D4"},
		{sheet: "汇总表（定）", cell: "F4"},
		{sheet: "汇总表（定）", cell: "D10"},
		{sheet: "汇总表（定）", cell: "A11"},
	} {
		formula, err := f.GetCellFormula(tc.sheet, tc.cell)
		if err != nil {
			t.Fatalf("get formula %s!%s: %v", tc.sheet, tc.cell, err)
		}
		if formula == "" {
			t.Fatalf("expected formula preserved at %s!%s, got empty", tc.sheet, tc.cell)
		}
	}
}
