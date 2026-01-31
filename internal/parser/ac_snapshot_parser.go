package parser

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
	"northstar/internal/model"
)

// ACSnapshotParser 住餐快照解析器
type ACSnapshotParser struct {
	file *excelize.File
}

// NewACSnapshotParser 创建住餐快照解析器
func NewACSnapshotParser(file *excelize.File) *ACSnapshotParser {
	return &ACSnapshotParser{file: file}
}

// ParseSheet 解析住餐快照 Sheet
func (p *ACSnapshotParser) ParseSheet(sheetName string) ([]*model.ACSnapshot, error) {
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
	index := buildACSnapshotIndex(headers)

	var records []*model.ACSnapshot
	for rowIdx := 1; rowIdx < len(rows); rowIdx++ {
		row := rows[rowIdx]
		rec := &model.ACSnapshot{
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

		setFloat(&rec.RevenueCurrentMonth, row, index.revenueCurrentMonth)
		setFloat(&rec.RevenueCurrentCumulative, row, index.revenueCurrentCumulative)

		setFloat(&rec.RoomCurrentMonth, row, index.roomCurrentMonth)
		setFloatPtr(&rec.RoomCurrentCumulative, row, index.roomCurrentCumulative)

		setFloat(&rec.FoodCurrentMonth, row, index.foodCurrentMonth)
		setFloatPtr(&rec.FoodCurrentCumulative, row, index.foodCurrentCumulative)

		setFloat(&rec.GoodsCurrentMonth, row, index.goodsCurrentMonth)
		setFloatPtr(&rec.GoodsCurrentCumulative, row, index.goodsCurrentCumulative)

		records = append(records, rec)
	}

	return records, nil
}

type acSnapshotIndex struct {
	creditCode   int
	name         int
	industryCode int
	companyScale int

	revenueCurrentMonth      int
	revenueCurrentCumulative int

	roomCurrentMonth      int
	roomCurrentCumulative int

	foodCurrentMonth      int
	foodCurrentCumulative int

	goodsCurrentMonth      int
	goodsCurrentCumulative int
}

func buildACSnapshotIndex(headers []string) acSnapshotIndex {
	idx := acSnapshotIndex{
		creditCode: -1,
		name: -1,
		industryCode: -1,
		companyScale: -1,
		revenueCurrentMonth: -1,
		revenueCurrentCumulative: -1,
		roomCurrentMonth: -1,
		roomCurrentCumulative: -1,
		foodCurrentMonth: -1,
		foodCurrentCumulative: -1,
		goodsCurrentMonth: -1,
		goodsCurrentCumulative: -1,
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

		// 营业额
		if strings.Contains(col, "营业额") {
			if idx.revenueCurrentMonth < 0 && strings.Contains(col, "本年-本月") {
				idx.revenueCurrentMonth = i
				continue
			}
			if idx.revenueCurrentCumulative < 0 && strings.Contains(col, "本年-1—本月") {
				idx.revenueCurrentCumulative = i
				continue
			}
		}

		// 客房收入
		if strings.Contains(col, "客房收入") {
			if idx.roomCurrentMonth < 0 && strings.Contains(col, "本年-本月") {
				idx.roomCurrentMonth = i
				continue
			}
			if idx.roomCurrentCumulative < 0 && strings.Contains(col, "本年-1—本月") {
				idx.roomCurrentCumulative = i
				continue
			}
		}

		// 餐费收入
		if strings.Contains(col, "餐费收入") {
			if idx.foodCurrentMonth < 0 && strings.Contains(col, "本年-本月") {
				idx.foodCurrentMonth = i
				continue
			}
			if idx.foodCurrentCumulative < 0 && strings.Contains(col, "本年-1—本月") {
				idx.foodCurrentCumulative = i
				continue
			}
		}

		// 商品销售额
		if strings.Contains(col, "商品销售额") {
			if idx.goodsCurrentMonth < 0 && strings.Contains(col, "本年-本月") {
				idx.goodsCurrentMonth = i
				continue
			}
			if idx.goodsCurrentCumulative < 0 && strings.Contains(col, "本年-1—本月") {
				idx.goodsCurrentCumulative = i
				continue
			}
		}
	}

	return idx
}

