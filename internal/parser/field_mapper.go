package parser

import (
	"strings"
)

// FieldMapper 字段映射器
type FieldMapper struct {
	currentYear  int
	currentMonth int
}

// NewFieldMapper 创建字段映射器
func NewFieldMapper(currentYear, currentMonth int) *FieldMapper {
	return &FieldMapper{
		currentYear:  currentYear,
		currentMonth: currentMonth,
	}
}

// MapWholesaleRetail 映射批发零售字段
func (m *FieldMapper) MapWholesaleRetail(columnNames []string) map[int]FieldMapping {
	mappings := make(map[int]FieldMapping)

	for idx, col := range columnNames {
		col = NormalizeColumnName(col)
		if col == "" {
			continue
		}

		mapping := m.mapWRColumn(col, idx)
		if mapping.DBField != "" {
			mappings[idx] = mapping
		}
	}

	return mappings
}

// MapAccommodationCatering 映射住宿餐饮字段
func (m *FieldMapper) MapAccommodationCatering(columnNames []string) map[int]FieldMapping {
	mappings := make(map[int]FieldMapping)

	for idx, col := range columnNames {
		col = NormalizeColumnName(col)
		if col == "" {
			continue
		}

		mapping := m.mapACColumn(col, idx)
		if mapping.DBField != "" {
			mappings[idx] = mapping
		}
	}

	return mappings
}

// mapWRColumn 映射批零单个列
func (m *FieldMapper) mapWRColumn(col string, idx int) FieldMapping {
	mapping := FieldMapping{
		ColumnIndex: idx,
		ColumnName:  col,
	}

	// 跳过增速列（避免覆盖金额字段；增速由系统计算）
	if strings.Contains(col, "增速") {
		return mapping
	}

	// 基础信息字段
	if MatchPattern(col, `统一社会信用代码`) {
		mapping.DBField = "credit_code"
		return mapping
	}
	if MatchPattern(col, `单位详细名称|单位名称|企业名称`) {
		mapping.DBField = "name"
		return mapping
	}
	if MatchPattern(col, `行业代码`) && !strings.Contains(col, "说明") {
		mapping.DBField = "industry_code"
		return mapping
	}
	if MatchPattern(col, `单位规模`) {
		mapping.DBField = "company_scale"
		return mapping
	}
	if MatchPattern(col, `零售额占比`) {
		mapping.DBField = "retail_ratio"
		return mapping
	}

	// 时间敏感字段 - 销售额
	if strings.Contains(col, "销售额") {
		timeType := InferFieldTimeType(col, m.currentYear, m.currentMonth)
		mapping.TimeType = timeType
		mapping.DBField = m.mapSalesField(timeType)
		return mapping
	}

	// 时间敏感字段 - 零售额
	if strings.Contains(col, "零售额") {
		timeType := InferFieldTimeType(col, m.currentYear, m.currentMonth)
		mapping.TimeType = timeType
		mapping.DBField = m.mapRetailField(timeType)
		return mapping
	}

	// 商品分类
	if MatchPattern(col, `粮油食品类`) {
		mapping.DBField = "cat_grain_oil_food"
		return mapping
	}
	if MatchPattern(col, `饮料类`) {
		mapping.DBField = "cat_beverage"
		return mapping
	}
	if MatchPattern(col, `烟酒类`) {
		mapping.DBField = "cat_tobacco_liquor"
		return mapping
	}
	if MatchPattern(col, `服装鞋帽针纺织品类|服装`) {
		mapping.DBField = "cat_clothing"
		return mapping
	}
	if MatchPattern(col, `日用品类`) {
		mapping.DBField = "cat_daily_use"
		return mapping
	}
	if MatchPattern(col, `汽车类`) {
		mapping.DBField = "cat_automobile"
		return mapping
	}

	// 补充字段
	if MatchPattern(col, `小微企业`) {
		mapping.DBField = "is_small_micro"
		return mapping
	}
	if MatchPattern(col, `吃穿用`) {
		mapping.DBField = "is_eat_wear_use"
		return mapping
	}
	if MatchPattern(col, `第一次上报的?IP|首次上报IP`) {
		mapping.DBField = "first_report_ip"
		return mapping
	}
	if MatchPattern(col, `填报IP`) {
		mapping.DBField = "fill_ip"
		return mapping
	}
	if MatchPattern(col, `网络销售额`) {
		mapping.DBField = "network_sales"
		return mapping
	}
	if MatchPattern(col, `开业年份`) {
		mapping.DBField = "opening_year"
		return mapping
	}
	if MatchPattern(col, `开业月份`) {
		mapping.DBField = "opening_month"
		return mapping
	}

	return mapping
}

// mapACColumn 映射住餐单个列
func (m *FieldMapper) mapACColumn(col string, idx int) FieldMapping {
	mapping := FieldMapping{
		ColumnIndex: idx,
		ColumnName:  col,
	}

	// 跳过增速列（避免覆盖金额字段；增速由系统计算）
	if strings.Contains(col, "增速") {
		return mapping
	}

	// 基础信息字段
	if MatchPattern(col, `统一社会信用代码`) {
		mapping.DBField = "credit_code"
		return mapping
	}
	if MatchPattern(col, `单位详细名称|单位名称|企业名称`) {
		mapping.DBField = "name"
		return mapping
	}
	if MatchPattern(col, `行业代码`) && !strings.Contains(col, "说明") {
		mapping.DBField = "industry_code"
		return mapping
	}
	if MatchPattern(col, `单位规模`) {
		mapping.DBField = "company_scale"
		return mapping
	}

	// 营业额
	if strings.Contains(col, "营业额") {
		timeType := InferFieldTimeType(col, m.currentYear, m.currentMonth)
		mapping.TimeType = timeType
		mapping.DBField = m.mapRevenueField(timeType)
		return mapping
	}

	// 客房收入
	if strings.Contains(col, "客房收入") || strings.Contains(col, "客房") {
		timeType := InferFieldTimeType(col, m.currentYear, m.currentMonth)
		mapping.TimeType = timeType
		mapping.DBField = m.mapRoomField(timeType)
		return mapping
	}

	// 餐费收入
	if strings.Contains(col, "餐费收入") || strings.Contains(col, "餐费") {
		timeType := InferFieldTimeType(col, m.currentYear, m.currentMonth)
		mapping.TimeType = timeType
		mapping.DBField = m.mapFoodField(timeType)
		return mapping
	}

	// 商品销售额
	if strings.Contains(col, "商品销售额") || (strings.Contains(col, "销售额") && !strings.Contains(col, "网络")) {
		timeType := InferFieldTimeType(col, m.currentYear, m.currentMonth)
		mapping.TimeType = timeType
		mapping.DBField = m.mapGoodsField(timeType)
		return mapping
	}

	// 零售额
	if strings.Contains(col, "零售额") {
		timeType := InferFieldTimeType(col, m.currentYear, m.currentMonth)
		mapping.TimeType = timeType
		// 住餐只有当月和去年同期
		if timeType == CurrentMonth {
			mapping.DBField = "retail_current_month"
		} else if timeType == LastYearMonth {
			mapping.DBField = "retail_last_year_month"
		}
		return mapping
	}

	// 补充字段
	if MatchPattern(col, `小微企业`) {
		mapping.DBField = "is_small_micro"
		return mapping
	}
	if MatchPattern(col, `吃穿用`) {
		mapping.DBField = "is_eat_wear_use"
		return mapping
	}
	if MatchPattern(col, `第一次上报的?IP|首次上报IP`) {
		mapping.DBField = "first_report_ip"
		return mapping
	}
	if MatchPattern(col, `填报IP`) {
		mapping.DBField = "fill_ip"
		return mapping
	}
	if MatchPattern(col, `网络销售额`) {
		mapping.DBField = "network_sales"
		return mapping
	}
	if MatchPattern(col, `开业年份`) {
		mapping.DBField = "opening_year"
		return mapping
	}
	if MatchPattern(col, `开业月份`) {
		mapping.DBField = "opening_month"
		return mapping
	}

	return mapping
}

// mapSalesField 映射销售额字段
func (m *FieldMapper) mapSalesField(timeType FieldTimeType) string {
	switch timeType {
	case CurrentMonth:
		return "sales_current_month"
	case PrevMonth:
		return "sales_prev_month"
	case LastYearMonth:
		return "sales_last_year_month"
	case CurrentCumulative:
		return "sales_current_cumulative"
	case PrevCumulative:
		return "sales_prev_cumulative"
	case LastYearPrevCumulative:
		return "sales_last_year_prev_cumulative"
	case LastYearCumulative:
		return "sales_last_year_cumulative"
	}
	return ""
}

// mapRetailField 映射零售额字段
func (m *FieldMapper) mapRetailField(timeType FieldTimeType) string {
	switch timeType {
	case CurrentMonth:
		return "retail_current_month"
	case PrevMonth:
		return "retail_prev_month"
	case LastYearMonth:
		return "retail_last_year_month"
	case CurrentCumulative:
		return "retail_current_cumulative"
	case PrevCumulative:
		return "retail_prev_cumulative"
	case LastYearPrevCumulative:
		return "retail_last_year_prev_cumulative"
	case LastYearCumulative:
		return "retail_last_year_cumulative"
	}
	return ""
}

// mapRevenueField 映射营业额字段
func (m *FieldMapper) mapRevenueField(timeType FieldTimeType) string {
	switch timeType {
	case CurrentMonth:
		return "revenue_current_month"
	case PrevMonth:
		return "revenue_prev_month"
	case LastYearMonth:
		return "revenue_last_year_month"
	case CurrentCumulative:
		return "revenue_current_cumulative"
	case PrevCumulative:
		return "revenue_prev_cumulative"
	case LastYearCumulative:
		return "revenue_last_year_cumulative"
	}
	return ""
}

// mapRoomField 映射客房收入字段
func (m *FieldMapper) mapRoomField(timeType FieldTimeType) string {
	switch timeType {
	case CurrentMonth:
		return "room_current_month"
	case PrevMonth:
		return "room_prev_month"
	case LastYearMonth:
		return "room_last_year_month"
	case CurrentCumulative:
		return "room_current_cumulative"
	case PrevCumulative:
		return "room_prev_cumulative"
	case LastYearCumulative:
		return "room_last_year_cumulative"
	}
	return ""
}

// mapFoodField 映射餐费收入字段
func (m *FieldMapper) mapFoodField(timeType FieldTimeType) string {
	switch timeType {
	case CurrentMonth:
		return "food_current_month"
	case PrevMonth:
		return "food_prev_month"
	case LastYearMonth:
		return "food_last_year_month"
	case CurrentCumulative:
		return "food_current_cumulative"
	case PrevCumulative:
		return "food_prev_cumulative"
	case LastYearCumulative:
		return "food_last_year_cumulative"
	}
	return ""
}

// mapGoodsField 映射商品销售额字段
func (m *FieldMapper) mapGoodsField(timeType FieldTimeType) string {
	switch timeType {
	case CurrentMonth:
		return "goods_current_month"
	case PrevMonth:
		return "goods_prev_month"
	case LastYearMonth:
		return "goods_last_year_month"
	case CurrentCumulative:
		return "goods_current_cumulative"
	case PrevCumulative:
		return "goods_prev_cumulative"
	case LastYearCumulative:
		return "goods_last_year_cumulative"
	}
	return ""
}
