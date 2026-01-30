package model

import "time"

// WholesaleRetail 批发零售企业（V3 数据库模型）
type WholesaleRetail struct {
	ID           int64  `json:"id"`
	CreditCode   string `json:"creditCode"`
	Name         string `json:"name"`
	IndustryCode string `json:"industryCode"`
	IndustryType string `json:"industryType"` // wholesale/retail
	CompanyScale int    `json:"companyScale"`
	RowNo        int    `json:"rowNo"`

	// 数据月份标识
	DataYear  int `json:"dataYear"`
	DataMonth int `json:"dataMonth"`

	// 销售额
	SalesPrevMonth              float64  `json:"salesPrevMonth"`
	SalesCurrentMonth           float64  `json:"salesCurrentMonth"`
	SalesLastYearMonth          float64  `json:"salesLastYearMonth"`
	SalesMonthRate              *float64 `json:"salesMonthRate"`              // 计算字段，可为 NULL
	SalesPrevCumulative         float64  `json:"salesPrevCumulative"`
	SalesLastYearPrevCumulative float64  `json:"salesLastYearPrevCumulative"`
	SalesCurrentCumulative      float64  `json:"salesCurrentCumulative"`
	SalesLastYearCumulative     float64  `json:"salesLastYearCumulative"`
	SalesCumulativeRate         *float64 `json:"salesCumulativeRate"` // 计算字段

	// 零售额
	RetailPrevMonth              float64  `json:"retailPrevMonth"`
	RetailCurrentMonth           float64  `json:"retailCurrentMonth"`
	RetailLastYearMonth          float64  `json:"retailLastYearMonth"`
	RetailMonthRate              *float64 `json:"retailMonthRate"`
	RetailPrevCumulative         float64  `json:"retailPrevCumulative"`
	RetailLastYearPrevCumulative float64  `json:"retailLastYearPrevCumulative"`
	RetailCurrentCumulative      float64  `json:"retailCurrentCumulative"`
	RetailLastYearCumulative     float64  `json:"retailLastYearCumulative"`
	RetailCumulativeRate         *float64 `json:"retailCumulativeRate"`
	RetailRatio                  *float64 `json:"retailRatio"` // 零销比

	// 商品分类
	CatGrainOilFood  float64 `json:"catGrainOilFood"`
	CatBeverage      float64 `json:"catBeverage"`
	CatTobaccoLiquor float64 `json:"catTobaccoLiquor"`
	CatClothing      float64 `json:"catClothing"`
	CatDailyUse      float64 `json:"catDailyUse"`
	CatAutomobile    float64 `json:"catAutomobile"`

	// 分类标记
	IsSmallMicro  int `json:"isSmallMicro"`
	IsEatWearUse  int `json:"isEatWearUse"`

	// 补充字段
	FirstReportIP string  `json:"firstReportIp"`
	FillIP        string  `json:"fillIp"`
	NetworkSales  float64 `json:"networkSales"`
	OpeningYear   *int    `json:"openingYear"`
	OpeningMonth  *int    `json:"openingMonth"`

	// 原始值备份
	OriginalSalesCurrentMonth  *float64 `json:"originalSalesCurrentMonth"`
	OriginalRetailCurrentMonth *float64 `json:"originalRetailCurrentMonth"`

	// 元数据
	SourceSheet string    `json:"sourceSheet"`
	SourceFile  string    `json:"sourceFile"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// AccommodationCatering 住宿餐饮企业（V3 数据库模型）
type AccommodationCatering struct {
	ID           int64  `json:"id"`
	CreditCode   string `json:"creditCode"`
	Name         string `json:"name"`
	IndustryCode string `json:"industryCode"`
	IndustryType string `json:"industryType"` // accommodation/catering
	CompanyScale int    `json:"companyScale"`
	RowNo        int    `json:"rowNo"`

	// 数据月份标识
	DataYear  int `json:"dataYear"`
	DataMonth int `json:"dataMonth"`

	// 营业额
	RevenuePrevMonth         float64  `json:"revenuePrevMonth"`
	RevenueCurrentMonth      float64  `json:"revenueCurrentMonth"`
	RevenueLastYearMonth     float64  `json:"revenueLastYearMonth"`
	RevenueMonthRate         *float64 `json:"revenueMonthRate"`
	RevenuePrevCumulative    float64  `json:"revenuePrevCumulative"`
	RevenueCurrentCumulative float64  `json:"revenueCurrentCumulative"`
	RevenueLastYearCumulative float64 `json:"revenueLastYearCumulative"`
	RevenueCumulativeRate    *float64 `json:"revenueCumulativeRate"`

	// 客房收入
	RoomPrevMonth         float64 `json:"roomPrevMonth"`
	RoomCurrentMonth      float64 `json:"roomCurrentMonth"`
	RoomLastYearMonth     float64 `json:"roomLastYearMonth"`
	RoomPrevCumulative    float64 `json:"roomPrevCumulative"`
	RoomCurrentCumulative float64 `json:"roomCurrentCumulative"`
	RoomLastYearCumulative float64 `json:"roomLastYearCumulative"`

	// 餐费收入
	FoodPrevMonth         float64 `json:"foodPrevMonth"`
	FoodCurrentMonth      float64 `json:"foodCurrentMonth"`
	FoodLastYearMonth     float64 `json:"foodLastYearMonth"`
	FoodPrevCumulative    float64 `json:"foodPrevCumulative"`
	FoodCurrentCumulative float64 `json:"foodCurrentCumulative"`
	FoodLastYearCumulative float64 `json:"foodLastYearCumulative"`

	// 商品销售额
	GoodsPrevMonth         float64 `json:"goodsPrevMonth"`
	GoodsCurrentMonth      float64 `json:"goodsCurrentMonth"`
	GoodsLastYearMonth     float64 `json:"goodsLastYearMonth"`
	GoodsPrevCumulative    float64 `json:"goodsPrevCumulative"`
	GoodsCurrentCumulative float64 `json:"goodsCurrentCumulative"`
	GoodsLastYearCumulative float64 `json:"goodsLastYearCumulative"`

	// 零售额
	RetailCurrentMonth  float64 `json:"retailCurrentMonth"`
	RetailLastYearMonth float64 `json:"retailLastYearMonth"`

	// 分类标记
	IsSmallMicro int `json:"isSmallMicro"`
	IsEatWearUse int `json:"isEatWearUse"`

	// 补充字段
	FirstReportIP string  `json:"firstReportIp"`
	FillIP        string  `json:"fillIp"`
	NetworkSales  float64 `json:"networkSales"`
	OpeningYear   *int    `json:"openingYear"`
	OpeningMonth  *int    `json:"openingMonth"`

	// 原始值备份
	OriginalRevenueCurrentMonth *float64 `json:"originalRevenueCurrentMonth"`
	OriginalRoomCurrentMonth    *float64 `json:"originalRoomCurrentMonth"`
	OriginalFoodCurrentMonth    *float64 `json:"originalFoodCurrentMonth"`
	OriginalGoodsCurrentMonth   *float64 `json:"originalGoodsCurrentMonth"`

	// 元数据
	SourceSheet string    `json:"sourceSheet"`
	SourceFile  string    `json:"sourceFile"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// WRSnapshot 批零历史快照
type WRSnapshot struct {
	ID            int64   `json:"id"`
	SnapshotYear  int     `json:"snapshotYear"`
	SnapshotMonth int     `json:"snapshotMonth"`
	SnapshotName  string  `json:"snapshotName"`
	CreditCode    string  `json:"creditCode"`
	Name          string  `json:"name"`
	IndustryCode  string  `json:"industryCode"`
	CompanyScale  int     `json:"companyScale"`

	SalesCurrentMonth       float64  `json:"salesCurrentMonth"`
	SalesCurrentCumulative  float64  `json:"salesCurrentCumulative"`
	SalesLastYearMonth      *float64 `json:"salesLastYearMonth"`
	SalesLastYearCumulative *float64 `json:"salesLastYearCumulative"`

	RetailCurrentMonth       float64  `json:"retailCurrentMonth"`
	RetailCurrentCumulative  float64  `json:"retailCurrentCumulative"`
	RetailLastYearMonth      *float64 `json:"retailLastYearMonth"`
	RetailLastYearCumulative *float64 `json:"retailLastYearCumulative"`

	CatGrainOilFood  *float64 `json:"catGrainOilFood"`
	CatBeverage      *float64 `json:"catBeverage"`
	CatTobaccoLiquor *float64 `json:"catTobaccoLiquor"`
	CatClothing      *float64 `json:"catClothing"`
	CatDailyUse      *float64 `json:"catDailyUse"`
	CatAutomobile    *float64 `json:"catAutomobile"`

	SourceSheet string    `json:"sourceSheet"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ACSnapshot 住餐历史快照
type ACSnapshot struct {
	ID            int64   `json:"id"`
	SnapshotYear  int     `json:"snapshotYear"`
	SnapshotMonth int     `json:"snapshotMonth"`
	SnapshotName  string  `json:"snapshotName"`
	CreditCode    string  `json:"creditCode"`
	Name          string  `json:"name"`
	IndustryCode  string  `json:"industryCode"`
	CompanyScale  int     `json:"companyScale"`

	RevenueCurrentMonth      float64  `json:"revenueCurrentMonth"`
	RevenueCurrentCumulative float64  `json:"revenueCurrentCumulative"`
	RoomCurrentMonth         float64  `json:"roomCurrentMonth"`
	RoomCurrentCumulative    *float64 `json:"roomCurrentCumulative"`
	FoodCurrentMonth         float64  `json:"foodCurrentMonth"`
	FoodCurrentCumulative    *float64 `json:"foodCurrentCumulative"`
	GoodsCurrentMonth        float64  `json:"goodsCurrentMonth"`
	GoodsCurrentCumulative   *float64 `json:"goodsCurrentCumulative"`

	SourceSheet string    `json:"sourceSheet"`
	CreatedAt   time.Time `json:"createdAt"`
}

// SheetMeta Sheet 元信息
type SheetMeta struct {
	ID               int64     `json:"id"`
	SheetName        string    `json:"sheetName"`
	SheetType        string    `json:"sheetType"`
	Confidence       float64   `json:"confidence"`
	TotalRows        int       `json:"totalRows"`
	TotalColumns     int       `json:"totalColumns"`
	ImportedRows     int       `json:"importedRows"`
	ColumnsJSON      string    `json:"columnsJson"`
	ColumnMappingJSON string   `json:"columnMappingJson"`
	Status           string    `json:"status"`
	ErrorMessage     string    `json:"errorMessage"`
	ImportLogID      *int64    `json:"importLogId"`
	SourceFile       string    `json:"sourceFile"`
	CreatedAt        time.Time `json:"createdAt"`
}

// ImportLog 导入日志
type ImportLog struct {
	ID             int64      `json:"id"`
	Filename       string     `json:"filename"`
	FilePath       string     `json:"filePath"`
	FileSize       int64      `json:"fileSize"`
	FileHash       string     `json:"fileHash"`
	TotalSheets    int        `json:"totalSheets"`
	ImportedSheets int        `json:"importedSheets"`
	SkippedSheets  int        `json:"skippedSheets"`
	TotalRows      int        `json:"totalRows"`
	ImportedRows   int        `json:"importedRows"`
	ErrorRows      int        `json:"errorRows"`
	Status         string     `json:"status"`
	ErrorMessage   string     `json:"errorMessage"`
	StartedAt      time.Time  `json:"startedAt"`
	CompletedAt    *time.Time `json:"completedAt"`
}
