package importer

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
	"northstar/internal/store"
)

func TestImport_BackfillFromRate_WhenCurrentMonthMissing(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	input := filepath.Join(tmpDir, "rate.xlsx")

	f := excelize.NewFile()
	sheet := "批发"
	f.SetSheetName("Sheet1", sheet)

	headers := []string{
		"统一社会信用代码",
		"单位详细名称",
		"[201-1] 行业代码(GB/T4754-2017)",
		"单位规模",

		"2025年12月销售额",
		"2024年;12月;商品销售额;千元",
		"12月销售额增速",

		"2025年1-12月销售额",
		"2024年;1-12月;商品销售额;千元",
		"1-12月增速",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(sheet, cell, h)
	}

	// 行 2：本年-本月/本年累计为空，但有上年同期、上年累计与增速；应按增速回填
	row2 := []any{
		"AAA",
		"企业A",
		"5101",
		1,

		nil, // 2025年12月销售额（缺失）
		100, // 2024年12月（上年-本月）
		10,  // 12月增速(%) => 2025年12月=110

		nil,  // 2025年1-12月销售额（缺失）
		1000, // 2024年1-12月（上年-1-12）
		20,   // 1-12月增速(%) => 2025年1-12=1200
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
	var salesCum float64
	if err := st.QueryRow(
		"SELECT sales_current_month, sales_current_cumulative FROM wholesale_retail WHERE credit_code = ?",
		"AAA",
	).Scan(&salesMonth, &salesCum); err != nil {
		t.Fatalf("query row: %v", err)
	}

	if salesMonth != 110 {
		if diff := salesMonth - 110; diff < -1e-9 || diff > 1e-9 {
			t.Fatalf("unexpected sales_current_month: %v", salesMonth)
		}
	}
	if salesCum != 1200 {
		if diff := salesCum - 1200; diff < -1e-9 || diff > 1e-9 {
			t.Fatalf("unexpected sales_current_cumulative: %v", salesCum)
		}
	}
}
