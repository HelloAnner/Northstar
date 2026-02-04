package dagcalc

import "northstar/internal/store"

// Indicator 指标定义
type Indicator struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

// IndicatorGroup 指标分组
type IndicatorGroup struct {
	Name       string      `json:"name"`
	Indicators []Indicator `json:"indicators"`
}

// IndicatorCalculator 指标计算器
type IndicatorCalculator struct {
	store *store.Store
}

// NewIndicatorCalculator 创建指标计算器
func NewIndicatorCalculator(store *store.Store) *IndicatorCalculator {
	return &IndicatorCalculator{store: store}
}

// CalculateIndicators 计算全部指标
func CalculateIndicators(store *store.Store, year, month int) ([]IndicatorGroup, error) {
	return NewIndicatorCalculator(store).CalculateAll(year, month)
}

// CalculateAll 计算所有16个指标
func (c *IndicatorCalculator) CalculateAll(year, month int) ([]IndicatorGroup, error) {
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

	limitAboveIndicators, err := c.calculateLimitAbove(year, month)
	if err != nil {
		return nil, err
	}
	groups[0].Indicators = limitAboveIndicators

	specialIndicators, err := c.calculateSpecialRates(year, month)
	if err != nil {
		return nil, err
	}
	groups[1].Indicators = specialIndicators

	industryIndicators, err := c.calculateIndustryRates(year, month)
	if err != nil {
		return nil, err
	}
	groups[2].Indicators = industryIndicators

	totalIndicators, err := c.calculateTotalSocial(year, month, specialIndicators)
	if err != nil {
		return nil, err
	}
	groups[3].Indicators = totalIndicators

	return groups, nil
}

func (c *IndicatorCalculator) calculateLimitAbove(year, month int) ([]Indicator, error) {
	wrOpts := store.WRQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	}
	wrRecords, err := c.store.GetWRByYearMonth(wrOpts)
	if err != nil {
		return nil, err
	}

	acOpts := store.ACQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	}
	acRecords, err := c.store.GetACByYearMonth(acOpts)
	if err != nil {
		return nil, err
	}

	var retailCurrentMonthSum float64
	var retailLastYearMonthSum float64
	var retailCurrentCumulativeSum float64
	var retailLastYearCumulativeSum float64

	for _, record := range wrRecords {
		retailCurrentMonthSum += record.RetailCurrentMonth
		retailLastYearMonthSum += record.RetailLastYearMonth
		retailCurrentCumulativeSum += record.RetailCurrentCumulative
		retailLastYearCumulativeSum += record.RetailLastYearCumulative
	}

	for _, record := range acRecords {
		retailCurrentMonthSum += record.FoodCurrentMonth + record.GoodsCurrentMonth
		retailLastYearMonthSum += record.FoodLastYearMonth + record.GoodsLastYearMonth
		retailCurrentCumulativeSum += record.FoodCurrentCumulative + record.GoodsCurrentCumulative
		retailLastYearCumulativeSum += record.FoodLastYearCumulative + record.GoodsLastYearCumulative
	}

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

func (c *IndicatorCalculator) calculateSpecialRates(year, month int) ([]Indicator, error) {
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

func (c *IndicatorCalculator) calculateIndustryRates(year, month int) ([]Indicator, error) {
	indicators := []Indicator{}

	wholesaleIndicators, err := c.calculateWRIndustryRate(year, month, "wholesale", "批发业销售额")
	if err != nil {
		return nil, err
	}
	indicators = append(indicators, wholesaleIndicators...)

	retailIndicators, err := c.calculateWRIndustryRate(year, month, "retail", "零售业销售额")
	if err != nil {
		return nil, err
	}
	indicators = append(indicators, retailIndicators...)

	accommodationIndicators, err := c.calculateACIndustryRate(year, month, "accommodation", "住宿业营业额")
	if err != nil {
		return nil, err
	}
	indicators = append(indicators, accommodationIndicators...)

	cateringIndicators, err := c.calculateACIndustryRate(year, month, "catering", "餐饮业营业额")
	if err != nil {
		return nil, err
	}
	indicators = append(indicators, cateringIndicators...)

	return indicators, nil
}

func (c *IndicatorCalculator) calculateWRIndustryRate(year, month int, industryType, name string) ([]Indicator, error) {
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

func (c *IndicatorCalculator) calculateACIndustryRate(year, month int, industryType, name string) ([]Indicator, error) {
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

func (c *IndicatorCalculator) calculateTotalSocial(year, month int, specialIndicators []Indicator) ([]Indicator, error) {
	limitAboveIndicators, err := c.calculateLimitAbove(year, month)
	if err != nil {
		return nil, err
	}

	limitAboveCumulativeValue := limitAboveIndicators[2].Value

	lastYearLimitBelowCumulative, err := c.store.GetConfigFloat("last_year_limit_below_cumulative")
	if err != nil {
		lastYearLimitBelowCumulative = 0
	}

	microSmallRate := specialIndicators[1].Value

	estimatedLimitBelowCumulative := lastYearLimitBelowCumulative * (1 + microSmallRate/100)

	totalSocialCumulative := limitAboveCumulativeValue + estimatedLimitBelowCumulative

	wrOpts := store.WRQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	}
	wrRecords, err := c.store.GetWRByYearMonth(wrOpts)
	if err != nil {
		return nil, err
	}
	acOpts := store.ACQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	}
	acRecords, err := c.store.GetACByYearMonth(acOpts)
	if err != nil {
		return nil, err
	}

	var retailLastYearCumulativeSum float64
	for _, record := range wrRecords {
		retailLastYearCumulativeSum += record.RetailLastYearCumulative
	}
	for _, record := range acRecords {
		retailLastYearCumulativeSum += record.FoodLastYearCumulative + record.GoodsLastYearCumulative
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

func intPtr(i int) *int {
	return &i
}
