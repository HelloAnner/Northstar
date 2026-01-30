package parser

import (
	"strings"
)

// SheetRecognizer Sheet 类型识别器
type SheetRecognizer struct{}

// NewSheetRecognizer 创建识别器
func NewSheetRecognizer() *SheetRecognizer {
	return &SheetRecognizer{}
}

// Recognize 识别 Sheet 类型
func (r *SheetRecognizer) Recognize(sheetName string, columnNames []string) SheetRecognitionResult {
	// 规范化列名
	normalized := make([]string, len(columnNames))
	for i, col := range columnNames {
		normalized[i] = NormalizeColumnName(col)
	}

	// 尝试从 Sheet 名提取年月
	year, month, _ := ExtractYearMonth(sheetName)

	// 依次尝试各种类型
	if result := r.recognizeWholesaleRetail(sheetName, normalized); result.Confidence >= 0.5 {
		result.DataYear = year
		result.DataMonth = month
		return result
	}

	if result := r.recognizeAccommodationCatering(sheetName, normalized); result.Confidence >= 0.5 {
		result.DataYear = year
		result.DataMonth = month
		return result
	}

	if result := r.recognizeSnapshot(sheetName, normalized); result.Confidence >= 0.5 {
		result.DataYear = year
		result.DataMonth = month
		return result
	}

	if result := r.recognizeSummary(sheetName, normalized); result.Confidence >= 0.3 {
		return result
	}

	// 无法识别
	return SheetRecognitionResult{
		SheetName:  sheetName,
		SheetType:  SheetTypeUnknown,
		Confidence: 0,
	}
}

// recognizeWholesaleRetail 识别批发零售主表
func (r *SheetRecognizer) recognizeWholesaleRetail(sheetName string, columns []string) SheetRecognitionResult {
	// 关键字段列表
	keyFields := []string{
		"统一社会信用代码",
		"单位详细名称|单位名称|企业名称",
		"行业代码",
		"销售额",
		"零售额",
		"单位规模",
		"粮油食品类",
	}

	matchCount := 0
	for _, field := range keyFields {
		for _, col := range columns {
			if MatchPattern(col, field) {
				matchCount++
				break
			}
		}
	}

	confidence := float64(matchCount) / float64(len(keyFields))

	// Sheet 名称辅助判定
	nameBoost := 0.0
	if strings.Contains(sheetName, "批发") {
		nameBoost = 0.2
		if confidence >= 0.5 {
			return SheetRecognitionResult{
				SheetName:  sheetName,
				SheetType:  SheetTypeWholesale,
				Confidence: confidence + nameBoost,
			}
		}
	}
	if strings.Contains(sheetName, "零售") {
		nameBoost = 0.2
		if confidence >= 0.5 {
			return SheetRecognitionResult{
				SheetName:  sheetName,
				SheetType:  SheetTypeRetail,
				Confidence: confidence + nameBoost,
			}
		}
	}
	if strings.Contains(sheetName, "批零") {
		nameBoost = 0.15
	}

	// 通过行业代码列内容判断
	// 这里先返回通用的批零类型，具体类型在解析时根据行业代码区分
	if confidence >= 0.5 {
		sheetType := SheetTypeWholesale // 默认批发
		if strings.Contains(sheetName, "零售") {
			sheetType = SheetTypeRetail
		}
		return SheetRecognitionResult{
			SheetName:  sheetName,
			SheetType:  sheetType,
			Confidence: confidence + nameBoost,
		}
	}

	return SheetRecognitionResult{
		SheetName:  sheetName,
		SheetType:  SheetTypeUnknown,
		Confidence: confidence,
	}
}

// recognizeAccommodationCatering 识别住宿餐饮主表
func (r *SheetRecognizer) recognizeAccommodationCatering(sheetName string, columns []string) SheetRecognitionResult {
	keyFields := []string{
		"统一社会信用代码",
		"单位详细名称|单位名称|企业名称",
		"行业代码",
		"营业额",
		"客房收入",
		"餐费收入",
	}

	matchCount := 0
	for _, field := range keyFields {
		for _, col := range columns {
			if MatchPattern(col, field) {
				matchCount++
				break
			}
		}
	}

	confidence := float64(matchCount) / float64(len(keyFields))

	// Sheet 名称辅助判定
	nameBoost := 0.0
	if strings.Contains(sheetName, "住宿") {
		nameBoost = 0.2
		if confidence >= 0.5 {
			return SheetRecognitionResult{
				SheetName:  sheetName,
				SheetType:  SheetTypeAccommodation,
				Confidence: confidence + nameBoost,
			}
		}
	}
	if strings.Contains(sheetName, "餐饮") {
		nameBoost = 0.2
		if confidence >= 0.5 {
			return SheetRecognitionResult{
				SheetName:  sheetName,
				SheetType:  SheetTypeCatering,
				Confidence: confidence + nameBoost,
			}
		}
	}
	if strings.Contains(sheetName, "住餐") {
		nameBoost = 0.15
	}

	if confidence >= 0.5 {
		sheetType := SheetTypeAccommodation // 默认住宿
		if strings.Contains(sheetName, "餐饮") {
			sheetType = SheetTypeCatering
		}
		return SheetRecognitionResult{
			SheetName:  sheetName,
			SheetType:  sheetType,
			Confidence: confidence + nameBoost,
		}
	}

	return SheetRecognitionResult{
		SheetName:  sheetName,
		SheetType:  SheetTypeUnknown,
		Confidence: confidence,
	}
}

// recognizeSnapshot 识别快照表
func (r *SheetRecognizer) recognizeSnapshot(sheetName string, columns []string) SheetRecognitionResult {
	// 快照表特征：列名包含 "本年-本月" / "上年-本月" 等格式
	snapshotKeywords := []string{
		"本年-本月",
		"本年-1—本月",
		"上年-本月",
		"上年-1—本月",
	}

	hasSnapshotFormat := false
	for _, col := range columns {
		if ContainsAny(col, snapshotKeywords) {
			hasSnapshotFormat = true
			break
		}
	}

	if !hasSnapshotFormat {
		return SheetRecognitionResult{
			SheetName:  sheetName,
			SheetType:  SheetTypeUnknown,
			Confidence: 0,
		}
	}

	// 判断是批零还是住餐快照
	hasSales := false
	hasRevenue := false

	for _, col := range columns {
		if strings.Contains(col, "销售额") || strings.Contains(col, "零售额") {
			hasSales = true
		}
		if strings.Contains(col, "营业额") || strings.Contains(col, "客房") || strings.Contains(col, "餐费") {
			hasRevenue = true
		}
	}

	confidence := 0.8
	if strings.Contains(sheetName, "批零") || strings.Contains(sheetName, "批发") || strings.Contains(sheetName, "零售") {
		confidence += 0.2
	}
	if strings.Contains(sheetName, "住餐") || strings.Contains(sheetName, "住宿") || strings.Contains(sheetName, "餐饮") {
		confidence += 0.2
	}

	// 判断类型
	sheetType := SheetTypeWRSnapshot
	if hasRevenue && !hasSales {
		sheetType = SheetTypeACSnapshot
	}

	// 从 Sheet 名提取年月
	year, month, found := ExtractYearMonth(sheetName)
	if found {
		return SheetRecognitionResult{
			SheetName:  sheetName,
			SheetType:  sheetType,
			Confidence: confidence,
			DataYear:   year,
			DataMonth:  month,
		}
	}

	return SheetRecognitionResult{
		SheetName:  sheetName,
		SheetType:  sheetType,
		Confidence: confidence * 0.8, // 没有年月信息，降低置信度
	}
}

// recognizeSummary 识别汇总表
func (r *SheetRecognizer) recognizeSummary(sheetName string, columns []string) SheetRecognitionResult {
	summaryKeywords := []string{
		"限上零售额",
		"限下",
		"小微",
		"吃穿用",
		"汇总",
		"社零",
		"增速",
	}

	// Sheet 名称包含汇总关键词
	nameMatch := false
	for _, kw := range summaryKeywords {
		if strings.Contains(sheetName, kw) {
			nameMatch = true
			break
		}
	}

	if nameMatch {
		return SheetRecognitionResult{
			SheetName:  sheetName,
			SheetType:  SheetTypeSummary,
			Confidence: 0.8,
		}
	}

	return SheetRecognitionResult{
		SheetName:  sheetName,
		SheetType:  SheetTypeUnknown,
		Confidence: 0,
	}
}

// RecognizeIndustryType 根据行业代码识别具体行业类型
func RecognizeIndustryType(industryCode string) string {
	if len(industryCode) < 2 {
		return "unknown"
	}

	prefix := industryCode[:2]
	switch prefix {
	case "51":
		return "wholesale"
	case "52":
		return "retail"
	case "61":
		return "accommodation"
	case "62":
		return "catering"
	default:
		return "unknown"
	}
}
