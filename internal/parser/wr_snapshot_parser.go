package parser

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
	"northstar/internal/model"
)

// WRSnapshotParser 批零快照解析器
type WRSnapshotParser struct {
	file *excelize.File
}

// NewWRSnapshotParser 创建批零快照解析器
func NewWRSnapshotParser(file *excelize.File) *WRSnapshotParser {
	return &WRSnapshotParser{file: file}
}

// ParseSheet 解析批零快照 Sheet
func (p *WRSnapshotParser) ParseSheet(sheetName string) ([]*model.WRSnapshot, error) {
	rows, err := p.file.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("sheet has no data rows")
	}

	year, month, found := ExtractYearMonth(sheetName)
	if !found {
		return nil, fmt.Errorf("cannot determine snapshot year/month from sheet name")
	}

	headers := rows[0]
	index := buildSnapshotIndex(headers)

	var records []*model.WRSnapshot
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		rec := &model.WRSnapshot{
			SnapshotYear:  year,
			SnapshotMonth: month,
			SnapshotName:  sheetName,
			SourceSheet:   sheetName,
		}

		setString(&rec.CreditCode, row, index.creditCode)
		setString(&rec.Name, row, index.name)
		setString(&rec.IndustryCode, row, index.industryCode)
		if index.companyScale >= 0 && index.companyScale < len(row) {
			rec.CompanyScale = parseInt(row[index.companyScale])
		}

		if rec.Name == "" {
			continue
		}

		setFloat(&rec.SalesCurrentMonth, row, index.salesCurrentMonth)
		setFloat(&rec.SalesCurrentCumulative, row, index.salesCurrentCumulative)
		setFloatPtr(&rec.SalesLastYearMonth, row, index.salesLastYearMonth)
		setFloatPtr(&rec.SalesLastYearCumulative, row, index.salesLastYearCumulative)

		setFloat(&rec.RetailCurrentMonth, row, index.retailCurrentMonth)
		setFloat(&rec.RetailCurrentCumulative, row, index.retailCurrentCumulative)
		setFloatPtr(&rec.RetailLastYearMonth, row, index.retailLastYearMonth)
		setFloatPtr(&rec.RetailLastYearCumulative, row, index.retailLastYearCumulative)

		setFloatPtr(&rec.CatGrainOilFood, row, index.catGrainOilFood)
		setFloatPtr(&rec.CatBeverage, row, index.catBeverage)
		setFloatPtr(&rec.CatTobaccoLiquor, row, index.catTobaccoLiquor)
		setFloatPtr(&rec.CatClothing, row, index.catClothing)
		setFloatPtr(&rec.CatDailyUse, row, index.catDailyUse)
		setFloatPtr(&rec.CatAutomobile, row, index.catAutomobile)

		records = append(records, rec)
	}

	return records, nil
}

type wrSnapshotIndex struct {
	creditCode int
	name       int
	industryCode int
	companyScale int

	salesCurrentMonth      int
	salesCurrentCumulative int
	salesLastYearMonth     int
	salesLastYearCumulative int

	retailCurrentMonth      int
	retailCurrentCumulative int
	retailLastYearMonth     int
	retailLastYearCumulative int

	catGrainOilFood  int
	catBeverage      int
	catTobaccoLiquor int
	catClothing      int
	catDailyUse      int
	catAutomobile    int
}

func buildSnapshotIndex(headers []string) wrSnapshotIndex {
	idx := wrSnapshotIndex{
		creditCode: -1,
		name: -1,
		industryCode: -1,
		companyScale: -1,
		salesCurrentMonth: -1,
		salesCurrentCumulative: -1,
		salesLastYearMonth: -1,
		salesLastYearCumulative: -1,
		retailCurrentMonth: -1,
		retailCurrentCumulative: -1,
		retailLastYearMonth: -1,
		retailLastYearCumulative: -1,
		catGrainOilFood: -1,
		catBeverage: -1,
		catTobaccoLiquor: -1,
		catClothing: -1,
		catDailyUse: -1,
		catAutomobile: -1,
	}

	for i, h := range headers {
		col := NormalizeColumnName(h)
		if col == "" {
			continue
		}

		// 基础信息
		if idx.creditCode < 0 && MatchPattern(col, `统一社会信用代码`) {
			idx.creditCode = i
			continue
		}
		if idx.name < 0 && MatchPattern(col, `单位详细名称|单位名称|企业名称`) {
			idx.name = i
			continue
		}
		if idx.industryCode < 0 && MatchPattern(col, `行业代码`) && !strings.Contains(col, "说明") {
			idx.industryCode = i
			continue
		}
		if idx.companyScale < 0 && MatchPattern(col, `单位规模`) {
			idx.companyScale = i
			continue
		}

		// 销售额
		if strings.Contains(col, "商品销售额") {
			if idx.salesCurrentMonth < 0 && strings.Contains(col, "本年-本月") {
				idx.salesCurrentMonth = i
				continue
			}
			if idx.salesCurrentCumulative < 0 && strings.Contains(col, "本年-1—本月") {
				idx.salesCurrentCumulative = i
				continue
			}
			if idx.salesLastYearMonth < 0 && strings.Contains(col, "上年-本月") {
				idx.salesLastYearMonth = i
				continue
			}
			if idx.salesLastYearCumulative < 0 && strings.Contains(col, "上年-1—本月") {
				idx.salesLastYearCumulative = i
				continue
			}
		}

		// 零售额
		if strings.Contains(col, "零售额") {
			if idx.retailCurrentMonth < 0 && strings.Contains(col, "本年-本月") {
				idx.retailCurrentMonth = i
				continue
			}
			if idx.retailCurrentCumulative < 0 && strings.Contains(col, "本年-1—本月") {
				idx.retailCurrentCumulative = i
				continue
			}
			if idx.retailLastYearMonth < 0 && strings.Contains(col, "上年-本月") {
				idx.retailLastYearMonth = i
				continue
			}
			if idx.retailLastYearCumulative < 0 && strings.Contains(col, "上年-1—本月") {
				idx.retailLastYearCumulative = i
				continue
			}
		}

		// 商品分类（只取本年-本月）
		if strings.Contains(col, "商品销售额-本年-本月") {
			switch {
			case idx.catGrainOilFood < 0 && (strings.Contains(col, "粮油") || strings.Contains(col, "食品")):
				idx.catGrainOilFood = i
			case idx.catBeverage < 0 && strings.Contains(col, "饮料"):
				idx.catBeverage = i
			case idx.catTobaccoLiquor < 0 && (strings.Contains(col, "烟") || strings.Contains(col, "酒")):
				idx.catTobaccoLiquor = i
			case idx.catClothing < 0 && strings.Contains(col, "服装"):
				idx.catClothing = i
			case idx.catDailyUse < 0 && strings.Contains(col, "日用品"):
				idx.catDailyUse = i
			case idx.catAutomobile < 0 && strings.Contains(col, "汽车"):
				idx.catAutomobile = i
			}
		}
	}

	return idx
}

func setString(dest *string, row []string, col int) {
	if col < 0 || col >= len(row) {
		return
	}
	v := strings.TrimSpace(row[col])
	if v == "" {
		return
	}
	*dest = v
}

func setFloat(dest *float64, row []string, col int) {
	if col < 0 || col >= len(row) {
		return
	}
	v := strings.TrimSpace(row[col])
	if v == "" {
		return
	}
	*dest = parseFloat(v)
}

func setFloatPtr(dest **float64, row []string, col int) {
	if col < 0 || col >= len(row) {
		return
	}
	v := strings.TrimSpace(row[col])
	if v == "" {
		return
	}
	val := parseFloat(v)
	*dest = &val
}

