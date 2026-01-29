package excel_test

import (
	"testing"

	"github.com/xuri/excelize/v2"

	"northstar/internal/service/excel"
)

func TestTemplateExporterWritesValues(t *testing.T) {
	tmpl := buildTemplateWorkbook(t)
	exporter := excel.NewTemplateExporter(tmpl)

	if err := exporter.WriteSummary(excel.SummaryValues{G4: 63041.5}); err != nil {
		t.Fatalf("WriteSummary failed: %v", err)
	}

	got, err := tmpl.GetCellValue("汇总表（定）", "G4")
	if err != nil {
		t.Fatalf("GetCellValue failed: %v", err)
	}
	if got != "63041.5" {
		t.Fatalf("G4=%q, want %q", got, "63041.5")
	}
}

func TestFixedTemplateWorkbookSheetNames(t *testing.T) {
	wb := excel.NewFixedTemplateWorkbook()
	want := []string{
		"批零总表",
		"住餐总表",
		"批发",
		"零售",
		"住宿",
		"餐饮",
		"吃穿用",
		"小微",
		"吃穿用（剔除）",
		"社零额（定）",
		"汇总表（定）",
	}
	if got := wb.GetSheetList(); len(got) != len(want) {
		t.Fatalf("sheet count=%d, want %d, got=%v", len(got), len(want), got)
	}
	for i := range want {
		if got := wb.GetSheetList()[i]; got != want[i] {
			t.Fatalf("sheet[%d]=%q, want %q", i, got, want[i])
		}
	}
}

func buildTemplateWorkbook(t *testing.T) *excelize.File {
	t.Helper()

	wb := excelize.NewFile()
	wb.SetSheetName("Sheet1", "汇总表（定）")
	return wb
}
