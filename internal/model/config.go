package model

// Config 系统配置
type Config struct {
	CurrentMonth                 int     `json:"currentMonth"`                 // 当前操作月份 (1-12)
	LastYearLimitBelowCumulative float64 `json:"lastYearLimitBelowCumulative"` // 上年累计限下社零额
}

// OptimizeConstraints 智能调整约束条件
type OptimizeConstraints struct {
	TargetGrowthRate   float64  `json:"targetGrowthRate"`   // 目标增速
	MaxIndividualRate  float64  `json:"maxIndividualRate"`  // 单个企业最大增速
	MinIndividualRate  float64  `json:"minIndividualRate"`  // 单个企业最小增速
	PriorityIndustries []string `json:"priorityIndustries"` // 优先调整的行业
}

// DefaultOptimizeConstraints 默认约束条件
func DefaultOptimizeConstraints() *OptimizeConstraints {
	return &OptimizeConstraints{
		MaxIndividualRate:  0.5,
		MinIndividualRate:  0,
		PriorityIndustries: []string{"catering", "retail"},
	}
}

// OptimizeResult 优化结果
type OptimizeResult struct {
	Success       bool                  `json:"success"`
	AchievedValue float64               `json:"achievedValue"`
	Adjustments   []CompanyAdjustment   `json:"adjustments"`
	Summary       OptimizeSummary       `json:"summary"`
	Indicators    *Indicators           `json:"indicators"`
}

// CompanyAdjustment 企业调整记录
type CompanyAdjustment struct {
	CompanyID     string  `json:"companyId"`
	CompanyName   string  `json:"companyName"`
	OriginalValue float64 `json:"originalValue"`
	AdjustedValue float64 `json:"adjustedValue"`
	ChangePercent float64 `json:"changePercent"`
}

// OptimizeSummary 优化汇总
type OptimizeSummary struct {
	AdjustedCount        int     `json:"adjustedCount"`
	TotalAdjustment      float64 `json:"totalAdjustment"`
	AverageChangePercent float64 `json:"averageChangePercent"`
}

// FieldMapping Excel字段映射
type FieldMapping struct {
	CompanyName              string `json:"companyName"`
	CreditCode               string `json:"creditCode"`
	IndustryCode             string `json:"industryCode"`
	CompanyScale             string `json:"companyScale"`
	RetailCurrentMonth       string `json:"retailCurrentMonth"`
	RetailLastYearMonth      string `json:"retailLastYearMonth"`
	RetailCurrentCumulative  string `json:"retailCurrentCumulative"`
	RetailLastYearCumulative string `json:"retailLastYearCumulative"`
	SalesCurrentMonth        string `json:"salesCurrentMonth"`
	SalesLastYearMonth       string `json:"salesLastYearMonth"`
	SalesCurrentCumulative   string `json:"salesCurrentCumulative"`
	SalesLastYearCumulative  string `json:"salesLastYearCumulative"`
}

// GenerationRule 历史数据生成规则
type GenerationRule struct {
	IndustryType    IndustryType `json:"industryType"`
	MinThreshold    float64      `json:"minThreshold"`
	MaxThreshold    float64      `json:"maxThreshold"`
	MonthlyVariance float64      `json:"monthlyVariance"`
}

// SheetInfo 工作表信息
type SheetInfo struct {
	Name     string `json:"name"`
	RowCount int    `json:"rowCount"`
}

// ImportResult 导入结果
type ImportResult struct {
	ImportedCount         int         `json:"importedCount"`
	GeneratedHistoryCount int         `json:"generatedHistoryCount"`
	Indicators            *Indicators `json:"indicators"`
}
