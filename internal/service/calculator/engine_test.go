package calculator

import (
	"testing"

	"northstar/internal/model"
	"northstar/internal/service/store"
)

// 创建测试用的企业数据
func createTestCompanies() []*model.Company {
	return []*model.Company{
		{
			ID:                       "c1",
			Name:                     "测试企业1",
			IndustryType:             model.IndustryRetail,
			CompanyScale:             1,
			IsEatWearUse:             true,
			RetailLastYearMonth:      1000,
			RetailCurrentMonth:       1100,
			RetailLastYearCumulative: 10000,
			RetailCurrentCumulative:  11000,
			SalesLastYearMonth:       1200,
			SalesCurrentMonth:        1320,
			SalesLastYearCumulative:  12000,
			SalesCurrentCumulative:   13200,
		},
		{
			ID:                       "c2",
			Name:                     "测试企业2",
			IndustryType:             model.IndustryWholesale,
			CompanyScale:             3, // 小微企业
			IsEatWearUse:             false,
			RetailLastYearMonth:      500,
			RetailCurrentMonth:       525,
			RetailLastYearCumulative: 5000,
			RetailCurrentCumulative:  5250,
			SalesLastYearMonth:       600,
			SalesCurrentMonth:        630,
			SalesLastYearCumulative:  6000,
			SalesCurrentCumulative:   6300,
		},
		{
			ID:                       "c3",
			Name:                     "测试企业3",
			IndustryType:             model.IndustryAccommodation,
			CompanyScale:             4, // 小微企业
			IsEatWearUse:             true,
			RetailLastYearMonth:      300,
			RetailCurrentMonth:       330,
			RetailLastYearCumulative: 3000,
			RetailCurrentCumulative:  3300,
			SalesLastYearMonth:       360,
			SalesCurrentMonth:        396,
			SalesLastYearCumulative:  3600,
			SalesCurrentCumulative:   3960,
		},
		{
			ID:                       "c4",
			Name:                     "测试企业4",
			IndustryType:             model.IndustryCatering,
			CompanyScale:             2,
			IsEatWearUse:             false,
			RetailLastYearMonth:      200,
			RetailCurrentMonth:       180,
			RetailLastYearCumulative: 2000,
			RetailCurrentCumulative:  1800,
			SalesLastYearMonth:       240,
			SalesCurrentMonth:        216,
			SalesLastYearCumulative:  2400,
			SalesCurrentCumulative:   2160,
		},
	}
}

// TestCalcRate 测试增速计算
func TestCalcRate(t *testing.T) {
	tests := []struct {
		name     string
		current  float64
		lastYear float64
		expected float64
	}{
		{"正增长", 1100, 1000, 0.1},
		{"负增长", 900, 1000, -0.1},
		{"零增长", 1000, 1000, 0},
		{"去年为零", 100, 0, 0},
		{"翻倍增长", 2000, 1000, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calcRate(tt.current, tt.lastYear)
			if !floatEquals(result, tt.expected) {
				t.Errorf("calcRate(%v, %v) = %v, want %v", tt.current, tt.lastYear, result, tt.expected)
			}
		})
	}
}

// TestAggregateSums 测试数据聚合
func TestAggregateSums(t *testing.T) {
	memStore := store.NewMemoryStore()
	engine := NewEngine(memStore)

	companies := createTestCompanies()
	for _, c := range companies {
		memStore.AddCompany(c)
	}

	sums := engine.GetSums()

	// 验证全部企业汇总
	expectedAllRetailCurrent := 1100.0 + 525.0 + 330.0 + 180.0
	if !floatEquals(sums.AllRetailCurrent, expectedAllRetailCurrent) {
		t.Errorf("AllRetailCurrent = %v, want %v", sums.AllRetailCurrent, expectedAllRetailCurrent)
	}

	// 验证吃穿用企业汇总 (c1 + c3)
	expectedEatWearUse := 1100.0 + 330.0
	if !floatEquals(sums.EatWearUseRetailCurrent, expectedEatWearUse) {
		t.Errorf("EatWearUseRetailCurrent = %v, want %v", sums.EatWearUseRetailCurrent, expectedEatWearUse)
	}

	// 验证小微企业汇总 (c2 + c3)
	expectedMicroSmall := 525.0 + 330.0
	if !floatEquals(sums.MicroSmallRetailCurrent, expectedMicroSmall) {
		t.Errorf("MicroSmallRetailCurrent = %v, want %v", sums.MicroSmallRetailCurrent, expectedMicroSmall)
	}
}

// TestCalculateIndicators 测试指标计算
func TestCalculateIndicators(t *testing.T) {
	memStore := store.NewMemoryStore()
	engine := NewEngine(memStore)

	companies := createTestCompanies()
	for _, c := range companies {
		memStore.AddCompany(c)
	}

	indicators := engine.Calculate()

	// 验证限上社零额当月数据
	expectedLimitAboveMonth := 1100.0 + 525.0 + 330.0 + 180.0
	if !floatEquals(indicators.LimitAboveMonthValue, expectedLimitAboveMonth) {
		t.Errorf("LimitAboveMonthValue = %v, want %v", indicators.LimitAboveMonthValue, expectedLimitAboveMonth)
	}

	// 验证限上社零额当月增速
	lastYearTotal := 1000.0 + 500.0 + 300.0 + 200.0
	expectedRate := (expectedLimitAboveMonth - lastYearTotal) / lastYearTotal
	if !floatEquals(indicators.LimitAboveMonthRate, expectedRate) {
		t.Errorf("LimitAboveMonthRate = %v, want %v", indicators.LimitAboveMonthRate, expectedRate)
	}
}

// TestEmptyCompanies 测试空数据情况
func TestEmptyCompanies(t *testing.T) {
	memStore := store.NewMemoryStore()
	engine := NewEngine(memStore)

	indicators := engine.Calculate()

	if indicators.LimitAboveMonthValue != 0 {
		t.Errorf("Empty companies should have LimitAboveMonthValue = 0, got %v", indicators.LimitAboveMonthValue)
	}

	if indicators.LimitAboveMonthRate != 0 {
		t.Errorf("Empty companies should have LimitAboveMonthRate = 0, got %v", indicators.LimitAboveMonthRate)
	}
}

// TestIndustryRates 测试行业增速计算
func TestIndustryRates(t *testing.T) {
	memStore := store.NewMemoryStore()
	engine := NewEngine(memStore)

	companies := createTestCompanies()
	for _, c := range companies {
		memStore.AddCompany(c)
	}

	indicators := engine.Calculate()

	// 零售业增速 (c1)
	retailRate := indicators.IndustryRates[model.IndustryRetail]
	expectedRetailMonthRate := (1320.0 - 1200.0) / 1200.0
	if !floatEquals(retailRate.MonthRate, expectedRetailMonthRate) {
		t.Errorf("Retail MonthRate = %v, want %v", retailRate.MonthRate, expectedRetailMonthRate)
	}

	// 餐饮业增速 (c4) - 负增长
	cateringRate := indicators.IndustryRates[model.IndustryCatering]
	expectedCateringMonthRate := (216.0 - 240.0) / 240.0
	if !floatEquals(cateringRate.MonthRate, expectedCateringMonthRate) {
		t.Errorf("Catering MonthRate = %v, want %v", cateringRate.MonthRate, expectedCateringMonthRate)
	}
}

// floatEquals 浮点数近似相等判断
func floatEquals(a, b float64) bool {
	const epsilon = 1e-9
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}
