package model

// CanonicalAmount 统一口径的数值集合（当月/累计、同比基数）
type CanonicalAmount struct {
	CurrentMonth       float64 `json:"currentMonth"`
	LastYearMonth      float64 `json:"lastYearMonth"`
	CurrentCumulative  float64 `json:"currentCumulative"`
	LastYearCumulative float64 `json:"lastYearCumulative"`
}

// CanonicalCompany 统一口径企业数据（供解析、联动与模板输出）
type CanonicalCompany struct {
	RowNo int `json:"rowNo"`

	CreditCode   string       `json:"creditCode"`
	Name         string       `json:"name"`
	IndustryCode string       `json:"industryCode"`
	IndustryType IndustryType `json:"industryType"`
	CompanyScale int          `json:"companyScale"`
	IsEatWearUse bool         `json:"isEatWearUse"`

	Retail CanonicalAmount `json:"retail"`
	Sales  CanonicalAmount `json:"sales"`

	// 住餐口径
	Revenue     CanonicalAmount `json:"revenue"`
	RoomRevenue CanonicalAmount `json:"roomRevenue"`
	FoodRevenue CanonicalAmount `json:"foodRevenue"`
	GoodsSales  CanonicalAmount `json:"goodsSales"`
}

// CanonicalWorkbookData 解析后的统一口径数据
type CanonicalWorkbookData struct {
	Month     int                 `json:"month"`
	Companies []*CanonicalCompany `json:"companies"`
}
