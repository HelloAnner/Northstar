package parser

import "testing"

func TestExtractYearMonthRange_Standard(t *testing.T) {
	t.Parallel()

	year, start, end, found := ExtractYearMonthRange("2025年1-11月销售额")
	if !found {
		t.Fatalf("expected found")
	}
	if year != 2025 || start != 1 || end != 11 {
		t.Fatalf("unexpected ysm: %d %d %d", year, start, end)
	}
}

func TestInferFieldTimeType_Sales_YearMonthVsRange(t *testing.T) {
	t.Parallel()

	currentYear := 2025
	currentMonth := 12

	if got := InferFieldTimeType("2025年11月销售额", currentYear, currentMonth); got != PrevMonth {
		t.Fatalf("2025年11月销售额 want=%v got=%v", PrevMonth, got)
	}
	if got := InferFieldTimeType("2025年12月销售额", currentYear, currentMonth); got != CurrentMonth {
		t.Fatalf("2025年12月销售额 want=%v got=%v", CurrentMonth, got)
	}
	if got := InferFieldTimeType("2025年1-11月销售额", currentYear, currentMonth); got != PrevCumulative {
		t.Fatalf("2025年1-11月销售额 want=%v got=%v", PrevCumulative, got)
	}
	if got := InferFieldTimeType("2025年1-12月销售额", currentYear, currentMonth); got != CurrentCumulative {
		t.Fatalf("2025年1-12月销售额 want=%v got=%v", CurrentCumulative, got)
	}
	if got := InferFieldTimeType("2024年12月销售额", currentYear, currentMonth); got != LastYearMonth {
		t.Fatalf("2024年12月销售额 want=%v got=%v", LastYearMonth, got)
	}
	if got := InferFieldTimeType("2024年1-11月销售额", currentYear, currentMonth); got != LastYearPrevCumulative {
		t.Fatalf("2024年1-11月销售额 want=%v got=%v", LastYearPrevCumulative, got)
	}
	if got := InferFieldTimeType("2024年1-12月销售额", currentYear, currentMonth); got != LastYearCumulative {
		t.Fatalf("2024年1-12月销售额 want=%v got=%v", LastYearCumulative, got)
	}
}

func TestFindCurrentYearMonth_RangeOnly(t *testing.T) {
	t.Parallel()

	year, month := FindCurrentYearMonth([]string{
		"统一社会信用代码",
		"单位详细名称",
		"2024年1-11月销售额",
		"2025年1-11月销售额",
	})
	if year != 2025 || month != 11 {
		t.Fatalf("unexpected ym: %d-%02d", year, month)
	}
}

func TestInferFieldTimeType_JanuaryPrevMonth(t *testing.T) {
	t.Parallel()

	currentYear := 2026
	currentMonth := 1

	if got := InferFieldTimeType("2025年12月销售额", currentYear, currentMonth); got != PrevMonth {
		t.Fatalf("2025年12月销售额 want=%v got=%v", PrevMonth, got)
	}
	if got := InferFieldTimeType("12月销售额", currentYear, currentMonth); got != PrevMonth {
		t.Fatalf("12月销售额 want=%v got=%v", PrevMonth, got)
	}
}
