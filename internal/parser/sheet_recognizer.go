package parser

import (
	"regexp"
	"strings"
)

var (
	yearMonthTokenPattern = regexp.MustCompile(`\d{4}年[^0-9]{0,3}0?\d{1,2}月`)
	yearRangeTokenPattern = regexp.MustCompile(`\d{4}年[^0-9]{0,3}0?\d{1,2}[-—]0?\d{1,2}月`)
)

// SheetRecognizer Sheet 类型识别器
type SheetRecognizer struct{}

// NewSheetRecognizer 创建识别器
func NewSheetRecognizer() *SheetRecognizer {
	return &SheetRecognizer{}
}

var (
	wrMainRequired = []string{
		"统一社会信用代?码",
		"单位详细名称|单位名称|企业名称",
		"行业代码",
		"YEAR_MONTH.*销售额",
		"YEAR_RANGE.*销售额",
		"YEAR_MONTH.*零售额",
		"YEAR_RANGE.*零售额",
		"单位规模",
	}
	acMainRequired = []string{
		"统一社会信用代?码",
		"单位详细名称|单位名称|企业名称",
		"行业代码",
		"YEAR_MONTH.*营业额",
		"YEAR_RANGE.*营业额",
		"YEAR_MONTH.*客房收入",
		"YEAR_MONTH.*餐费收入",
		"YEAR_MONTH.*销售额",
	}
	wrSnapshotRequired = []string{
		"统一社会信用代?码",
		"单位详细名称|单位名称|企业名称",
		"行业代码",
		"商品销售额.*本年-本月",
		"商品销售额.*本年-1-本月",
		"零售额.*本年-本月",
		"零售额.*本年-1-本月",
		"单位规模",
	}
	acSnapshotRequired = []string{
		"统一社会信用代?码",
		"单位详细名称|单位名称|企业名称",
		"行业代码",
		"营业额.*本年-本月",
		"营业额.*本年-1-本月",
		"客房收入.*本年-本月",
		"餐费收入.*本年-本月",
		"商品销售额.*本年-本月",
	}
	summaryRequired = []string{
		"限上零售额",
		"限下",
		"小微",
		"吃穿用",
		"增速",
	}
)

// Recognize 识别 Sheet 类型
func (r *SheetRecognizer) Recognize(sheetName string, columnNames []string) SheetRecognitionResult {
	// 规范化列名
	normalized := make([]string, len(columnNames))
	for i, col := range columnNames {
		normalized[i] = normalizeHeaderForRecognition(col)
	}

	// 尝试从 Sheet 名提取年月
	year, month, _ := ExtractYearMonth(sheetName)

	// 快照表优先：一旦命中 “本年-本月/上年-本月” 口径，不允许被主表误判
	if snap := r.recognizeSnapshot(sheetName, normalized); snap.Confidence >= 0.7 {
		snap.DataYear = year
		snap.DataMonth = month
		return snap
	}

	// 评分后择优（避免主表/快照/住餐被“先匹配”误判）
	candidates := []SheetRecognitionResult{
		r.recognizeAccommodationCatering(sheetName, normalized),
		r.recognizeWholesaleRetail(sheetName, normalized),
		r.recognizeSummary(sheetName, normalized),
	}

	best := SheetRecognitionResult{
		SheetName:  sheetName,
		SheetType:  SheetTypeUnknown,
		Confidence: 0,
	}
	for _, c := range candidates {
		if c.Confidence > best.Confidence {
			best = c
		}
	}

	// 置信度阈值（summary 允许更低一些）
	if best.SheetType == SheetTypeSummary {
		if best.Confidence < 0.3 {
			return SheetRecognitionResult{SheetName: sheetName, SheetType: SheetTypeUnknown, Confidence: 0}
		}
		return best
	}
	if best.Confidence < 0.5 {
		return SheetRecognitionResult{SheetName: sheetName, SheetType: SheetTypeUnknown, Confidence: 0}
	}

	// 主表/快照补充年月（对主表可能为 0；真正月份由列名推断）
	best.DataYear = year
	best.DataMonth = month
	return best
}

// recognizeWholesaleRetail 识别批发零售主表
func (r *SheetRecognizer) recognizeWholesaleRetail(sheetName string, columns []string) SheetRecognitionResult {
	confidence := scoreByRequiredHeaders(columns, wrMainRequired)
	nameBoost := 0.0
	sheetType := SheetTypeWholesale
	if strings.Contains(sheetName, "批发") {
		nameBoost = 0.2
		sheetType = SheetTypeWholesale
	}
	if strings.Contains(sheetName, "零售") {
		nameBoost = 0.2
		sheetType = SheetTypeRetail
	}
	if strings.Contains(sheetName, "批零") {
		nameBoost = 0.15
	}
	return SheetRecognitionResult{
		SheetName:  sheetName,
		SheetType:  sheetType,
		Confidence: clampConfidence(confidence + nameBoost),
	}
}

// recognizeAccommodationCatering 识别住宿餐饮主表
func (r *SheetRecognizer) recognizeAccommodationCatering(sheetName string, columns []string) SheetRecognitionResult {
	confidence := scoreByRequiredHeaders(columns, acMainRequired)
	nameBoost := 0.0
	sheetType := SheetTypeAccommodation
	if strings.Contains(sheetName, "住宿") {
		nameBoost = 0.2
		sheetType = SheetTypeAccommodation
	}
	if strings.Contains(sheetName, "餐饮") {
		nameBoost = 0.2
		sheetType = SheetTypeCatering
	}
	if strings.Contains(sheetName, "住餐") {
		nameBoost = 0.15
	}
	return SheetRecognitionResult{
		SheetName:  sheetName,
		SheetType:  sheetType,
		Confidence: clampConfidence(confidence + nameBoost),
	}
}

// recognizeSnapshot 识别快照表
func (r *SheetRecognizer) recognizeSnapshot(sheetName string, columns []string) SheetRecognitionResult {
	snapshotKeywords := []string{
		"本年-本月",
		"本年-1-本月",
		"上年-本月",
		"上年-1-本月",
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

	wrScore := scoreByRequiredHeaders(columns, wrSnapshotRequired)
	acScore := scoreByRequiredHeaders(columns, acSnapshotRequired)

	sheetType := SheetTypeWRSnapshot
	confidence := wrScore
	nameBoost := 0.0
	if acScore > wrScore {
		sheetType = SheetTypeACSnapshot
		confidence = acScore
	}
	if sheetType == SheetTypeWRSnapshot {
		if strings.Contains(sheetName, "批零") || strings.Contains(sheetName, "批发") || strings.Contains(sheetName, "零售") {
			nameBoost = 0.2
		}
	} else {
		if strings.Contains(sheetName, "住餐") || strings.Contains(sheetName, "住宿") || strings.Contains(sheetName, "餐饮") {
			nameBoost = 0.2
		}
	}

	year, month, found := ExtractYearMonth(sheetName)
	if found {
		return SheetRecognitionResult{
			SheetName:  sheetName,
			SheetType:  sheetType,
			Confidence: clampConfidence(confidence + nameBoost),
			DataYear:   year,
			DataMonth:  month,
		}
	}
	return SheetRecognitionResult{
		SheetName:  sheetName,
		SheetType:  sheetType,
		Confidence: clampConfidence(confidence + nameBoost),
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

	confidence := scoreByRequiredHeaders(columns, summaryRequired)
	for _, kw := range summaryKeywords {
		if strings.Contains(sheetName, kw) {
			if confidence < 0.7 {
				confidence = 0.7
			}
			break
		}
	}

	return SheetRecognitionResult{
		SheetName:  sheetName,
		SheetType:  SheetTypeSummary,
		Confidence: confidence,
	}
}

func normalizeHeaderForRecognition(name string) string {
	normalized := NormalizeColumnName(name)
	normalized = yearRangeTokenPattern.ReplaceAllString(normalized, "YEAR_RANGE")
	normalized = yearMonthTokenPattern.ReplaceAllString(normalized, "YEAR_MONTH")
	return normalized
}

func scoreByRequiredHeaders(columns []string, required []string) float64 {
	if len(required) == 0 {
		return 0
	}
	matchCount := 0
	for _, field := range required {
		for _, col := range columns {
			if MatchPattern(col, field) {
				matchCount++
				break
			}
		}
	}
	return float64(matchCount) / float64(len(required))
}

// RecognizeIndustryType 根据行业代码识别具体行业类型
func RecognizeIndustryType(industryCode string) string {
	code := normalizeIndustryCode(industryCode)
	if len(code) < 2 {
		return "unknown"
	}

	prefix := code[:2]
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

func clampConfidence(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
