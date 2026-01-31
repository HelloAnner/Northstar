package parser

import "time"

// SheetType Sheet 类型
type SheetType string

const (
	SheetTypeWholesale      SheetType = "wholesale"
	SheetTypeRetail         SheetType = "retail"
	SheetTypeAccommodation  SheetType = "accommodation"
	SheetTypeCatering       SheetType = "catering"
	SheetTypeWRSnapshot     SheetType = "wr_snapshot"
	SheetTypeACSnapshot     SheetType = "ac_snapshot"
	SheetTypeSummary        SheetType = "summary"
	SheetTypeUnknown        SheetType = "unknown"
)

// FieldTimeType 字段时间类型
type FieldTimeType int

const (
	CurrentMonth       FieldTimeType = iota // 本月/当月
	PrevMonth                               // 上月
	LastYearMonth                           // 去年同期
	CurrentCumulative                       // 本年累计
	PrevCumulative                          // 本年累计到上月
	LastYearCumulative                      // 上年累计
	LastYearPrevCumulative                  // 上年累计到上月（用于批零主表）
)

// SheetRecognitionResult Sheet 识别结果
type SheetRecognitionResult struct {
	SheetName  string    `json:"sheetName"`
	SheetType  SheetType `json:"sheetType"`
	Confidence float64   `json:"confidence"` // 置信度 0-1
	DataYear   int       `json:"dataYear"`   // 识别出的数据年份
	DataMonth  int       `json:"dataMonth"`  // 识别出的数据月份
}

// FieldMapping 字段映射结果
type FieldMapping struct {
	ColumnIndex int           `json:"columnIndex"` // Excel 列索引
	ColumnName  string        `json:"columnName"`  // Excel 列名
	DBField     string        `json:"dbField"`     // 数据库字段名
	TimeType    FieldTimeType `json:"timeType"`    // 字段时间类型
}

// ParseResult 解析结果
type ParseResult struct {
	SheetName      string   `json:"sheetName"`
	SheetType      SheetType `json:"sheetType"`
	Status         string   `json:"status"`     // imported/skipped/error
	ImportedRows   int      `json:"importedRows"`
	ErrorRows      int      `json:"errorRows"`
	Errors         []string `json:"errors,omitempty"`
	Duration       time.Duration `json:"duration"`
}

// ImportReport 导入报告
type ImportReport struct {
	Filename       string        `json:"filename"`
	TotalSheets    int           `json:"totalSheets"`
	ImportedSheets int           `json:"importedSheets"`
	SkippedSheets  int           `json:"skippedSheets"`
	TotalRows      int           `json:"totalRows"`
	ImportedRows   int           `json:"importedRows"`
	ErrorRows      int           `json:"errorRows"`
	Duration       time.Duration `json:"duration"`
	Sheets         []ParseResult `json:"sheets"`
}
