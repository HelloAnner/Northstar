package model

// IndustryType 行业类型
type IndustryType string

const (
	IndustryWholesale     IndustryType = "wholesale"     // 批发
	IndustryRetail        IndustryType = "retail"        // 零售
	IndustryAccommodation IndustryType = "accommodation" // 住宿
	IndustryCatering      IndustryType = "catering"      // 餐饮
)

// Company 企业数据模型
type Company struct {
	ID           string       `json:"id"`
	RowNo        int          `json:"rowNo"`
	Name         string       `json:"name"`
	CreditCode   string       `json:"creditCode"`
	IndustryCode string       `json:"industryCode"`
	IndustryType IndustryType `json:"industryType"`
	CompanyScale int          `json:"companyScale"` // 1/2/3/4，3/4为小微
	IsEatWearUse bool         `json:"isEatWearUse"` // 是否属于吃穿用类

	// 零售额相关
	RetailLastYearMonth      float64 `json:"retailLastYearMonth"`      // 上年同期零售额
	RetailCurrentMonth       float64 `json:"retailCurrentMonth"`       // 本期零售额
	RetailLastYearCumulative float64 `json:"retailLastYearCumulative"` // 上年累计零售额
	RetailCurrentCumulative  float64 `json:"retailCurrentCumulative"`  // 本年累计零售额

	// 销售额/营业额相关
	SalesLastYearMonth      float64 `json:"salesLastYearMonth"`      // 上年同期销售额
	SalesCurrentMonth       float64 `json:"salesCurrentMonth"`       // 本期销售额
	SalesLastYearCumulative float64 `json:"salesLastYearCumulative"` // 上年累计销售额
	SalesCurrentCumulative  float64 `json:"salesCurrentCumulative"`  // 本年累计销售额

	// 住餐：明细拆分（不参与零售额汇总）
	RoomRevenueLastYearMonth      float64 `json:"roomRevenueLastYearMonth"`      // 上年同期客房收入
	RoomRevenueCurrentMonth       float64 `json:"roomRevenueCurrentMonth"`       // 本期客房收入
	RoomRevenueLastYearCumulative float64 `json:"roomRevenueLastYearCumulative"` // 上年累计客房收入
	RoomRevenueCurrentCumulative  float64 `json:"roomRevenueCurrentCumulative"`  // 本年累计客房收入

	FoodRevenueLastYearMonth      float64 `json:"foodRevenueLastYearMonth"`      // 上年同期餐费收入
	FoodRevenueCurrentMonth       float64 `json:"foodRevenueCurrentMonth"`       // 本期餐费收入
	FoodRevenueLastYearCumulative float64 `json:"foodRevenueLastYearCumulative"` // 上年累计餐费收入
	FoodRevenueCurrentCumulative  float64 `json:"foodRevenueCurrentCumulative"`  // 本年累计餐费收入

	GoodsSalesLastYearMonth      float64 `json:"goodsSalesLastYearMonth"`      // 上年同期商品销售额
	GoodsSalesCurrentMonth       float64 `json:"goodsSalesCurrentMonth"`       // 本期商品销售额
	GoodsSalesLastYearCumulative float64 `json:"goodsSalesLastYearCumulative"` // 上年累计商品销售额
	GoodsSalesCurrentCumulative  float64 `json:"goodsSalesCurrentCumulative"`  // 本年累计商品销售额

	// 原始值（用于重置）
	OriginalInitialized              bool    `json:"originalInitialized"`
	OriginalRowNo                    int     `json:"originalRowNo"`
	OriginalName                     string  `json:"originalName"`
	OriginalRetailLastYearMonth      float64 `json:"originalRetailLastYearMonth"`
	OriginalRetailCurrentMonth       float64 `json:"originalRetailCurrentMonth"`
	OriginalRetailLastYearCumulative float64 `json:"originalRetailLastYearCumulative"`
	OriginalRetailCurrentCumulative  float64 `json:"originalRetailCurrentCumulative"`
	OriginalSalesLastYearMonth       float64 `json:"originalSalesLastYearMonth"`
	OriginalSalesCurrentMonth        float64 `json:"originalSalesCurrentMonth"`
	OriginalSalesLastYearCumulative  float64 `json:"originalSalesLastYearCumulative"`
	OriginalSalesCurrentCumulative   float64 `json:"originalSalesCurrentCumulative"`

	OriginalRoomRevenueLastYearMonth      float64 `json:"originalRoomRevenueLastYearMonth"`
	OriginalRoomRevenueCurrentMonth       float64 `json:"originalRoomRevenueCurrentMonth"`
	OriginalRoomRevenueLastYearCumulative float64 `json:"originalRoomRevenueLastYearCumulative"`
	OriginalRoomRevenueCurrentCumulative  float64 `json:"originalRoomRevenueCurrentCumulative"`

	OriginalFoodRevenueLastYearMonth      float64 `json:"originalFoodRevenueLastYearMonth"`
	OriginalFoodRevenueCurrentMonth       float64 `json:"originalFoodRevenueCurrentMonth"`
	OriginalFoodRevenueLastYearCumulative float64 `json:"originalFoodRevenueLastYearCumulative"`
	OriginalFoodRevenueCurrentCumulative  float64 `json:"originalFoodRevenueCurrentCumulative"`

	OriginalGoodsSalesLastYearMonth      float64 `json:"originalGoodsSalesLastYearMonth"`
	OriginalGoodsSalesCurrentMonth       float64 `json:"originalGoodsSalesCurrentMonth"`
	OriginalGoodsSalesLastYearCumulative float64 `json:"originalGoodsSalesLastYearCumulative"`
	OriginalGoodsSalesCurrentCumulative  float64 `json:"originalGoodsSalesCurrentCumulative"`
}

// IsMicroSmall 判断是否为小微企业
func (c *Company) IsMicroSmall() bool {
	return c.CompanyScale == 3 || c.CompanyScale == 4
}

// MonthGrowthRate 计算当月增速
func (c *Company) MonthGrowthRate() float64 {
	if c.RetailLastYearMonth == 0 {
		return 0
	}
	return (c.RetailCurrentMonth - c.RetailLastYearMonth) / c.RetailLastYearMonth
}

// CumulativeGrowthRate 计算累计增速
func (c *Company) CumulativeGrowthRate() float64 {
	if c.RetailLastYearCumulative == 0 {
		return 0
	}
	return (c.RetailCurrentCumulative - c.RetailLastYearCumulative) / c.RetailLastYearCumulative
}

// ValidationError 校验错误
type ValidationError struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // error or warning
}

// Validate 校验企业数据
func (c *Company) Validate() []ValidationError {
	var errors []ValidationError

	// 零售额不能超过销售额
	if (c.IndustryType == IndustryWholesale || c.IndustryType == IndustryRetail) &&
		c.RetailCurrentMonth > c.SalesCurrentMonth && c.SalesCurrentMonth > 0 {
		errors = append(errors, ValidationError{
			Field:    "retailCurrentMonth",
			Message:  "零售额不能超过总销售额",
			Severity: "error",
		})
	}

	// 数值不能为负
	if c.RetailCurrentMonth < 0 {
		errors = append(errors, ValidationError{
			Field:    "retailCurrentMonth",
			Message:  "零售额不能为负数",
			Severity: "error",
		})
	}

	// 增速异常警告
	rate := c.MonthGrowthRate()
	if rate > 1.0 || rate < -0.5 {
		errors = append(errors, ValidationError{
			Field:    "retailCurrentMonth",
			Message:  "增速超出合理范围(-50% ~ 100%)",
			Severity: "warning",
		})
	}

	return errors
}
