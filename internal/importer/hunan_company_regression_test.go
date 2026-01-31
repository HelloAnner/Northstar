package importer

import (
	"path/filepath"
	"testing"

	"northstar/internal/store"
)

func TestImport_PRD_HuaNanCompany_LastYearMonthNotMissing(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	coord := NewCoordinator(st)
	ch := coord.Import(ImportOptions{
		FilePath:         filepath.Join("..", "..", "prd", "12月月报（预估）_补全企业名称社会代码_20260129.xlsx"),
		OriginalFilename: "12月月报（预估）_补全企业名称社会代码_20260129.xlsx",
		ClearExisting:    true,
		UpdateConfigYM:   true,
		CalculateFields:  true,
	})
	for evt := range ch {
		if evt.Type == "error" {
			t.Fatalf("import error: %s", evt.Message)
		}
	}

	name := "华南烟酒测试企业0355有限责任公司"

	var wrCount int
	var wrLY float64
	if err := st.QueryRow(
		"SELECT COUNT(*), COALESCE(MAX(sales_last_year_month), 0) FROM wholesale_retail WHERE name = ?",
		name,
	).Scan(&wrCount, &wrLY); err != nil {
		t.Fatalf("query wr: %v", err)
	}

	var acCount int
	var acGoodsLY float64
	if err := st.QueryRow(
		"SELECT COUNT(*), COALESCE(MAX(goods_last_year_month), 0) FROM accommodation_catering WHERE name = ?",
		name,
	).Scan(&acCount, &acGoodsLY); err != nil {
		t.Fatalf("query ac: %v", err)
	}

	t.Logf("company=%s wrCount=%d wrSalesLY=%.2f acCount=%d acGoodsLY=%.2f", name, wrCount, wrLY, acCount, acGoodsLY)

	if wrCount == 0 && acCount == 0 {
		t.Fatalf("company not found in wr/ac tables")
	}
	if wrCount > 0 && wrLY == 0 {
		t.Fatalf("wr sales_last_year_month missing (0)")
	}
	if acCount > 0 && acGoodsLY == 0 {
		t.Fatalf("ac goods_last_year_month missing (0)")
	}
}
