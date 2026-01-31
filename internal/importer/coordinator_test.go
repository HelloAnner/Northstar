package importer

import (
	"path/filepath"
	"testing"

	"northstar/internal/parser"
	"northstar/internal/store"
)

func TestImport_DecemberMonthlyReport_20260129(t *testing.T) {
	t.Parallel()

	input := filepath.Join("..", "..", "prd", "12月月报（预估）_补全企业名称社会代码_20260129.xlsx")

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
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

	var report any
	recognized := map[string]string{}
	for evt := range ch {
		if evt.Type == "error" {
			t.Fatalf("import error event: %s", evt.Message)
		}
		if evt.Type == "info" {
			if m, ok := evt.Data.(map[string]interface{}); ok {
				sn, _ := m["sheet_name"].(string)
				st, _ := m["sheet_type"].(string)
				if sn != "" && st != "" {
					recognized[sn] = st
				}
			}
		}
		if evt.Type == "done" {
			report = evt.Data
		}
	}

	if report == nil {
		t.Fatalf("missing done report")
	}

	r, ok := report.(*parser.ImportReport)
	if !ok {
		t.Fatalf("unexpected report type: %T", report)
	}

	if r.TotalSheets != 15 {
		t.Fatalf("unexpected total sheets: %d, sheets=%v", r.TotalSheets, collectSheetStatuses(r))
	}
	if r.ImportedSheets != 12 {
		t.Fatalf("unexpected imported sheets: %d, sheets=%v, errors=%v, recognized=%v",
			r.ImportedSheets, collectSheetStatuses(r), collectSheetErrors(r), recognized)
	}
	if r.SkippedSheets != 3 {
		t.Fatalf("unexpected skipped sheets: %d, sheets=%v", r.SkippedSheets, collectSheetStatuses(r))
	}

	for _, s := range r.Sheets {
		if s.Status == "error" {
			t.Fatalf("sheet %s parse error: %v", s.SheetName, s.Errors)
		}
	}

	year, month, err := st.GetCurrentYearMonth()
	if err != nil {
		t.Fatalf("get current ym: %v", err)
	}
	if year != 2025 || month != 12 {
		t.Fatalf("unexpected current ym: %d-%02d", year, month)
	}

	wholesale := "wholesale"
	retail := "retail"
	accommodation := "accommodation"
	catering := "catering"

	wrWholesaleCount, err := st.CountWR(store.WRQueryOptions{DataYear: &year, DataMonth: &month, IndustryType: &wholesale})
	if err != nil {
		t.Fatalf("count wr wholesale: %v", err)
	}
	wrRetailCount, err := st.CountWR(store.WRQueryOptions{DataYear: &year, DataMonth: &month, IndustryType: &retail})
	if err != nil {
		t.Fatalf("count wr retail: %v", err)
	}
	acAccommodationCount, err := st.CountAC(store.ACQueryOptions{DataYear: &year, DataMonth: &month, IndustryType: &accommodation})
	if err != nil {
		t.Fatalf("count ac accommodation: %v", err)
	}
	acCateringCount, err := st.CountAC(store.ACQueryOptions{DataYear: &year, DataMonth: &month, IndustryType: &catering})
	if err != nil {
		t.Fatalf("count ac catering: %v", err)
	}

	if wrWholesaleCount == 0 || wrRetailCount == 0 || acAccommodationCount == 0 || acCateringCount == 0 {
		t.Fatalf("industry_type not populated: wr(wholesale=%d retail=%d) ac(accommodation=%d catering=%d) distinct=%v samples=%v",
			wrWholesaleCount, wrRetailCount, acAccommodationCount, acCateringCount, distinctIndustryTypes(t, st), sampleIndustryRows(t, st))
	}

	// 元数据入库：每个 Sheet 都应有一条 sheets_meta
	var sheetMetaCount int
	if err := st.QueryRow("SELECT COUNT(*) FROM sheets_meta").Scan(&sheetMetaCount); err != nil {
		t.Fatalf("count sheets_meta: %v", err)
	}
	if sheetMetaCount != 15 {
		t.Fatalf("unexpected sheets_meta count: %d", sheetMetaCount)
	}

	var importLogCount int
	if err := st.QueryRow("SELECT COUNT(*) FROM import_logs").Scan(&importLogCount); err != nil {
		t.Fatalf("count import_logs: %v", err)
	}
	if importLogCount != 1 {
		t.Fatalf("unexpected import_logs count: %d", importLogCount)
	}
}

func collectSheetStatuses(r *parser.ImportReport) map[string]string {
	out := make(map[string]string, len(r.Sheets))
	for _, s := range r.Sheets {
		out[s.SheetName] = s.Status
	}
	return out
}

func distinctIndustryTypes(t *testing.T, st *store.Store) map[string]map[string]int {
	t.Helper()
	out := map[string]map[string]int{
		"wholesale_retail": {},
		"accommodation_catering": {},
	}

	rows, err := st.Query("SELECT industry_type, COUNT(*) FROM wholesale_retail GROUP BY industry_type")
	if err != nil {
		out["wholesale_retail"]["<query_error>"] = 1
	} else {
		defer rows.Close()
		for rows.Next() {
			var k string
			var c int
			_ = rows.Scan(&k, &c)
			out["wholesale_retail"][k] = c
		}
	}

	rows, err = st.Query("SELECT industry_type, COUNT(*) FROM accommodation_catering GROUP BY industry_type")
	if err != nil {
		out["accommodation_catering"]["<query_error>"] = 1
	} else {
		defer rows.Close()
		for rows.Next() {
			var k string
			var c int
			_ = rows.Scan(&k, &c)
			out["accommodation_catering"][k] = c
		}
	}

	return out
}

func sampleIndustryRows(t *testing.T, st *store.Store) map[string][]map[string]string {
	t.Helper()
	out := map[string][]map[string]string{
		"wholesale_retail": {},
		"accommodation_catering": {},
	}

	rows, err := st.Query("SELECT name, industry_code, industry_type, source_sheet FROM wholesale_retail ORDER BY id LIMIT 5")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name, code, typ, sheet string
			_ = rows.Scan(&name, &code, &typ, &sheet)
			out["wholesale_retail"] = append(out["wholesale_retail"], map[string]string{
				"name": name, "industry_code": code, "industry_type": typ, "source_sheet": sheet,
			})
		}
	}

	rows, err = st.Query("SELECT name, industry_code, industry_type, source_sheet FROM accommodation_catering ORDER BY id LIMIT 5")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name, code, typ, sheet string
			_ = rows.Scan(&name, &code, &typ, &sheet)
			out["accommodation_catering"] = append(out["accommodation_catering"], map[string]string{
				"name": name, "industry_code": code, "industry_type": typ, "source_sheet": sheet,
			})
		}
	}

	return out
}

func collectSheetErrors(r *parser.ImportReport) map[string][]string {
	out := make(map[string][]string)
	for _, s := range r.Sheets {
		if s.Status == "error" && len(s.Errors) > 0 {
			out[s.SheetName] = s.Errors
		}
	}
	return out
}
