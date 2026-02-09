/**
 * 模板输入单元格逻辑测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

package exporter

import (
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"northstar/internal/model"
	"northstar/internal/store"
)

func TestFillSocialRetailSheet_PlatformRetailInputs(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	// 2025-12
	if err := st.Exec(`
		INSERT INTO wholesale_retail (credit_code, name, industry_code, industry_type, company_scale, row_no, data_year, data_month, retail_current_month, source_sheet, source_file)
		VALUES ('WR001', '批发企业', '5101', 'wholesale', 1, 1, 2025, 12, 100.4, '', '')
	`); err != nil {
		t.Fatalf("insert wr 2025-12 failed: %v", err)
	}
	if err := st.Exec(`
		INSERT INTO accommodation_catering (credit_code, name, industry_code, industry_type, company_scale, row_no, data_year, data_month, food_current_month, goods_current_month, source_sheet, source_file)
		VALUES ('AC001', '住宿企业', '6101', 'accommodation', 1, 1, 2025, 12, 10.6, 20.4, '', '')
	`); err != nil {
		t.Fatalf("insert ac 2025-12 failed: %v", err)
	}

	// 2025-11
	if err := st.Exec(`
		INSERT INTO wholesale_retail (credit_code, name, industry_code, industry_type, company_scale, row_no, data_year, data_month, retail_current_month, source_sheet, source_file)
		VALUES ('WR001', '批发企业', '5101', 'wholesale', 1, 1, 2025, 11, 80.2, '', '')
	`); err != nil {
		t.Fatalf("insert wr 2025-11 failed: %v", err)
	}
	if err := st.Exec(`
		INSERT INTO accommodation_catering (credit_code, name, industry_code, industry_type, company_scale, row_no, data_year, data_month, food_current_month, goods_current_month, source_sheet, source_file)
		VALUES ('AC001', '住宿企业', '6101', 'accommodation', 1, 1, 2025, 11, 10.2, 5.7, '', '')
	`); err != nil {
		t.Fatalf("insert ac 2025-11 failed: %v", err)
	}

	// 2024-12
	if err := st.Exec(`
		INSERT INTO wholesale_retail (credit_code, name, industry_code, industry_type, company_scale, row_no, data_year, data_month, retail_current_month, source_sheet, source_file)
		VALUES ('WR001', '批发企业', '5101', 'wholesale', 1, 1, 2024, 12, 90.7, '', '')
	`); err != nil {
		t.Fatalf("insert wr 2024-12 failed: %v", err)
	}
	if err := st.Exec(`
		INSERT INTO accommodation_catering (credit_code, name, industry_code, industry_type, company_scale, row_no, data_year, data_month, food_current_month, goods_current_month, source_sheet, source_file)
		VALUES ('AC001', '住宿企业', '6101', 'accommodation', 1, 1, 2024, 12, 4.2, 5.2, '', '')
	`); err != nil {
		t.Fatalf("insert ac 2024-12 failed: %v", err)
	}

	// 2024-11
	if err := st.Exec(`
		INSERT INTO wholesale_retail (credit_code, name, industry_code, industry_type, company_scale, row_no, data_year, data_month, retail_current_month, source_sheet, source_file)
		VALUES ('WR001', '批发企业', '5101', 'wholesale', 1, 1, 2024, 11, 70.2, '', '')
	`); err != nil {
		t.Fatalf("insert wr 2024-11 failed: %v", err)
	}
	if err := st.Exec(`
		INSERT INTO accommodation_catering (credit_code, name, industry_code, industry_type, company_scale, row_no, data_year, data_month, food_current_month, goods_current_month, source_sheet, source_file)
		VALUES ('AC001', '住宿企业', '6101', 'accommodation', 1, 1, 2024, 11, 3.3, 4.4, '', '')
	`); err != nil {
		t.Fatalf("insert ac 2024-11 failed: %v", err)
	}

	f, err := OpenEmbeddedMonthReportTemplate()
	if err != nil {
		t.Fatalf("open template failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	idx := indicatorIndex{
		"小微企业增速_当月": {ID: "小微企业增速_当月", Value: 6},
		"吃穿用增速_当月": {ID: "吃穿用增速_当月", Value: 8},
	}

	if err := fillSocialRetailSheetAndMaterialize(f, st, 2025, 12, idx); err != nil {
		t.Fatalf("fill social retail failed: %v", err)
	}

	k4, _ := getCellFloat(f, "社零额（定）", "K4")
	k6, _ := getCellFloat(f, "社零额（定）", "K6")
	k13, _ := getCellFloat(f, "社零额（定）", "K13")
	k16, _ := getCellFloat(f, "社零额（定）", "K16")

	if k16 != 131 {
		t.Fatalf("K16 mismatch: %v", k16)
	}
	if k4 != 96 {
		t.Fatalf("K4 mismatch: %v", k4)
	}
	if k13 != 100 {
		t.Fatalf("K13 mismatch: %v", k13)
	}
	if k6 != 78 {
		t.Fatalf("K6 mismatch: %v", k6)
	}
}

func TestFillSocialRetailSheet_TitleUsesYearMonth(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	f, err := OpenEmbeddedMonthReportTemplate()
	if err != nil {
		t.Fatalf("open template failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	year := 2025
	month := 12
	if err := fillSocialRetailSheetAndMaterialize(f, st, year, month, indicatorIndex{}); err != nil {
		t.Fatalf("fill social retail failed: %v", err)
	}

	title, _ := getCellString(f, "社零额（定）", "A1")
	expect := "社零额计算举例（" + strconv.Itoa(year) + "年" + strconv.Itoa(month) + "月数据）"
	if title != expect {
		t.Fatalf("unexpected title: %q", title)
	}
}

func TestFillSocialRetailSheet_RewriteFormulaYear(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	f, err := OpenEmbeddedMonthReportTemplate()
	if err != nil {
		t.Fatalf("open template failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	year := 2026
	month := 3
	if err := fillSocialRetailSheetAndMaterialize(f, st, year, month, indicatorIndex{}); err != nil {
		t.Fatalf("fill social retail failed: %v", err)
	}

	j3, err := f.GetCellFormula("社零额（定）", "J3")
	if err != nil {
		t.Fatalf("get J3 formula failed: %v", err)
	}
	expectJ3 := "\"2026年\"&J2-1&\"月社零额（当月累计-上月累计）\""
	if j3 != expectJ3 {
		t.Fatalf("unexpected J3 formula: %q", j3)
	}

	j5, err := f.GetCellFormula("社零额（定）", "J5")
	if err != nil {
		t.Fatalf("get J5 formula failed: %v", err)
	}
	expectJ5 := "\"2025年\"&J2-1&\"月社零额（当月累计-上月累计）\""
	if j5 != expectJ5 {
		t.Fatalf("unexpected J5 formula: %q", j5)
	}
}

func TestRewriteFixedSummarySheet_UsesFormulaText(t *testing.T) {
	f, err := OpenEmbeddedMonthReportTemplate()
	if err != nil {
		t.Fatalf("open template failed: %v", err)
	}
	defer func() { _ = f.Close() }()

	wh := wrSums{salesCur: 110, salesLast: 100, salesCurCum: 200, salesLastCum: 180, retailCur: 1000, retailLast: 900, retailCurCum: 3000, retailLastCum: 2700}
	re := wrSums{salesCur: 220, salesLast: 200, salesCurCum: 400, salesLastCum: 360, retailCur: 2000, retailLast: 1800, retailCurCum: 5000, retailLastCum: 4500}
	acc := wrSums{salesCur: 55, salesLast: 50, salesCurCum: 120, salesLastCum: 100, retailCur: 300, retailLast: 250, retailCurCum: 800, retailLastCum: 700}
	cat := wrSums{salesCur: 66, salesLast: 60, salesCurCum: 150, salesLastCum: 140, retailCur: 400, retailLast: 350, retailCurCum: 900, retailLastCum: 800}

	wrRecords := []*model.WholesaleRetail{{SalesCurrentMonth: 110, SalesLastYearMonth: 100}}
	acRecords := []*model.AccommodationCatering{{RevenueCurrentMonth: 55, RevenueLastYearMonth: 50}}

	indicators := indicatorIndex{
		"吃穿用增速_当月":        {ID: "吃穿用增速_当月", Value: 12},
		"小微企业增速_当月":        {ID: "小微企业增速_当月", Value: 8},
		"社零总额_累计值": {ID: "社零总额_累计值", Value: 1234567},
		"社零总额增速_累计":  {ID: "社零总额增速_累计", Value: 6.5},
	}

	if err := rewriteFixedSummarySheet(f, 2025, 12, wh, re, acc, cat, indicators, wrRecords, acRecords); err != nil {
		t.Fatalf("rewrite summary failed: %v", err)
	}

	title, _ := getCellString(f, "汇总表（定）", "A1")
	if title != "2025年1-12月限上单位上报情况表" {
		t.Fatalf("unexpected title: %q", title)
	}

	summary, _ := getCellString(f, "汇总表（定）", "X3")
	if !strings.Contains(summary, "社会消费品零售总额：全县2家限上商贸单位已全部上报。") {
		t.Fatalf("summary text missing header: %q", summary)
	}
	if !strings.Contains(summary, "1月，批发、零售、住宿、餐饮业销售额(营业额)同比分别增长") {
		t.Fatalf("summary text missing month: %q", summary)
	}
	if !strings.Contains(summary, "1-12") {
		t.Fatalf("summary text missing period: %q", summary)
	}
}
