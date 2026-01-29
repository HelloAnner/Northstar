package excel_test

import (
	"testing"

	"github.com/xuri/excelize/v2"

	"northstar/internal/model"
	"northstar/internal/service/excel"
)

func TestRecognizeSheetTypes(t *testing.T) {
	wb := buildWorkbookWithHeaders(t, map[string][]string{
		"批发": {
			"统一社会信用代码",
			"单位详细名称",
			"[201-1] 行业代码(GB/T4754-2017)",
			"2025年12月销售额",
			"2025年12月零售额",
			"2025年1-12月销售额",
			"2025年1-12月零售额",
			"2024年;12月;商品销售额;千元",
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
		"吃穿用": {
			"处理地编码",
			"统一社会信用代码",
			"单位详细名称",
			"201-1-行业代码（GB/T4754-2017）",
			"商品销售额;本年-本月",
			"零售额;本年-本月",
			"201-1-单位规模",
			"其中：通过公共网络实现的商品销售额;本年-本月",
		},
		"小微": {
			"处理地编码",
			"统一社会信用代码",
			"单位详细名称",
			"(衍生指标)计算用小微当月零售额",
			"(衍生指标)计算用小微上年同月零售额",
		},
	})

	rec := excel.NewRecognizer()
	result := rec.RecognizeWorkbook(wb)

	if got, want := result["批发"].Type, model.SheetTypeWholesaleMain; got != want {
		t.Fatalf("sheet 批发 type=%s, want %s", got, want)
	}
	if got, want := result["零售"].Type, model.SheetTypeRetailMain; got != want {
		t.Fatalf("sheet 零售 type=%s, want %s", got, want)
	}
	if got, want := result["2025年11月批零"].Type, model.SheetTypeWholesaleRetailSnapshot; got != want {
		t.Fatalf("sheet 2025年11月批零 type=%s, want %s", got, want)
	}
	if got, want := result["吃穿用"].Type, model.SheetTypeEatWearUse; got != want {
		t.Fatalf("sheet 吃穿用 type=%s, want %s", got, want)
	}
	if got, want := result["小微"].Type, model.SheetTypeMicroSmall; got != want {
		t.Fatalf("sheet 小微 type=%s, want %s", got, want)
	}
}

func buildWorkbookWithHeaders(t *testing.T, sheets map[string][]string) *excelize.File {
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
		cell := "A1"
		if err := wb.SetSheetRow(name, cell, &row); err != nil {
			t.Fatalf("SetSheetRow %s failed: %v", name, err)
		}
	}

	return wb
}
