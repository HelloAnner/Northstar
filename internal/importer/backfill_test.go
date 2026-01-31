package importer

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
	"northstar/internal/store"
)

func TestImport_BackfillCurrentMonthFromCumulative_WhenZero(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "mini.xlsx")

	f := excelize.NewFile()
	sheet := "批发"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"统一社会信用代码",
		"单位详细名称",
		"[201-1] 行业代码(GB/T4754-2017)",
		"单位规模",
		"粮油食品类",

		"2025年11月销售额",
		"2025年12月销售额",
		"2025年1-11月销售额",
		"2025年1-12月销售额",

		"2025年11月零售额",
		"2025年12月零售额",
		"2025年1-11月零售额",
		"2025年1-12月零售额",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}

	// 行 2：本月为空（会被解析为 0），但累计有值；应回填为累计差
	row2 := []any{
		"AAA",
		"企业A",
		"5101",
		1,
		0,

		10,  // 11月销售额
		nil, // 12月销售额（缺失）
		100, // 1-11
		150, // 1-12

		5,   // 11月零售额
		nil, // 12月零售额（缺失）
		40,  // 1-11
		60,  // 1-12
	}
	for i, v := range row2 {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		_ = f.SetCellValue(sheet, cell, v)
	}

	if err := f.SaveAs(input); err != nil {
		t.Fatalf("save xlsx: %v", err)
	}
	_ = f.Close()

	dbPath := filepath.Join(tmpDir, "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	coordinator := NewCoordinator(st)
	ch := coordinator.Import(ImportOptions{
		FilePath:        input,
		ClearExisting:   true,
		UpdateConfigYM:  true,
		CalculateFields: true,
	})

	for evt := range ch {
		if evt.Type == "error" {
			t.Fatalf("import error event: %s", evt.Message)
		}
	}

	var salesMonth float64
	var retailMonth float64
	if err := st.QueryRow(
		"SELECT sales_current_month, retail_current_month FROM wholesale_retail WHERE credit_code = ?",
		"AAA",
	).Scan(&salesMonth, &retailMonth); err != nil {
		t.Fatalf("query row: %v", err)
	}

	if salesMonth != 50 {
		t.Fatalf("unexpected sales_current_month: %v", salesMonth)
	}
	if retailMonth != 20 {
		t.Fatalf("unexpected retail_current_month: %v", retailMonth)
	}
}

