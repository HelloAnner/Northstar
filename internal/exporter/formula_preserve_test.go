package exporter

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
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
		"小微企业增速_当月":  calculator.Indicator{ID: "小微企业增速_当月", Value: 0},
		"吃穿用增速_当月":  calculator.Indicator{ID: "吃穿用增速_当月", Value: 0},
		"社零总额_累计值": calculator.Indicator{ID: "社零总额_累计值", Value: 0},
		"社零总额增速_累计":  calculator.Indicator{ID: "社零总额增速_累计", Value: 0},
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

func TestParameterizeNumbersThenMaterialize(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)
	if err := f.SetCellValue(sheet, "A1", 123); err != nil {
		t.Fatalf("set A1: %v", err)
	}
	if err := f.SetCellValue(sheet, "B1", "同比增长12.5%"); err != nil {
		t.Fatalf("set B1: %v", err)
	}
	if err := f.SetCellFormula(sheet, "C1", "=A1+1"); err != nil {
		t.Fatalf("set C1 formula: %v", err)
	}

	params, err := parameterizeWorkbookNumbers(f)
	if err != nil {
		t.Fatalf("parameterize: %v", err)
	}

	a1, err := f.GetCellValue(sheet, "A1")
	if err != nil {
		t.Fatalf("get A1: %v", err)
	}
	if !strings.Contains(a1, "{{P") {
		t.Fatalf("expected placeholder in A1, got %q", a1)
	}
	b1, err := f.GetCellValue(sheet, "B1")
	if err != nil {
		t.Fatalf("get B1: %v", err)
	}
	if !strings.Contains(b1, "{{P") {
		t.Fatalf("expected placeholder in B1, got %q", b1)
	}
	formula, err := f.GetCellFormula(sheet, "C1")
	if err != nil {
		t.Fatalf("get C1 formula: %v", err)
	}
	if formula != "=A1+1" {
		t.Fatalf("unexpected formula: %q", formula)
	}

	if err := materializeWorkbookNumbers(f, params); err != nil {
		t.Fatalf("materialize: %v", err)
	}

	a1, err = f.GetCellValue(sheet, "A1")
	if err != nil {
		t.Fatalf("get A1 after: %v", err)
	}
	if a1 != "123" {
		t.Fatalf("unexpected A1 after: %q", a1)
	}
	if ct, err := f.GetCellType(sheet, "A1"); err != nil {
		t.Fatalf("get A1 type: %v", err)
	} else if ct != excelize.CellTypeNumber && ct != excelize.CellTypeUnset {
		t.Fatalf("unexpected A1 type: %v", ct)
	}
	b1, err = f.GetCellValue(sheet, "B1")
	if err != nil {
		t.Fatalf("get B1 after: %v", err)
	}
	if b1 != "同比增长12.5%" {
		t.Fatalf("unexpected B1 after: %q", b1)
	}
}
