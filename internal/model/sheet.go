package model

// SheetType 工作表类型（用于输入容错识别）
type SheetType string

const (
	SheetTypeUnknown SheetType = "unknown"

	SheetTypeWholesaleMain SheetType = "wholesale_main" // 批发主表
	SheetTypeRetailMain    SheetType = "retail_main"    // 零售主表

	SheetTypeAccommodationMain SheetType = "accommodation_main" // 住宿主表
	SheetTypeCateringMain      SheetType = "catering_main"      // 餐饮主表

	SheetTypeWholesaleRetailSnapshot       SheetType = "wholesale_retail_snapshot"       // 批零快照（月度）
	SheetTypeAccommodationCateringSnapshot SheetType = "accommodation_catering_snapshot" // 住餐快照（月度）

	SheetTypeEatWearUse         SheetType = "eat_wear_use"          // 吃穿用
	SheetTypeMicroSmall         SheetType = "micro_small"           // 小微
	SheetTypeEatWearUseExcluded SheetType = "eat_wear_use_excluded" // 吃穿用（剔除）

	SheetTypeFixedSocialRetail SheetType = "fixed_social_retail" // 社零额（定）
	SheetTypeFixedSummary      SheetType = "fixed_summary"       // 汇总表（定）
)

// SheetRecognition 单个 sheet 的识别结果
type SheetRecognition struct {
	SheetName     string    `json:"sheetName"`
	Type          SheetType `json:"type"`
	Score         float64   `json:"score"`
	MissingFields []string  `json:"missingFields"`
}
