package model

// IndustryRate 行业增速
type IndustryRate struct {
	MonthRate      float64 `json:"monthRate"`
	CumulativeRate float64 `json:"cumulativeRate"`
}

// Indicators 16项指标数据
type Indicators struct {
	// 指标组一：限上社零额 (4个指标)
	LimitAboveMonthValue      float64 `json:"limitAboveMonthValue"`      // 1. 限上社零额(当月值)
	LimitAboveMonthRate       float64 `json:"limitAboveMonthRate"`       // 2. 限上社零额增速(当月)
	LimitAboveCumulativeValue float64 `json:"limitAboveCumulativeValue"` // 3. 限上社零额(累计值)
	LimitAboveCumulativeRate  float64 `json:"limitAboveCumulativeRate"`  // 4. 限上社零额增速(累计)

	// 指标组二：专项增速 (2个指标)
	EatWearUseMonthRate float64 `json:"eatWearUseMonthRate"` // 5. 吃穿用增速(当月)
	MicroSmallMonthRate float64 `json:"microSmallMonthRate"` // 6. 小微企业增速(当月)

	// 指标组三：四大行业增速 (8个指标)
	IndustryRates map[IndustryType]IndustryRate `json:"industryRates"` // 7-14

	// 指标组四：社零总额 (2个指标)
	TotalSocialCumulativeValue float64 `json:"totalSocialCumulativeValue"` // 15. 社零总额(累计值)
	TotalSocialCumulativeRate  float64 `json:"totalSocialCumulativeRate"`  // 16. 社零总额增速(累计)
}

// NewIndicators 创建默认指标
func NewIndicators() *Indicators {
	return &Indicators{
		IndustryRates: make(map[IndustryType]IndustryRate),
	}
}

// IndicatorSums 预聚合的汇总值
type IndicatorSums struct {
	// 全部企业
	AllRetailCurrent           float64
	AllRetailLastYear          float64
	AllRetailCurrentCumulative float64
	AllRetailLastYearCumulative float64

	// 吃穿用企业
	EatWearUseRetailCurrent  float64
	EatWearUseRetailLastYear float64

	// 小微企业
	MicroSmallRetailCurrent  float64
	MicroSmallRetailLastYear float64

	// 四大行业
	Industries map[IndustryType]*IndustrySums
}

// IndustrySums 行业汇总
type IndustrySums struct {
	SalesCurrent           float64
	SalesLastYear          float64
	SalesCurrentCumulative float64
	SalesLastYearCumulative float64
}

// NewIndicatorSums 创建汇总结构
func NewIndicatorSums() *IndicatorSums {
	return &IndicatorSums{
		Industries: map[IndustryType]*IndustrySums{
			IndustryWholesale:     {},
			IndustryRetail:        {},
			IndustryAccommodation: {},
			IndustryCatering:      {},
		},
	}
}
