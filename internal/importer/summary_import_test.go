/**
 * 汇总表导入测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

package importer

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
	"northstar/internal/store"
)

func TestImport_SavesSummaryTables(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	filePath := filepath.Join(tmp, "import.xlsx")
	if err := buildSummaryWorkbook(filePath); err != nil {
		t.Fatalf("build file failed: %v", err)
	}

	coord := NewCoordinator(st)
	ch := coord.Import(ImportOptions{FilePath: filePath, OriginalFilename: "import.xlsx"})
	for range ch {
		// 等待导入完成
	}

	if count := countTable(t, st, "summary_limit_above_retail"); count == 0 {
		t.Fatalf("summary_limit_above_retail not saved")
	}
}

func buildSummaryWorkbook(path string) error {
	f := excelize.NewFile()
	whSheet := "批发"
	f.SetSheetName("Sheet1", whSheet)
	headers := []string{
		"统一社会信用代码",
		"单位详细名称",
		"[201-1] 行业代码(GB/T4754-2017)",
		"2025年12月销售额",
		"2024年;12月;商品销售额;千元",
		"2025年1-12月销售额",
		"2024年;1-12月;商品销售额;千元",
		"2025年12月零售额",
		"2024年;12月;商品零售额;千元",
		"2025年1-12月零售额",
		"2024年;1-12月;商品零售额;千元",
		"单位规模",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(whSheet, cell, h)
	}
	values := []interface{}{
		"913000000000000000",
		"测试企业",
		"5101",
		100,
		90,
		1100,
		1000,
		80,
		70,
		900,
		800,
		2,
	}
	for i, v := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		_ = f.SetCellValue(whSheet, cell, v)
	}

	summary := "限上零售额"
	_, _ = f.NewSheet(summary)
	_ = f.SetCellValue(summary, "B1", 45839)
	_ = f.SetCellValue(summary, "C1", 45474)
	_ = f.SetCellValue(summary, "D1", "增速")
	_ = f.SetCellValue(summary, "A2", "批发")
	_ = f.SetCellValue(summary, "B2", 100)
	_ = f.SetCellValue(summary, "C2", 90)
	_ = f.SetCellValue(summary, "D2", 11.11)

	return f.SaveAs(path)
}
