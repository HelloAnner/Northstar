package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"northstar/internal/model"
)

// WRParser 批零主表解析器
type WRParser struct {
	file         *excelize.File
	recognizer   *SheetRecognizer
	currentYear  int
	currentMonth int
}

// NewWRParser 创建批零解析器
func NewWRParser(file *excelize.File) *WRParser {
	return &WRParser{
		file:       file,
		recognizer: NewSheetRecognizer(),
	}
}

// ParseSheet 解析批零 Sheet
func (p *WRParser) ParseSheet(sheetName string) ([]*model.WholesaleRetail, error) {
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
	if result.SheetType != SheetTypeWholesale && result.SheetType != SheetTypeRetail {
		return nil, fmt.Errorf("not a wholesale/retail sheet")
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
	mappings := mapper.MapWholesaleRetail(headers)

	// 解析数据行
	var records []*model.WholesaleRetail
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		record := p.parseWRRow(row, mappings, sheetName, rowIdx+1)
		if record != nil {
			record.DataYear = year
			record.DataMonth = month
			records = append(records, record)
		}
	}

	return records, nil
}

// parseWRRow 解析单行数据
func (p *WRParser) parseWRRow(row []string, mappings map[int]FieldMapping, sheetName string, rowNo int) *model.WholesaleRetail {
	record := &model.WholesaleRetail{
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
		p.setWRFieldValue(record, mapping.DBField, value)
	}

	// 验证必填字段
	if record.Name == "" {
		return nil // 跳过无名称的行
	}

	// 补齐行业类型（用于指标计算与过滤）
	if record.IndustryType == "" && record.IndustryCode != "" {
		record.IndustryType = RecognizeIndustryType(record.IndustryCode)
	}

	// 备份原始值
	if record.SalesCurrentMonth != 0 {
		val := record.SalesCurrentMonth
		record.OriginalSalesCurrentMonth = &val
	}
	if record.RetailCurrentMonth != 0 {
		val := record.RetailCurrentMonth
		record.OriginalRetailCurrentMonth = &val
	}

	return record
}

// setWRFieldValue 设置字段值
func (p *WRParser) setWRFieldValue(record *model.WholesaleRetail, field, value string) {
	switch field {
	case "credit_code":
		record.CreditCode = value
	case "name":
		record.Name = value
	case "industry_code":
		record.IndustryCode = value
	case "company_scale":
		record.CompanyScale = parseInt(value)
	case "retail_ratio":
		val := parseFloat(value)
		record.RetailRatio = &val

	// 销售额
	case "sales_prev_month":
		record.SalesPrevMonth = parseFloat(value)
	case "sales_current_month":
		record.SalesCurrentMonth = parseFloat(value)
	case "sales_last_year_month":
		record.SalesLastYearMonth = parseFloat(value)
	case "sales_prev_cumulative":
		record.SalesPrevCumulative = parseFloat(value)
	case "sales_last_year_prev_cumulative":
		record.SalesLastYearPrevCumulative = parseFloat(value)
	case "sales_current_cumulative":
		record.SalesCurrentCumulative = parseFloat(value)
	case "sales_last_year_cumulative":
		record.SalesLastYearCumulative = parseFloat(value)

	// 零售额
	case "retail_prev_month":
		record.RetailPrevMonth = parseFloat(value)
	case "retail_current_month":
		record.RetailCurrentMonth = parseFloat(value)
	case "retail_last_year_month":
		record.RetailLastYearMonth = parseFloat(value)
	case "retail_prev_cumulative":
		record.RetailPrevCumulative = parseFloat(value)
	case "retail_last_year_prev_cumulative":
		record.RetailLastYearPrevCumulative = parseFloat(value)
	case "retail_current_cumulative":
		record.RetailCurrentCumulative = parseFloat(value)
	case "retail_last_year_cumulative":
		record.RetailLastYearCumulative = parseFloat(value)

	// 商品分类
	case "cat_grain_oil_food":
		record.CatGrainOilFood = parseFloat(value)
	case "cat_beverage":
		record.CatBeverage = parseFloat(value)
	case "cat_tobacco_liquor":
		record.CatTobaccoLiquor = parseFloat(value)
	case "cat_clothing":
		record.CatClothing = parseFloat(value)
	case "cat_daily_use":
		record.CatDailyUse = parseFloat(value)
	case "cat_automobile":
		record.CatAutomobile = parseFloat(value)

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

// parseInt 安全转换为整数
func parseInt(s string) int {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "") // 移除千分位
	i, _ := strconv.Atoi(s)
	return i
}

// parseFloat 安全转换为浮点数
func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", "") // 移除千分位
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
