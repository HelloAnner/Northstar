package parser

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestSheetRecognizer_DecemberMonthlyReport_20260129(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "prd", "12月月报（预估）_补全企业名称社会代码_20260129.xlsx")
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("open excel: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })

	r := NewSheetRecognizer()
	expect := map[string]SheetType{
		// 快照
		"2024年12月批零": SheetTypeWRSnapshot,
		"2025年11月批零": SheetTypeWRSnapshot,
		"2024年3月":    SheetTypeWRSnapshot,
		"2025年2月":    SheetTypeWRSnapshot,
		"2024年12月住餐": SheetTypeACSnapshot,
		"2025年11月住餐": SheetTypeACSnapshot,
		"2024年3月住":  SheetTypeACSnapshot,
		"2025年2月住":  SheetTypeACSnapshot,

		// 主表
		"批发": SheetTypeWholesale,
		"零售": SheetTypeRetail,
		"住宿": SheetTypeAccommodation,
		"餐饮": SheetTypeCatering,

		// 汇总
		"限上零售额": SheetTypeSummary,
		"小微":    SheetTypeSummary,
		"吃穿用":   SheetTypeSummary,
	}

	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) == 0 {
			t.Fatalf("read sheet %s: %v", sheet, err)
		}
		headers := rows[0]
		res := r.Recognize(sheet, headers)
		if want, ok := expect[sheet]; ok {
			if res.SheetType != want {
				t.Fatalf("sheet %s type mismatch: got=%s conf=%.2f want=%s", sheet, res.SheetType, res.Confidence, want)
			}
		}
	}
}

