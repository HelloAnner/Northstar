package parser

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
	"northstar/internal/model"
)

// ACParser 住餐主表解析器
type ACParser struct {
	file         *excelize.File
	recognizer   *SheetRecognizer
	currentYear  int
	currentMonth int
}

// NewACParser 创建住餐解析器
func NewACParser(file *excelize.File) *ACParser {
	return &ACParser{
		file:       file,
		recognizer: NewSheetRecognizer(),
	}
}

// ParseSheet 解析住餐 Sheet
func (p *ACParser) ParseSheet(sheetName string) ([]*model.AccommodationCatering, error) {
	// 读取所有行
	rows, err := p.file.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet: %w", err)
	}

	if len(rows) < 2 {
		return nil, fmt.Errorf("sheet has no data rows")
	}

	// 第一行是表头
	headers := rows[0]

	// 识别 Sheet 类型
	result := p.recognizer.Recognize(sheetName, headers)
	if result.SheetType != SheetTypeAccommodation && result.SheetType != SheetTypeCatering {
		return nil, fmt.Errorf("not an accommodation/catering sheet")
	}

	// 从列名中识别当前年月
	year, month := FindCurrentYearMonth(headers)
	if year == 0 || month == 0 {
		return nil, fmt.Errorf("cannot determine data year/month from columns")
	}

	p.currentYear = year
	p.currentMonth = month

	// 创建字段映射器
	mapper := NewFieldMapper(year, month)
	mappings := mapper.MapAccommodationCatering(headers)

	// 解析数据行
	var records []*model.AccommodationCatering
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		record := p.parseACRow(row, mappings, sheetName, rowIdx+1)
		if record != nil {
			record.DataYear = year
			record.DataMonth = month
			records = append(records, record)
		}
	}

	return records, nil
}

// parseACRow 解析单行数据
func (p *ACParser) parseACRow(row []string, mappings map[int]FieldMapping, sheetName string, rowNo int) *model.AccommodationCatering {
	record := &model.AccommodationCatering{
		RowNo:       rowNo,
		SourceSheet: sheetName,
	}

	// 遍历所有映射
	for colIdx, mapping := range mappings {
		if colIdx >= len(row) {
			continue
		}

		value := strings.TrimSpace(row[colIdx])
		if value == "" {
			continue
		}

		// 根据字段名设置值
		p.setACFieldValue(record, mapping.DBField, value)
	}

	// 验证必填字段
	if record.Name == "" {
		return nil // 跳过无名称的行
	}

	// 备份原始值
	if record.RevenueCurrentMonth != 0 {
		val := record.RevenueCurrentMonth
		record.OriginalRevenueCurrentMonth = &val
	}
	if record.RoomCurrentMonth != 0 {
		val := record.RoomCurrentMonth
		record.OriginalRoomCurrentMonth = &val
	}
	if record.FoodCurrentMonth != 0 {
		val := record.FoodCurrentMonth
		record.OriginalFoodCurrentMonth = &val
	}
	if record.GoodsCurrentMonth != 0 {
		val := record.GoodsCurrentMonth
		record.OriginalGoodsCurrentMonth = &val
	}

	return record
}

// setACFieldValue 设置字段值
func (p *ACParser) setACFieldValue(record *model.AccommodationCatering, field, value string) {
	switch field {
	case "credit_code":
		record.CreditCode = value
	case "name":
		record.Name = value
	case "industry_code":
		record.IndustryCode = value
	case "company_scale":
		record.CompanyScale = parseInt(value)

	// 营业额
	case "revenue_prev_month":
		record.RevenuePrevMonth = parseFloat(value)
	case "revenue_current_month":
		record.RevenueCurrentMonth = parseFloat(value)
	case "revenue_last_year_month":
		record.RevenueLastYearMonth = parseFloat(value)
	case "revenue_prev_cumulative":
		record.RevenuePrevCumulative = parseFloat(value)
	case "revenue_current_cumulative":
		record.RevenueCurrentCumulative = parseFloat(value)
	case "revenue_last_year_cumulative":
		record.RevenueLastYearCumulative = parseFloat(value)

	// 客房收入
	case "room_prev_month":
		record.RoomPrevMonth = parseFloat(value)
	case "room_current_month":
		record.RoomCurrentMonth = parseFloat(value)
	case "room_last_year_month":
		record.RoomLastYearMonth = parseFloat(value)
	case "room_prev_cumulative":
		record.RoomPrevCumulative = parseFloat(value)
	case "room_current_cumulative":
		record.RoomCurrentCumulative = parseFloat(value)
	case "room_last_year_cumulative":
		record.RoomLastYearCumulative = parseFloat(value)

	// 餐费收入
	case "food_prev_month":
		record.FoodPrevMonth = parseFloat(value)
	case "food_current_month":
		record.FoodCurrentMonth = parseFloat(value)
	case "food_last_year_month":
		record.FoodLastYearMonth = parseFloat(value)
	case "food_prev_cumulative":
		record.FoodPrevCumulative = parseFloat(value)
	case "food_current_cumulative":
		record.FoodCurrentCumulative = parseFloat(value)
	case "food_last_year_cumulative":
		record.FoodLastYearCumulative = parseFloat(value)

	// 商品销售额
	case "goods_prev_month":
		record.GoodsPrevMonth = parseFloat(value)
	case "goods_current_month":
		record.GoodsCurrentMonth = parseFloat(value)
	case "goods_last_year_month":
		record.GoodsLastYearMonth = parseFloat(value)
	case "goods_prev_cumulative":
		record.GoodsPrevCumulative = parseFloat(value)
	case "goods_current_cumulative":
		record.GoodsCurrentCumulative = parseFloat(value)
	case "goods_last_year_cumulative":
		record.GoodsLastYearCumulative = parseFloat(value)

	// 零售额
	case "retail_current_month":
		record.RetailCurrentMonth = parseFloat(value)
	case "retail_last_year_month":
		record.RetailLastYearMonth = parseFloat(value)

	// 标记
	case "is_small_micro":
		record.IsSmallMicro = parseInt(value)
	case "is_eat_wear_use":
		record.IsEatWearUse = parseInt(value)

	// 补充字段
	case "first_report_ip":
		record.FirstReportIP = value
	case "fill_ip":
		record.FillIP = value
	case "network_sales":
		record.NetworkSales = parseFloat(value)
	case "opening_year":
		val := parseInt(value)
		record.OpeningYear = &val
	case "opening_month":
		val := parseInt(value)
		record.OpeningMonth = &val
	}
}
