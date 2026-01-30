package calculator

import (
	"northstar/internal/store"
)

// Indicator 指标定义
type Indicator struct {
	ID    string  `json:"id"`    // 指标ID
	Name  string  `json:"name"`  // 指标名称
	Value float64 `json:"value"` // 指标值
	Unit  string  `json:"unit"`  // 单位 (如 万元、%)
}

// IndicatorGroup 指标分组
type IndicatorGroup struct {
	Name       string      `json:"name"`       // 分组名称
	Indicators []Indicator `json:"indicators"` // 指标列表
}

// Calculator 指标计算器
type Calculator struct {
	store *store.Store
}

// NewCalculator 创建计算器
func NewCalculator(store *store.Store) *Calculator {
	return &Calculator{
		store: store,
	}
}

// CalculateAll 计算所有16个指标
func (c *Calculator) CalculateAll(year, month int) ([]IndicatorGroup, error) {
	groups := []IndicatorGroup{
		{
			Name:       "限上社零额",
			Indicators: []Indicator{},
		},
		{
			Name:       "专项增速",
			Indicators: []Indicator{},
		},
		{
			Name:       "四大行业增速",
			Indicators: []Indicator{},
		},
		{
			Name:       "社零总额",
			Indicators: []Indicator{},
		},
	}

	// 计算限上社零额（4个指标）
	limitAboveIndicators, err := c.calculateLimitAbove(year, month)
	if err != nil {
		return nil, err
	}
	groups[0].Indicators = limitAboveIndicators

	// 计算专项增速（2个指标）
	specialIndicators, err := c.calculateSpecialRates(year, month)
	if err != nil {
		return nil, err
	}
	groups[1].Indicators = specialIndicators

	// 计算四大行业增速（8个指标）
	industryIndicators, err := c.calculateIndustryRates(year, month)
	if err != nil {
		return nil, err
	}
	groups[2].Indicators = industryIndicators

	// 计算社零总额（2个指标）
	totalIndicators, err := c.calculateTotalSocial(year, month, specialIndicators)
	if err != nil {
		return nil, err
	}
	groups[3].Indicators = totalIndicators

	return groups, nil
}

// calculateLimitAbove 计算限上社零额（4个指标）
func (c *Calculator) calculateLimitAbove(year, month int) ([]Indicator, error) {
	// 查询所有批零企业
	wrOpts := store.WRQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	}
	wrRecords, err := c.store.GetWRByYearMonth(wrOpts)
	if err != nil {
		return nil, err
	}

	// 查询所有住餐企业
	acOpts := store.ACQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	}
	acRecords, err := c.store.GetACByYearMonth(acOpts)
	if err != nil {
		return nil, err
	}

	// 汇总零售额
	var retailCurrentMonthSum float64
	var retailLastYearMonthSum float64
	var retailCurrentCumulativeSum float64
	var retailLastYearCumulativeSum float64

	// 批零企业零售额
	for _, record := range wrRecords {
		retailCurrentMonthSum += record.RetailCurrentMonth
		retailLastYearMonthSum += record.RetailLastYearMonth
		retailCurrentCumulativeSum += record.RetailCurrentCumulative
		retailLastYearCumulativeSum += record.RetailLastYearCumulative
	}

	// 住餐企业零售额
	for _, record := range acRecords {
		retailCurrentMonthSum += record.RetailCurrentMonth
		retailLastYearMonthSum += record.RetailLastYearMonth
		// 住餐没有累计零售额，不累加
	}

	// 计算增速
	monthRate := 0.0
	if retailLastYearMonthSum != 0 {
		monthRate = (retailCurrentMonthSum - retailLastYearMonthSum) / retailLastYearMonthSum * 100
	}

	cumulativeRate := 0.0
	if retailLastYearCumulativeSum != 0 {
		cumulativeRate = (retailCurrentCumulativeSum - retailLastYearCumulativeSum) / retailLastYearCumulativeSum * 100
	}

	return []Indicator{
		{
			ID:    "limitAbove_month_value",
			Name:  "限上社零额（当月值）",
			Value: retailCurrentMonthSum,
			Unit:  "万元",
		},
		{
			ID:    "limitAbove_month_rate",
			Name:  "限上社零额增速（当月）",
			Value: monthRate,
			Unit:  "%",
		},
		{
			ID:    "limitAbove_cumulative_value",
			Name:  "限上社零额（累计值）",
			Value: retailCurrentCumulativeSum,
			Unit:  "万元",
		},
		{
			ID:    "limitAbove_cumulative_rate",
			Name:  "限上社零额增速（累计）",
			Value: cumulativeRate,
			Unit:  "%",
		},
	}, nil
}

// calculateSpecialRates 计算专项增速（2个指标）
func (c *Calculator) calculateSpecialRates(year, month int) ([]Indicator, error) {
	// 吃穿用增速
	eatWearUseOpts := store.WRQueryOptions{
		DataYear:     &year,
		DataMonth:    &month,
		IsEatWearUse: intPtr(1),
	}
	eatWearUseRecords, err := c.store.GetWRByYearMonth(eatWearUseOpts)
	if err != nil {
		return nil, err
	}

	var eatWearUseCurrentSum float64
	var eatWearUseLastYearSum float64
	for _, record := range eatWearUseRecords {
		eatWearUseCurrentSum += record.RetailCurrentMonth
		eatWearUseLastYearSum += record.RetailLastYearMonth
	}

	eatWearUseRate := 0.0
	if eatWearUseLastYearSum != 0 {
		eatWearUseRate = (eatWearUseCurrentSum - eatWearUseLastYearSum) / eatWearUseLastYearSum * 100
	}

	// 小微企业增速
	smallMicroOpts := store.WRQueryOptions{
		DataYear:     &year,
		DataMonth:    &month,
		IsSmallMicro: intPtr(1),
	}
	smallMicroRecords, err := c.store.GetWRByYearMonth(smallMicroOpts)
	if err != nil {
		return nil, err
	}

	var smallMicroCurrentSum float64
	var smallMicroLastYearSum float64
	for _, record := range smallMicroRecords {
		smallMicroCurrentSum += record.RetailCurrentMonth
		smallMicroLastYearSum += record.RetailLastYearMonth
	}

	smallMicroRate := 0.0
	if smallMicroLastYearSum != 0 {
		smallMicroRate = (smallMicroCurrentSum - smallMicroLastYearSum) / smallMicroLastYearSum * 100
	}

	return []Indicator{
		{
			ID:    "eatWearUse_month_rate",
			Name:  "吃穿用增速（当月）",
			Value: eatWearUseRate,
			Unit:  "%",
		},
		{
			ID:    "microSmall_month_rate",
			Name:  "小微企业增速（当月）",
			Value: smallMicroRate,
			Unit:  "%",
		},
	}, nil
}

// calculateIndustryRates 计算四大行业增速（8个指标）
func (c *Calculator) calculateIndustryRates(year, month int) ([]Indicator, error) {
	indicators := []Indicator{}

	// 批发业
	wholesaleIndicators, err := c.calculateWRIndustryRate(year, month, "wholesale", "批发业销售额")
	if err != nil {
		return nil, err
	}
	indicators = append(indicators, wholesaleIndicators...)

	// 零售业
	retailIndicators, err := c.calculateWRIndustryRate(year, month, "retail", "零售业销售额")
	if err != nil {
		return nil, err
	}
	indicators = append(indicators, retailIndicators...)

	// 住宿业
	accommodationIndicators, err := c.calculateACIndustryRate(year, month, "accommodation", "住宿业营业额")
	if err != nil {
		return nil, err
	}
	indicators = append(indicators, accommodationIndicators...)

	// 餐饮业
	cateringIndicators, err := c.calculateACIndustryRate(year, month, "catering", "餐饮业营业额")
	if err != nil {
		return nil, err
	}
	indicators = append(indicators, cateringIndicators...)

	return indicators, nil
}

// calculateWRIndustryRate 计算批零行业增速
func (c *Calculator) calculateWRIndustryRate(year, month int, industryType, name string) ([]Indicator, error) {
	opts := store.WRQueryOptions{
		DataYear:     &year,
		DataMonth:    &month,
		IndustryType: &industryType,
	}
	records, err := c.store.GetWRByYearMonth(opts)
	if err != nil {
		return nil, err
	}

	var currentMonthSum float64
	var lastYearMonthSum float64
	var currentCumulativeSum float64
	var lastYearCumulativeSum float64

	for _, record := range records {
		currentMonthSum += record.SalesCurrentMonth
		lastYearMonthSum += record.SalesLastYearMonth
		currentCumulativeSum += record.SalesCurrentCumulative
		lastYearCumulativeSum += record.SalesLastYearCumulative
	}

	monthRate := 0.0
	if lastYearMonthSum != 0 {
		monthRate = (currentMonthSum - lastYearMonthSum) / lastYearMonthSum * 100
	}

	cumulativeRate := 0.0
	if lastYearCumulativeSum != 0 {
		cumulativeRate = (currentCumulativeSum - lastYearCumulativeSum) / lastYearCumulativeSum * 100
	}

	idPrefix := industryType
	return []Indicator{
		{
			ID:    idPrefix + "_month_rate",
			Name:  name + "增速（当月）",
			Value: monthRate,
			Unit:  "%",
		},
		{
			ID:    idPrefix + "_cumulative_rate",
			Name:  name + "增速（累计）",
			Value: cumulativeRate,
			Unit:  "%",
		},
	}, nil
}

// calculateACIndustryRate 计算住餐行业增速
func (c *Calculator) calculateACIndustryRate(year, month int, industryType, name string) ([]Indicator, error) {
	opts := store.ACQueryOptions{
		DataYear:     &year,
		DataMonth:    &month,
		IndustryType: &industryType,
	}
	records, err := c.store.GetACByYearMonth(opts)
	if err != nil {
		return nil, err
	}

	var currentMonthSum float64
	var lastYearMonthSum float64
	var currentCumulativeSum float64
	var lastYearCumulativeSum float64

	for _, record := range records {
		currentMonthSum += record.RevenueCurrentMonth
		lastYearMonthSum += record.RevenueLastYearMonth
		currentCumulativeSum += record.RevenueCurrentCumulative
		lastYearCumulativeSum += record.RevenueLastYearCumulative
	}

	monthRate := 0.0
	if lastYearMonthSum != 0 {
		monthRate = (currentMonthSum - lastYearMonthSum) / lastYearMonthSum * 100
	}

	cumulativeRate := 0.0
	if lastYearCumulativeSum != 0 {
		cumulativeRate = (currentCumulativeSum - lastYearCumulativeSum) / lastYearCumulativeSum * 100
	}

	idPrefix := industryType
	return []Indicator{
		{
			ID:    idPrefix + "_month_rate",
			Name:  name + "增速（当月）",
			Value: monthRate,
			Unit:  "%",
		},
		{
			ID:    idPrefix + "_cumulative_rate",
			Name:  name + "增速（累计）",
			Value: cumulativeRate,
			Unit:  "%",
		},
	}, nil
}

// calculateTotalSocial 计算社零总额（2个指标）
func (c *Calculator) calculateTotalSocial(year, month int, specialIndicators []Indicator) ([]Indicator, error) {
	// 获取限上累计社零额
	limitAboveIndicators, err := c.calculateLimitAbove(year, month)
	if err != nil {
		return nil, err
	}

	limitAboveCumulativeValue := limitAboveIndicators[2].Value // 第3个指标是累计值

	// 获取配置：上年累计限下社零额
	lastYearLimitBelowCumulative, err := c.store.GetConfigFloat("last_year_limit_below_cumulative")
	if err != nil {
		lastYearLimitBelowCumulative = 0
	}

	// 获取小微企业增速
	microSmallRate := specialIndicators[1].Value // 第2个指标是小微增速

	// 估算本年累计限下社零额
	estimatedLimitBelowCumulative := lastYearLimitBelowCumulative * (1 + microSmallRate/100)

	// 社零总额（累计值）
	totalSocialCumulative := limitAboveCumulativeValue + estimatedLimitBelowCumulative

	// 社零总额增速（累计）
	// 上年社零总额（累计）
	wrOpts := store.WRQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	}
	wrRecords, err := c.store.GetWRByYearMonth(wrOpts)
	if err != nil {
		return nil, err
	}

	var retailLastYearCumulativeSum float64
	for _, record := range wrRecords {
		retailLastYearCumulativeSum += record.RetailLastYearCumulative
	}

	lastYearTotalCumulative := retailLastYearCumulativeSum + lastYearLimitBelowCumulative

	totalSocialRate := 0.0
	if lastYearTotalCumulative != 0 {
		totalSocialRate = (totalSocialCumulative - lastYearTotalCumulative) / lastYearTotalCumulative * 100
	}

	return []Indicator{
		{
			ID:    "totalSocial_cumulative_value",
			Name:  "社零总额（累计值）",
			Value: totalSocialCumulative,
			Unit:  "万元",
		},
		{
			ID:    "totalSocial_cumulative_rate",
			Name:  "社零总额增速（累计）",
			Value: totalSocialRate,
			Unit:  "%",
		},
	}, nil
}

// intPtr 返回整数指针
func intPtr(i int) *int {
	return &i
}
