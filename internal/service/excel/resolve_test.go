package excel_test

import (
	"testing"

	"github.com/xuri/excelize/v2"

	"northstar/internal/model"
	"northstar/internal/service/excel"
)

func TestResolveWorkbookForMonth(t *testing.T) {
	wb := buildWorkbookWithHeadersForResolve(t, map[string][]string{
		"批发": {
			"统一社会信用代码",
			"单位详细名称",
			"[201-1] 行业代码(GB/T4754-2017)",
			"2025年12月销售额",
			"2025年12月零售额",
			"2025年1-12月销售额",
			"2025年1-12月零售额",
		},
		"零售": {
			"统一社会信用代码",
			"单位详细名称",
			"[201-1] 行业代码(GB/T4754-2017)",
			"2025年12月销售额",
			"2025年12月零售额",
			"2025年1-12月销售额",
			"2025年1-12月零售额",
		},
		"2025年11月批零": {
			"统一社会信用代码",
			"单位详细名称",
			"201-1-行业代码（GB/T4754-2017）",
			"商品销售额;本年-本月",
			"商品销售额;本年-1—本月",
			"零售额;本年-本月",
			"零售额;本年-1—本月",
			"201-1-单位规模",
		},
		"2025年12月批零": {
			"统一社会信用代码",
			"单位详细名称",
			"201-1-行业代码（GB/T4754-2017）",
			"商品销售额;本年-本月",
			"商品销售额;本年-1—本月",
			"零售额;本年-本月",
			"零售额;本年-1—本月",
			"201-1-单位规模",
		},
	})

	result := excel.ResolveWorkbook(wb, excel.ResolveOptions{
		Month: 12,
	})

	if got, want := result.MainSheets[model.SheetTypeWholesaleMain], "批发"; got != want {
		t.Fatalf("MainSheets[wholesale_main]=%q, want %q", got, want)
	}
	if got, want := result.MainSheets[model.SheetTypeRetailMain], "零售"; got != want {
		t.Fatalf("MainSheets[retail_main]=%q, want %q", got, want)
	}
	if got, want := result.SnapshotSheets[model.SheetTypeWholesaleRetailSnapshot], "2025年12月批零"; got != want {
		t.Fatalf("SnapshotSheets[wholesale_retail_snapshot]=%q, want %q", got, want)
	}
}

func TestResolveWorkbookHonorsOverrides(t *testing.T) {
	wb := buildWorkbookWithHeadersForResolve(t, map[string][]string{
		"批发": {
			"统一社会信用代码",
			"单位详细名称",
			"[201-1] 行业代码(GB/T4754-2017)",
			"2025年12月销售额",
			"2025年12月零售额",
			"2025年1-12月销售额",
			"2025年1-12月零售额",
		},
		"批发(确认)": {
			"统一社会信用代码",
			"单位详细名称",
		},
	})

	result := excel.ResolveWorkbook(wb, excel.ResolveOptions{
		Month: 12,
		Overrides: map[string]model.SheetType{
			"批发(确认)": model.SheetTypeWholesaleMain,
		},
	})

	if got, want := result.MainSheets[model.SheetTypeWholesaleMain], "批发(确认)"; got != want {
		t.Fatalf("override wholesale_main=%q, want %q", got, want)
	}
}

func buildWorkbookWithHeadersForResolve(t *testing.T, sheets map[string][]string) *excelize.File {
	t.Helper()

	wb := excelize.NewFile()
	defaultSheet := wb.GetSheetName(wb.GetActiveSheetIndex())
	if defaultSheet != "" {
		_ = wb.DeleteSheet(defaultSheet)
	}

	for name, headers := range sheets {
		wb.NewSheet(name)
		row := make([]interface{}, 0, len(headers))
		for _, h := range headers {
			row = append(row, h)
		}
		if err := wb.SetSheetRow(name, "A1", &row); err != nil {
			t.Fatalf("SetSheetRow %s failed: %v", name, err)
		}
	}

	return wb
}
