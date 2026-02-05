/**
 * raw 导入采集测试
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

func TestImport_SavesRawSheets(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	filePath := filepath.Join(tmp, "import.xlsx")
	if err := buildMinimalWholesaleFile(filePath); err != nil {
		t.Fatalf("build file failed: %v", err)
	}

	coord := NewCoordinator(st)
	ch := coord.Import(ImportOptions{FilePath: filePath, OriginalFilename: "import.xlsx"})
	for range ch {
		// 等待导入完成
	}

	if count := countTable(t, st, "sheet_columns"); count == 0 {
		t.Fatalf("sheet_columns not saved")
	}
	if count := countTable(t, st, "sheet_rows"); count == 0 {
		t.Fatalf("sheet_rows not saved")
	}
	if count := countTable(t, st, "sheet_cells"); count == 0 {
		t.Fatalf("sheet_cells not saved")
	}
}

func buildMinimalWholesaleFile(path string) error {
	f := excelize.NewFile()
	sheet := "批发"
	f.SetSheetName("Sheet1", sheet)
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
		_ = f.SetCellValue(sheet, cell, h)
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
		_ = f.SetCellValue(sheet, cell, v)
	}
	return f.SaveAs(path)
}

func countTable(t *testing.T, st *store.Store, table string) int {
	row := st.QueryRow("SELECT COUNT(*) FROM " + table)
	var cnt int
	if err := row.Scan(&cnt); err != nil {
		t.Fatalf("count %s failed: %v", table, err)
	}
	return cnt
}
