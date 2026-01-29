package excel_test

import (
	"testing"

	"northstar/internal/service/excel"
)

func TestParseCanonicalFromWorkbook(t *testing.T) {
	wb := buildWorkbookWithHeadersForResolve(t, map[string][]string{
		"批发": {
			"统一社会信用代码",
			"单位详细名称",
			"[201-1] 行业代码(GB/T4754-2017)",
			"单位规模",
			"2025年12月销售额",
			"2024年;12月;商品销售额;千元",
			"2025年1-12月销售额",
			"2024年;1-12月;商品销售额;千元",
			"2025年12月零售额",
			"2024年;12月;商品零售额;千元",
			"2025年1-12月零售额",
			"2024年;1-12月;商品零售额;千元",
		},
	})

	row2 := []interface{}{
		"91330000TEST000001",
		"测试企业1",
		"5111",
		"3",
		"1000",
		"800",
		"12000",
		"10000",
		"400",
		"300",
		"4800",
		"3600",
	}
	if err := wb.SetSheetRow("批发", "A2", &row2); err != nil {
		t.Fatalf("SetSheetRow failed: %v", err)
	}

	result, err := excel.ParseCanonical(wb, excel.ParseOptions{Month: 12})
	if err != nil {
		t.Fatalf("ParseCanonical failed: %v", err)
	}
	if result == nil {
		t.Fatalf("ParseCanonical result is nil")
	}
	if len(result.Companies) == 0 {
		t.Fatalf("Companies should not be empty")
	}

	c := result.Companies[0]
	if c.CreditCode != "91330000TEST000001" {
		t.Fatalf("CreditCode=%q", c.CreditCode)
	}
	if c.Retail.CurrentMonth != 400 {
		t.Fatalf("Retail.CurrentMonth=%v, want 400", c.Retail.CurrentMonth)
	}
	if c.Sales.CurrentMonth != 1000 {
		t.Fatalf("Sales.CurrentMonth=%v, want 1000", c.Sales.CurrentMonth)
	}
}

func TestParseCanonical_InferMonthFromCumulativeDelta_WholesaleRetail(t *testing.T) {
	wb := buildWorkbookWithHeadersForResolve(t, map[string][]string{
		"批发": {
			"统一社会信用代码",
			"单位详细名称",
			"[201-1] 行业代码(GB/T4754-2017)",
			"单位规模",
			"2025年12月销售额",
			"2024年;12月;商品销售额;千元",
			"2025年1-11月销售额",
			"2025年1-12月销售额",
			"2024年;1-12月;商品销售额;千元",
			"2025年12月零售额",
			"2024年;12月;商品零售额;千元",
			"2025年1-11月零售额",
			"2025年1-12月零售额",
			"2024年;1-12月;商品零售额;千元",
		},
	})

	// 当月值留空，通过 “1-12累计 - 1-11累计” 推导当月值
	row2 := []interface{}{
		"91330000TEST000002",
		"测试企业2",
		"5111",
		"3",
		"",     // 2025年12月销售额
		"848",  // 2024年;12月;商品销售额;千元
		"11289", // 2025年1-11月销售额
		"11782", // 2025年1-12月销售额
		"13698", // 2024年;1-12月;商品销售额;千元
		"",      // 2025年12月零售额
		"251",   // 2024年;12月;商品零售额;千元
		"3066",  // 2025年1-11月零售额
		"3066",  // 2025年1-12月零售额（当月推导为0）
		"4403",  // 2024年;1-12月;商品零售额;千元
	}
	if err := wb.SetSheetRow("批发", "A2", &row2); err != nil {
		t.Fatalf("SetSheetRow failed: %v", err)
	}

	result, err := excel.ParseCanonical(wb, excel.ParseOptions{Month: 12})
	if err != nil {
		t.Fatalf("ParseCanonical failed: %v", err)
	}
	c := result.Companies[0]
	if got, want := c.Sales.CurrentMonth, float64(493); got != want {
		t.Fatalf("Sales.CurrentMonth=%v, want %v", got, want)
	}
	if got, want := c.Sales.CurrentCumulative, float64(11782); got != want {
		t.Fatalf("Sales.CurrentCumulative=%v, want %v", got, want)
	}
	if got, want := c.Retail.CurrentMonth, float64(0); got != want {
		t.Fatalf("Retail.CurrentMonth=%v, want %v", got, want)
	}
}

func TestParseCanonical_InferMonthFromCumulativeDelta_AccommodationCatering(t *testing.T) {
	wb := buildWorkbookWithHeadersForResolve(t, map[string][]string{
		"住宿": {
			"统一社会信用代码",
			"单位详细名称",
			"[201-1] 行业代码(GB/T4754-2017)",
			"2025年12月营业额",
			"2024年12月;营业额总计;千元",
			"2025年1-11月营业额",
			"2025年1-12月营业额",
			"2024年1-12月;营业额总计;千元",
			"2025年12月客房收入",
			"2024年12月;营业额总计;客房收入;千元",
			"2025年1-11月客房收入",
			"2025年1-12月客房收入",
			"2024年1-12月;营业额总计;客房收入;千元",
			"2025年12月餐费收入",
			"2024年12月;营业额总计;餐费收入;千元",
			"2025年1-11月餐费收入",
			"1-12月餐费收入",
			"2024年1-12月;营业额总计;餐费收入;千元",
			"2025年12月销售额",
			"2024年12月;营业额总计;商品销售额;千元",
			"2025年1-11月销售额",
			"1-12月销售额",
			"2024年1-12月;营业额总计;商品销售额;千元",
		},
	})

	row2 := []interface{}{
		"91330000TEST000003",
		"测试企业3",
		"6511",
		"",    // 2025年12月营业额
		"50",  // 2024年12月;营业额总计;千元
		"100", // 2025年1-11月营业额
		"120", // 2025年1-12月营业额 -> 推导当月 20
		"900",
		"",   // 2025年12月客房收入
		"30", // 2024年12月;营业额总计;客房收入;千元
		"70", // 2025年1-11月客房收入
		"80", // 2025年1-12月客房收入 -> 推导当月 10
		"600",
		"",   // 2025年12月餐费收入
		"10", // 2024年12月;营业额总计;餐费收入;千元
		"20", // 2025年1-11月餐费收入
		"35", // 1-12月餐费收入 -> 推导当月 15
		"200",
		"",    // 2025年12月销售额
		"8",   // 2024年12月;营业额总计;商品销售额;千元
		"10",  // 2025年1-11月销售额
		"20",  // 1-12月销售额 -> 推导当月 10
		"150",
	}
	if err := wb.SetSheetRow("住宿", "A2", &row2); err != nil {
		t.Fatalf("SetSheetRow failed: %v", err)
	}

	result, err := excel.ParseCanonical(wb, excel.ParseOptions{Month: 12})
	if err != nil {
		t.Fatalf("ParseCanonical failed: %v", err)
	}
	c := result.Companies[0]
	if got, want := c.Revenue.CurrentMonth, float64(20); got != want {
		t.Fatalf("Revenue.CurrentMonth=%v, want %v", got, want)
	}
	if got, want := c.RoomRevenue.CurrentMonth, float64(10); got != want {
		t.Fatalf("RoomRevenue.CurrentMonth=%v, want %v", got, want)
	}
	if got, want := c.FoodRevenue.CurrentMonth, float64(15); got != want {
		t.Fatalf("FoodRevenue.CurrentMonth=%v, want %v", got, want)
	}
	if got, want := c.GoodsSales.CurrentMonth, float64(10); got != want {
		t.Fatalf("GoodsSales.CurrentMonth=%v, want %v", got, want)
	}
}
