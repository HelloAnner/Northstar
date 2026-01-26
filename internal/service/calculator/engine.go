package calculator

import (
	"northstar/internal/model"
	"northstar/internal/service/store"
)

// Engine 指标计算引擎
type Engine struct {
	store *store.MemoryStore
}

// NewEngine 创建计算引擎
func NewEngine(store *store.MemoryStore) *Engine {
	return &Engine{store: store}
}

// Calculate 计算所有指标
func (e *Engine) Calculate() *model.Indicators {
	companies := e.store.GetAllCompanies()
	config := e.store.GetConfig()

	// 预聚合各分组数据
	sums := e.aggregateSums(companies)

	// 计算各指标
	indicators := model.NewIndicators()

	// 指标组一：限上社零额
	indicators.LimitAboveMonthValue = sums.AllRetailCurrent
	indicators.LimitAboveMonthRate = calcRate(sums.AllRetailCurrent, sums.AllRetailLastYear)
	indicators.LimitAboveCumulativeValue = sums.AllRetailCurrentCumulative
	indicators.LimitAboveCumulativeRate = calcRate(sums.AllRetailCurrentCumulative, sums.AllRetailLastYearCumulative)

	// 指标组二：专项增速
	indicators.EatWearUseMonthRate = calcRate(sums.EatWearUseRetailCurrent, sums.EatWearUseRetailLastYear)
	indicators.MicroSmallMonthRate = calcRate(sums.MicroSmallRetailCurrent, sums.MicroSmallRetailLastYear)

	// 指标组三：四大行业增速
	for industryType, industrySums := range sums.Industries {
		indicators.IndustryRates[industryType] = model.IndustryRate{
			MonthRate:      calcRate(industrySums.SalesCurrent, industrySums.SalesLastYear),
			CumulativeRate: calcRate(industrySums.SalesCurrentCumulative, industrySums.SalesLastYearCumulative),
		}
	}

	// 指标组四：社零总额
	// 估算本年累计限下社零额 = 上年累计限下社零额 × (1 + 小微企业增速)
	estimatedLimitBelow := config.LastYearLimitBelowCumulative * (1 + indicators.MicroSmallMonthRate)
	indicators.TotalSocialCumulativeValue = indicators.LimitAboveCumulativeValue + estimatedLimitBelow

	// 上年社零总额(累计) = 上年累计限上 + 上年累计限下
	lastYearTotal := sums.AllRetailLastYearCumulative + config.LastYearLimitBelowCumulative
	indicators.TotalSocialCumulativeRate = calcRate(indicators.TotalSocialCumulativeValue, lastYearTotal)

	return indicators
}

// aggregateSums 预聚合各分组数据
func (e *Engine) aggregateSums(companies []*model.Company) *model.IndicatorSums {
	sums := model.NewIndicatorSums()

	for _, c := range companies {
		// 全部企业汇总
		sums.AllRetailCurrent += c.RetailCurrentMonth
		sums.AllRetailLastYear += c.RetailLastYearMonth
		sums.AllRetailCurrentCumulative += c.RetailCurrentCumulative
		sums.AllRetailLastYearCumulative += c.RetailLastYearCumulative

		// 吃穿用企业
		if c.IsEatWearUse {
			sums.EatWearUseRetailCurrent += c.RetailCurrentMonth
			sums.EatWearUseRetailLastYear += c.RetailLastYearMonth
		}

		// 小微企业
		if c.IsMicroSmall() {
			sums.MicroSmallRetailCurrent += c.RetailCurrentMonth
			sums.MicroSmallRetailLastYear += c.RetailLastYearMonth
		}

		// 四大行业
		if industrySums, ok := sums.Industries[c.IndustryType]; ok {
			industrySums.SalesCurrent += c.SalesCurrentMonth
			industrySums.SalesLastYear += c.SalesLastYearMonth
			industrySums.SalesCurrentCumulative += c.SalesCurrentCumulative
			industrySums.SalesLastYearCumulative += c.SalesLastYearCumulative
		}
	}

	return sums
}

// calcRate 计算增速
func calcRate(current, lastYear float64) float64 {
	if lastYear == 0 {
		return 0
	}
	return (current - lastYear) / lastYear
}

// GetSums 获取汇总数据（用于测试）
func (e *Engine) GetSums() *model.IndicatorSums {
	companies := e.store.GetAllCompanies()
	return e.aggregateSums(companies)
}
