package excel

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"northstar/internal/model"
)

// MonthReportExporter 月报（定）模板导出：按固定坐标写入数据，保留样式/公式/合并
type MonthReportExporter struct {
	wb *excelize.File
}

// NewMonthReportExporter 创建月报导出器
func NewMonthReportExporter(wb *excelize.File) *MonthReportExporter {
	return &MonthReportExporter{wb: wb}
}

type industrySums struct {
	SalesCur     float64
	SalesLast    float64
	SalesCurCum  float64
	SalesLastCum float64

	RetailCur     float64
	RetailLast    float64
	RetailCurCum  float64
	RetailLastCum float64
}

// FillFromCompanies 将当前内存数据写入“12月月报（定）.xlsx”模板
func (e *MonthReportExporter) FillFromCompanies(companies []*model.Company) error {
	if e == nil || e.wb == nil {
		return errors.New("workbook is nil")
	}

	byKey := make(map[string]*model.Company, len(companies))
	for _, c := range companies {
		if c == nil {
			continue
		}
		key := companyInvariantKey(c)
		byKey[key] = c
	}

	// 1) 行业主表
	whSums, err := e.writeWholesaleRetailSheet("批发", byKey)
	if err != nil {
		return err
	}
	reSums, err := e.writeWholesaleRetailSheet("零售", byKey)
	if err != nil {
		return err
	}
	accSums, err := e.writeAccommodationCateringSheet("住宿", byKey)
	if err != nil {
		return err
	}
	catSums, err := e.writeAccommodationCateringSheet("餐饮", byKey)
	if err != nil {
		return err
	}

	// 2) 总表（严格按模板行顺序回写）
	if err := e.writeTotalSheets(byKey); err != nil {
		return err
	}

	// 3) 批发表的全局零售额汇总区（依赖四行业汇总）
	if err := e.writeOverallRetailOnWholesaleSheet(whSums, reSums, accSums, catSums); err != nil {
		return err
	}

	// 4) 汇总表（定）关键单元格（展示口径：万元、1位小数）
	if err := e.writeFixedSummarySheet(); err != nil {
		return err
	}

	// 5) 视图类表格（吃穿用 / 小微）
	// 当前策略：严格保留模板原样（不自动重算），避免在无完整规则实现前破坏 1:1 模板一致性。

	return nil
}

func (e *MonthReportExporter) writeWholesaleRetailSheet(sheetName string, byKey map[string]*model.Company) (industrySums, error) {
	ensureSheet(e.wb, sheetName)

	maxRow, err := findMaxDataRow(e.wb, sheetName, "C", 2)
	if err != nil {
		return industrySums{}, err
	}

	for row := 2; row <= maxRow; row++ {
		// 行不一定有信用代码/名称，但一定有行业代码
		industryCode, _ := e.wb.GetCellValue(sheetName, fmt.Sprintf("C%d", row))
		industryCode = strings.TrimSpace(industryCode)
		if industryCode == "" {
			continue
		}

		// 若模板缺少用于匹配的关键字段（全空），跳过，避免 key 碰撞导致错误覆盖。
		if isAllBlankCells(e.wb, sheetName,
			fmt.Sprintf("E%d", row),
			fmt.Sprintf("H%d", row),
			fmt.Sprintf("K%d", row),
			fmt.Sprintf("N%d", row),
		) || isAllZeroCells(e.wb, sheetName,
			fmt.Sprintf("E%d", row),
			fmt.Sprintf("H%d", row),
			fmt.Sprintf("K%d", row),
			fmt.Sprintf("N%d", row),
		) {
			continue
		}

		salesLast, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("E%d", row))
		salesLastCum, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("H%d", row))
		retailLast, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("K%d", row))
		retailLastCum, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("N%d", row))

		t := detectIndustryType(industryCode)
		key := invariantKeyFromParts(t, industryCode, salesLast, salesLastCum, retailLast, retailLastCum)
		c := byKey[key]
		if c == nil {
			continue
		}

		// D/G/J/M：本期；E/H/K/N：上年同期/上年累计
		if err := setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("D%d", row), c.SalesCurrentMonth); err != nil {
			return industrySums{}, err
		}
		if err := setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("G%d", row), c.SalesCurrentCumulative); err != nil {
			return industrySums{}, err
		}
		if err := setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("J%d", row), c.RetailCurrentMonth); err != nil {
			return industrySums{}, err
		}
		if err := setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("M%d", row), c.RetailCurrentCumulative); err != nil {
			return industrySums{}, err
		}

		// F/I/L/O：增速（%）
		if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("F%d", row), round2(ratePercent(c.SalesCurrentMonth, c.SalesLastYearMonth))); err != nil {
			return industrySums{}, err
		}
		if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("I%d", row), round2(ratePercent(c.SalesCurrentCumulative, c.SalesLastYearCumulative))); err != nil {
			return industrySums{}, err
		}
		if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("L%d", row), round2(ratePercent(c.RetailCurrentMonth, c.RetailLastYearMonth))); err != nil {
			return industrySums{}, err
		}
		if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("O%d", row), round2(ratePercent(c.RetailCurrentCumulative, c.RetailLastYearCumulative))); err != nil {
			return industrySums{}, err
		}

	}

	sums, err := sumWholesaleRetailSheetValues(e.wb, sheetName, maxRow)
	if err != nil {
		return industrySums{}, err
	}

	// 写行业汇总（模板：maxRow+1 为合计行，maxRow+2 为增速行）
	sumRow := maxRow + 1
	growthRow := maxRow + 2

	// 合计：销售额在 D/E/G/H，零售额在 J/K/M/N
	if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("D%d", sumRow), sums.SalesCur); err != nil {
		return industrySums{}, err
	}
	if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("E%d", sumRow), sums.SalesLast); err != nil {
		return industrySums{}, err
	}
	if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("G%d", sumRow), sums.SalesCurCum); err != nil {
		return industrySums{}, err
	}
	if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("H%d", sumRow), sums.SalesLastCum); err != nil {
		return industrySums{}, err
	}
	if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("J%d", sumRow), sums.RetailCur); err != nil {
		return industrySums{}, err
	}
	if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("K%d", sumRow), sums.RetailLast); err != nil {
		return industrySums{}, err
	}
	if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("M%d", sumRow), sums.RetailCurCum); err != nil {
		return industrySums{}, err
	}
	if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("N%d", sumRow), sums.RetailLastCum); err != nil {
		return industrySums{}, err
	}

	// 增速行：模板里只填 E/H（销售额当月/累计增速）
	if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("E%d", growthRow), round2(ratePercent(sums.SalesCur, sums.SalesLast))); err != nil {
		return industrySums{}, err
	}
	if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("H%d", growthRow), round2(ratePercent(sums.SalesCurCum, sums.SalesLastCum))); err != nil {
		return industrySums{}, err
	}

	return sums, nil
}

func (e *MonthReportExporter) writeAccommodationCateringSheet(sheetName string, byKey map[string]*model.Company) (industrySums, error) {
	ensureSheet(e.wb, sheetName)

	maxRow, err := findMaxDataRow(e.wb, sheetName, "C", 2)
	if err != nil {
		return industrySums{}, err
	}

	for row := 2; row <= maxRow; row++ {
		industryCode, _ := e.wb.GetCellValue(sheetName, fmt.Sprintf("C%d", row))
		industryCode = strings.TrimSpace(industryCode)
		if industryCode == "" {
			continue
		}

		if isAllBlankCells(e.wb, sheetName,
			fmt.Sprintf("E%d", row),
			fmt.Sprintf("H%d", row),
			fmt.Sprintf("W%d", row),
			fmt.Sprintf("Y%d", row),
		) || isAllZeroCells(e.wb, sheetName,
			fmt.Sprintf("E%d", row),
			fmt.Sprintf("H%d", row),
			fmt.Sprintf("W%d", row),
			fmt.Sprintf("Y%d", row),
		) {
			continue
		}

		revLast, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("E%d", row))
		revLastCum, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("H%d", row))

		// 住餐行业的“社零口径零售额”落在 V/W/X/Y（22-25）
		retailLast, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("W%d", row))
		retailLastCum, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("Y%d", row))

		t := detectIndustryType(industryCode)
		key := invariantKeyFromParts(t, industryCode, revLast, revLastCum, retailLast, retailLastCum)
		c := byKey[key]
		if c == nil {
			continue
		}

		// 营业额（Sales* 口径）
		if err := setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("D%d", row), c.SalesCurrentMonth); err != nil {
			return industrySums{}, err
		}
		if err := setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("G%d", row), c.SalesCurrentCumulative); err != nil {
			return industrySums{}, err
		}
		if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("F%d", row), round2(ratePercent(c.SalesCurrentMonth, c.SalesLastYearMonth))); err != nil {
			return industrySums{}, err
		}
		if err := e.wb.SetCellValue(sheetName, fmt.Sprintf("I%d", row), round2(ratePercent(c.SalesCurrentCumulative, c.SalesLastYearCumulative))); err != nil {
			return industrySums{}, err
		}

		// 客房收入
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("J%d", row), c.RoomRevenueCurrentMonth)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("L%d", row), c.RoomRevenueCurrentCumulative)

		// 餐费收入
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("N%d", row), c.FoodRevenueCurrentMonth)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("P%d", row), c.FoodRevenueCurrentCumulative)

		// 商品销售额
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("R%d", row), c.GoodsSalesCurrentMonth)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("T%d", row), c.GoodsSalesCurrentCumulative)

		// 社零口径零售额（餐费 + 商品销售额）写入 V/W/X/Y
		retCur := c.FoodRevenueCurrentMonth + c.GoodsSalesCurrentMonth
		retLast := c.FoodRevenueLastYearMonth + c.GoodsSalesLastYearMonth
		retCurCum := c.FoodRevenueCurrentCumulative + c.GoodsSalesCurrentCumulative
		retLastCum := c.FoodRevenueLastYearCumulative + c.GoodsSalesLastYearCumulative

		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("V%d", row), retCur)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("W%d", row), retLast)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("X%d", row), retCurCum)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("Y%d", row), retLastCum)
	}

	sums, err := sumAccommodationCateringSheetValues(e.wb, sheetName, maxRow)
	if err != nil {
		return industrySums{}, err
	}

	sumRow := maxRow + 1
	growthRow := maxRow + 2

	// 合计：营业额在 D/E/G/H；社零口径零售额在 V/W/X/Y
	_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("D%d", sumRow), sums.SalesCur)
	_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("E%d", sumRow), sums.SalesLast)
	_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("G%d", sumRow), sums.SalesCurCum)
	_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("H%d", sumRow), sums.SalesLastCum)
	_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("V%d", sumRow), sums.RetailCur)
	_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("W%d", sumRow), sums.RetailLast)
	_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("X%d", sumRow), sums.RetailCurCum)
	_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("Y%d", sumRow), sums.RetailLastCum)

	// 增速行：模板里只填 E/H（营业额当月/累计增速）
	_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("E%d", growthRow), round2(ratePercent(sums.SalesCur, sums.SalesLast)))
	_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("H%d", growthRow), round2(ratePercent(sums.SalesCurCum, sums.SalesLastCum)))

	return sums, nil
}

func sumWholesaleRetailSheetValues(wb *excelize.File, sheet string, maxRow int) (industrySums, error) {
	sums := industrySums{}
	for row := 2; row <= maxRow; row++ {
		industryCode, err := wb.GetCellValue(sheet, fmt.Sprintf("C%d", row))
		if err != nil {
			return industrySums{}, err
		}
		if strings.TrimSpace(industryCode) == "" {
			continue
		}

		sums.SalesCur += mustCellFloat(wb, sheet, fmt.Sprintf("D%d", row))
		sums.SalesLast += mustCellFloat(wb, sheet, fmt.Sprintf("E%d", row))
		sums.SalesCurCum += mustCellFloat(wb, sheet, fmt.Sprintf("G%d", row))
		sums.SalesLastCum += mustCellFloat(wb, sheet, fmt.Sprintf("H%d", row))

		sums.RetailCur += mustCellFloat(wb, sheet, fmt.Sprintf("J%d", row))
		sums.RetailLast += mustCellFloat(wb, sheet, fmt.Sprintf("K%d", row))
		sums.RetailCurCum += mustCellFloat(wb, sheet, fmt.Sprintf("M%d", row))
		sums.RetailLastCum += mustCellFloat(wb, sheet, fmt.Sprintf("N%d", row))
	}
	return sums, nil
}

func sumAccommodationCateringSheetValues(wb *excelize.File, sheet string, maxRow int) (industrySums, error) {
	sums := industrySums{}
	for row := 2; row <= maxRow; row++ {
		industryCode, err := wb.GetCellValue(sheet, fmt.Sprintf("C%d", row))
		if err != nil {
			return industrySums{}, err
		}
		if strings.TrimSpace(industryCode) == "" {
			continue
		}

		sums.SalesCur += mustCellFloat(wb, sheet, fmt.Sprintf("D%d", row))
		sums.SalesLast += mustCellFloat(wb, sheet, fmt.Sprintf("E%d", row))
		sums.SalesCurCum += mustCellFloat(wb, sheet, fmt.Sprintf("G%d", row))
		sums.SalesLastCum += mustCellFloat(wb, sheet, fmt.Sprintf("H%d", row))

		sums.RetailCur += mustCellFloat(wb, sheet, fmt.Sprintf("V%d", row))
		sums.RetailLast += mustCellFloat(wb, sheet, fmt.Sprintf("W%d", row))
		sums.RetailCurCum += mustCellFloat(wb, sheet, fmt.Sprintf("X%d", row))
		sums.RetailLastCum += mustCellFloat(wb, sheet, fmt.Sprintf("Y%d", row))
	}
	return sums, nil
}

func mustCellFloat(wb *excelize.File, sheet string, axis string) float64 {
	v, _ := getCellFloat(wb, sheet, axis)
	return v
}

func (e *MonthReportExporter) writeTotalSheets(byKey map[string]*model.Company) error {
	if err := e.writeWholesaleRetailTotalSheet("批零总表", byKey); err != nil {
		return err
	}
	if err := e.writeAccommodationCateringTotalSheet("住餐总表", byKey); err != nil {
		return err
	}
	return nil
}

func (e *MonthReportExporter) writeWholesaleRetailTotalSheet(sheetName string, byKey map[string]*model.Company) error {
	ensureSheet(e.wb, sheetName)

	maxRow, err := findMaxDataRow(e.wb, sheetName, "C", 2)
	if err != nil {
		return err
	}

	for row := 2; row <= maxRow; row++ {
		industryCode, _ := e.wb.GetCellValue(sheetName, fmt.Sprintf("C%d", row))
		industryCode = strings.TrimSpace(industryCode)
		if industryCode == "" {
			continue
		}

		if isAllBlankCells(e.wb, sheetName,
			fmt.Sprintf("E%d", row),
			fmt.Sprintf("H%d", row),
			fmt.Sprintf("K%d", row),
			fmt.Sprintf("N%d", row),
		) || isAllZeroCells(e.wb, sheetName,
			fmt.Sprintf("E%d", row),
			fmt.Sprintf("H%d", row),
			fmt.Sprintf("K%d", row),
			fmt.Sprintf("N%d", row),
		) {
			continue
		}

		salesLast, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("E%d", row))
		salesLastCum, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("H%d", row))
		retailLast, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("K%d", row))
		retailLastCum, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("N%d", row))

		t := detectIndustryType(industryCode)
		key := invariantKeyFromParts(t, industryCode, salesLast, salesLastCum, retailLast, retailLastCum)
		c := byKey[key]
		if c == nil {
			continue
		}

		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("D%d", row), c.SalesCurrentMonth)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("G%d", row), c.SalesCurrentCumulative)
		_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("F%d", row), round2(ratePercent(c.SalesCurrentMonth, c.SalesLastYearMonth)))
		_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("I%d", row), round2(ratePercent(c.SalesCurrentCumulative, c.SalesLastYearCumulative)))

		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("J%d", row), c.RetailCurrentMonth)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("M%d", row), c.RetailCurrentCumulative)
		_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("L%d", row), round2(ratePercent(c.RetailCurrentMonth, c.RetailLastYearMonth)))
		_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("O%d", row), round2(ratePercent(c.RetailCurrentCumulative, c.RetailLastYearCumulative)))
	}
	return nil
}

func (e *MonthReportExporter) writeAccommodationCateringTotalSheet(sheetName string, byKey map[string]*model.Company) error {
	ensureSheet(e.wb, sheetName)

	maxRow, err := findMaxDataRow(e.wb, sheetName, "C", 2)
	if err != nil {
		return err
	}

	for row := 2; row <= maxRow; row++ {
		industryCode, _ := e.wb.GetCellValue(sheetName, fmt.Sprintf("C%d", row))
		industryCode = strings.TrimSpace(industryCode)
		if industryCode == "" {
			continue
		}

		if isAllBlankCells(e.wb, sheetName,
			fmt.Sprintf("E%d", row),
			fmt.Sprintf("H%d", row),
			fmt.Sprintf("O%d", row),
			fmt.Sprintf("Q%d", row),
			fmt.Sprintf("S%d", row),
			fmt.Sprintf("U%d", row),
		) || isAllZeroCells(e.wb, sheetName,
			fmt.Sprintf("E%d", row),
			fmt.Sprintf("H%d", row),
			fmt.Sprintf("O%d", row),
			fmt.Sprintf("Q%d", row),
			fmt.Sprintf("S%d", row),
			fmt.Sprintf("U%d", row),
		) {
			continue
		}

		revLast, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("E%d", row))
		revLastCum, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("H%d", row))

		foodLast, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("O%d", row))
		foodLastCum, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("Q%d", row))
		goodsLast, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("S%d", row))
		goodsLastCum, _ := getCellFloat(e.wb, sheetName, fmt.Sprintf("U%d", row))

		retailLast := foodLast + goodsLast
		retailLastCum := foodLastCum + goodsLastCum

		t := detectIndustryType(industryCode)
		key := invariantKeyFromParts(t, industryCode, revLast, revLastCum, retailLast, retailLastCum)
		c := byKey[key]
		if c == nil {
			continue
		}

		// 营业额（Sales* 口径）
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("D%d", row), c.SalesCurrentMonth)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("G%d", row), c.SalesCurrentCumulative)
		_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("F%d", row), round2(ratePercent(c.SalesCurrentMonth, c.SalesLastYearMonth)))
		_ = e.wb.SetCellValue(sheetName, fmt.Sprintf("I%d", row), round2(ratePercent(c.SalesCurrentCumulative, c.SalesLastYearCumulative)))

		// 客房/餐费/商品销售额：住餐明细拆分
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("J%d", row), c.RoomRevenueCurrentMonth)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("L%d", row), c.RoomRevenueCurrentCumulative)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("N%d", row), c.FoodRevenueCurrentMonth)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("P%d", row), c.FoodRevenueCurrentCumulative)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("R%d", row), c.GoodsSalesCurrentMonth)
		_ = setFloatPreserveBlank(e.wb, sheetName, fmt.Sprintf("T%d", row), c.GoodsSalesCurrentCumulative)
	}
	return nil
}

func (e *MonthReportExporter) writeOverallRetailOnWholesaleSheet(wh industrySums, re industrySums, acc industrySums, cat industrySums) error {
	sheet := "批发"
	ensureSheet(e.wb, sheet)

	whMax, err := findMaxDataRow(e.wb, sheet, "C", 2)
	if err != nil {
		return err
	}

	// 批发模板：maxRow=24
	sumRow := whMax + 1 // 25
	growthRow := whMax + 2

	// 住餐行业的“社零口径零售额”合计，填到批发汇总区（J/K/M/N 的 2 行）
	accRow := growthRow + 1
	catRow := growthRow + 2
	totalRow := growthRow + 3
	totalGrowthRow := growthRow + 4

	// 零售行业合计（来自零售 sheet 合计行）
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("J%d", growthRow), re.RetailCur)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("K%d", growthRow), re.RetailLast)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("M%d", growthRow), re.RetailCurCum)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("N%d", growthRow), re.RetailLastCum)

	// 住宿/餐饮零售口径（社零贡献）写到下一行
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("J%d", accRow), acc.RetailCur)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("K%d", accRow), acc.RetailLast)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("M%d", accRow), acc.RetailCurCum)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("N%d", accRow), acc.RetailLastCum)

	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("J%d", catRow), cat.RetailCur)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("K%d", catRow), cat.RetailLast)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("M%d", catRow), cat.RetailCurCum)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("N%d", catRow), cat.RetailLastCum)

	overallRetailCur := wh.RetailCur + re.RetailCur + acc.RetailCur + cat.RetailCur
	overallRetailLast := wh.RetailLast + re.RetailLast + acc.RetailLast + cat.RetailLast
	overallRetailCurCum := wh.RetailCurCum + re.RetailCurCum + acc.RetailCurCum + cat.RetailCurCum
	overallRetailLastCum := wh.RetailLastCum + re.RetailLastCum + acc.RetailLastCum + cat.RetailLastCum

	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("J%d", totalRow), overallRetailCur)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("K%d", totalRow), overallRetailLast)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("M%d", totalRow), overallRetailCurCum)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("N%d", totalRow), overallRetailLastCum)

	// 模板把总体零售额增速写在 K/N（保持与原表一致）
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("K%d", totalGrowthRow), round2(ratePercent(overallRetailCur, overallRetailLast)))
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("N%d", totalGrowthRow), round2(ratePercent(overallRetailCurCum, overallRetailLastCum)))

	// 同时确保批发合计行（sumRow）内的零售合计为批发行业口径
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("J%d", sumRow), wh.RetailCur)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("K%d", sumRow), wh.RetailLast)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("M%d", sumRow), wh.RetailCurCum)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("N%d", sumRow), wh.RetailLastCum)

	return nil
}

func (e *MonthReportExporter) writeFixedSummarySheet() error {
	ensureSheet(e.wb, "汇总表（定）")

	// 数据来源：批发 sheet 的“全行业零售额合计行”
	whMax, err := findMaxDataRow(e.wb, "批发", "C", 2)
	if err != nil {
		return err
	}
	overallRow := whMax + 5 // totalRow = growthRow+3 = (whMax+2)+3 = whMax+5
	overallCur, _ := getCellFloat(e.wb, "批发", fmt.Sprintf("J%d", overallRow))
	overallLast, _ := getCellFloat(e.wb, "批发", fmt.Sprintf("K%d", overallRow))
	overallCurCum, _ := getCellFloat(e.wb, "批发", fmt.Sprintf("M%d", overallRow))
	overallLastCum, _ := getCellFloat(e.wb, "批发", fmt.Sprintf("N%d", overallRow))

	overallMonthRate := ratePercent(overallCur, overallLast)
	overallCumRate := ratePercent(overallCurCum, overallLastCum)

	// 万元口径
	_ = e.wb.SetCellValue("汇总表（定）", "G4", overallCur/10)
	_ = e.wb.SetCellValue("汇总表（定）", "H4", overallLast/10)
	_ = e.wb.SetCellValue("汇总表（定）", "I4", overallCurCum/10)
	_ = e.wb.SetCellValue("汇总表（定）", "J4", overallLastCum/10)

	// 四大行业增速（展示 1 位小数）
	whSalesRate, _ := getCellFloat(e.wb, "批发", fmt.Sprintf("E%d", whMax+2))
	reMax, err := findMaxDataRow(e.wb, "零售", "C", 2)
	if err != nil {
		return err
	}
	reSalesRate, _ := getCellFloat(e.wb, "零售", fmt.Sprintf("E%d", reMax+2))
	accMax, err := findMaxDataRow(e.wb, "住宿", "C", 2)
	if err != nil {
		return err
	}
	accSalesRate, _ := getCellFloat(e.wb, "住宿", fmt.Sprintf("E%d", accMax+2))
	catMax, err := findMaxDataRow(e.wb, "餐饮", "C", 2)
	if err != nil {
		return err
	}
	catSalesRate, _ := getCellFloat(e.wb, "餐饮", fmt.Sprintf("E%d", catMax+2))

	whCumRate, _ := getCellFloat(e.wb, "批发", fmt.Sprintf("H%d", whMax+2))
	reCumRate, _ := getCellFloat(e.wb, "零售", fmt.Sprintf("H%d", reMax+2))
	accCumRate, _ := getCellFloat(e.wb, "住宿", fmt.Sprintf("H%d", accMax+2))
	catCumRate, _ := getCellFloat(e.wb, "餐饮", fmt.Sprintf("H%d", catMax+2))

	_ = e.wb.SetCellValue("汇总表（定）", "O4", round1(accSalesRate))
	_ = e.wb.SetCellValue("汇总表（定）", "P4", round1(accCumRate))
	_ = e.wb.SetCellValue("汇总表（定）", "Q4", round1(catSalesRate))
	_ = e.wb.SetCellValue("汇总表（定）", "R4", round1(catCumRate))
	_ = e.wb.SetCellValue("汇总表（定）", "S4", round1(overallMonthRate))
	_ = e.wb.SetCellValue("汇总表（定）", "T4", round1(overallCumRate))

	// 批零两行业增速（模板位于 K4/M4，保留 1 位小数）
	_ = e.wb.SetCellValue("汇总表（定）", "K4", round1(whSalesRate))
	_ = e.wb.SetCellValue("汇总表（定）", "L4", round1(whCumRate))
	_ = e.wb.SetCellValue("汇总表（定）", "M4", round1(reSalesRate))
	_ = e.wb.SetCellValue("汇总表（定）", "N4", round1(reCumRate))

	return nil
}

func (e *MonthReportExporter) writeEatWearUseSheet(byKey map[string]*model.Company) error {
	sheet := "吃穿用"
	ensureSheet(e.wb, sheet)

	maxRow, err := findMaxDataRow(e.wb, sheet, "D", 2)
	if err != nil {
		return err
	}

	for row := 2; row <= maxRow; row++ {
		industryCode, _ := e.wb.GetCellValue(sheet, fmt.Sprintf("D%d", row))
		industryCode = strings.TrimSpace(industryCode)
		if industryCode == "" {
			continue
		}

		salesLast, _ := getCellFloat(e.wb, sheet, fmt.Sprintf("F%d", row))
		salesLastCum, _ := getCellFloat(e.wb, sheet, fmt.Sprintf("I%d", row))
		retailLast, _ := getCellFloat(e.wb, sheet, fmt.Sprintf("L%d", row))
		retailLastCum, _ := getCellFloat(e.wb, sheet, fmt.Sprintf("O%d", row))

		t := detectIndustryType(industryCode)
		key := invariantKeyFromParts(t, industryCode, salesLast, salesLastCum, retailLast, retailLastCum)
		c := byKey[key]
		if c == nil {
			continue
		}

		// Sales block: E-H (cur/last/rate/cumCur), I=lastCum, J=cumRate
		_ = e.wb.SetCellValue(sheet, fmt.Sprintf("E%d", row), c.SalesCurrentMonth)
		_ = e.wb.SetCellValue(sheet, fmt.Sprintf("G%d", row), round2(ratePercent(c.SalesCurrentMonth, c.SalesLastYearMonth)))
		_ = e.wb.SetCellValue(sheet, fmt.Sprintf("H%d", row), c.SalesCurrentCumulative)
		_ = e.wb.SetCellValue(sheet, fmt.Sprintf("J%d", row), round2(ratePercent(c.SalesCurrentCumulative, c.SalesLastYearCumulative)))

		// Retail block: K-N (cur/last/rate/cumCur), O=lastCum, P=cumRate
		_ = e.wb.SetCellValue(sheet, fmt.Sprintf("K%d", row), c.RetailCurrentMonth)
		_ = e.wb.SetCellValue(sheet, fmt.Sprintf("M%d", row), round2(ratePercent(c.RetailCurrentMonth, c.RetailLastYearMonth)))
		_ = e.wb.SetCellValue(sheet, fmt.Sprintf("N%d", row), c.RetailCurrentCumulative)
		_ = e.wb.SetCellValue(sheet, fmt.Sprintf("P%d", row), round2(ratePercent(c.RetailCurrentCumulative, c.RetailLastYearCumulative)))

		// 吃穿用：只对批零行业生效
		if c.IndustryType == model.IndustryWholesale || c.IndustryType == model.IndustryRetail {
			ewCur := 0.0
			ewLast := 0.0
			if c.IsEatWearUse {
				ewCur = c.RetailCurrentMonth
				ewLast = c.RetailLastYearMonth
			}
			_ = e.wb.SetCellValue(sheet, fmt.Sprintf("Q%d", row), ewCur)
			_ = e.wb.SetCellValue(sheet, fmt.Sprintf("R%d", row), ewLast)
			_ = e.wb.SetCellValue(sheet, fmt.Sprintf("S%d", row), round2(ratePercent(ewCur, ewLast)))
		}

		// 小微：只对批零行业生效（单位规模 3/4）
		scale, _ := getCellFloat(e.wb, sheet, fmt.Sprintf("T%d", row))
		microCur := 0.0
		microLast := 0.0
		if (c.IndustryType == model.IndustryWholesale || c.IndustryType == model.IndustryRetail) && (int(scale) == 3 || int(scale) == 4) {
			microCur = c.RetailCurrentMonth
			microLast = c.RetailLastYearMonth
		}
		_ = e.wb.SetCellValue(sheet, fmt.Sprintf("U%d", row), microCur)
		_ = e.wb.SetCellValue(sheet, fmt.Sprintf("V%d", row), microLast)
	}

	// 合计行：模板为 maxRow+1
	sumRow := maxRow + 1
	ewCurSum := 0.0
	ewLastSum := 0.0
	for row := 2; row <= maxRow; row++ {
		v, _ := getCellFloat(e.wb, sheet, fmt.Sprintf("Q%d", row))
		ewCurSum += v
		v, _ = getCellFloat(e.wb, sheet, fmt.Sprintf("R%d", row))
		ewLastSum += v
	}
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("Q%d", sumRow), ewCurSum)
	_ = e.wb.SetCellValue(sheet, fmt.Sprintf("R%d", sumRow), ewLastSum)

	return nil
}

func (e *MonthReportExporter) writeMicroSmallSheet() error {
	ensureSheet(e.wb, "小微")

	// 从“吃穿用”表的 U/V（计算用小微）导出；筛选条件：行业代码前两位为 51/52 且单位规模为 3/4
	ewSheet := "吃穿用"
	maxRow, err := findMaxDataRow(e.wb, ewSheet, "D", 2)
	if err != nil {
		return err
	}

	type pair struct{ cur, last float64 }
	rows := make([]pair, 0, 256)

	for row := 2; row <= maxRow; row++ {
		code, _ := e.wb.GetCellValue(ewSheet, fmt.Sprintf("D%d", row))
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if !strings.HasPrefix(code, "51") && !strings.HasPrefix(code, "52") {
			continue
		}
		scale, _ := getCellFloat(e.wb, ewSheet, fmt.Sprintf("T%d", row))
		if int(scale) != 3 && int(scale) != 4 {
			continue
		}

		cur, _ := getCellFloat(e.wb, ewSheet, fmt.Sprintf("U%d", row))
		last, _ := getCellFloat(e.wb, ewSheet, fmt.Sprintf("V%d", row))
		rows = append(rows, pair{cur: cur, last: last})
	}

	// 写入小微 sheet（从第2行开始）；模板末尾两行是“合计 + 增速”
	writeRow := 2
	sumCur := 0.0
	sumLast := 0.0
	for _, p := range rows {
		_ = e.wb.SetCellValue("小微", fmt.Sprintf("D%d", writeRow), p.cur)
		_ = e.wb.SetCellValue("小微", fmt.Sprintf("E%d", writeRow), p.last)
		_ = e.wb.SetCellValue("小微", fmt.Sprintf("F%d", writeRow), round2(ratePercent(p.cur, p.last)))

		sumCur += p.cur
		sumLast += p.last
		writeRow++
	}

	sumRow := writeRow
	growthRow := writeRow + 1

	_ = e.wb.SetCellValue("小微", fmt.Sprintf("D%d", sumRow), sumCur)
	_ = e.wb.SetCellValue("小微", fmt.Sprintf("E%d", sumRow), sumLast)
	_ = e.wb.SetCellValue("小微", fmt.Sprintf("E%d", growthRow), round2(ratePercent(sumCur, sumLast)))

	return nil
}

func ratePercent(current float64, lastYear float64) float64 {
	if lastYear == 0 {
		return -100
	}
	return (current/lastYear - 1) * 100
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

func findMaxDataRow(wb *excelize.File, sheet string, codeCol string, startRow int) (int, error) {
	max := startRow - 1
	for r := startRow; r <= 50000; r++ {
		v, err := wb.GetCellValue(sheet, fmt.Sprintf("%s%d", codeCol, r))
		if err != nil {
			return 0, err
		}
		if strings.TrimSpace(v) == "" {
			break
		}
		max = r
	}
	if max < startRow {
		return 0, fmt.Errorf("no data rows in sheet %s", sheet)
	}
	return max, nil
}

func getCellFloat(wb *excelize.File, sheet string, axis string) (float64, error) {
	raw, err := wb.GetCellValue(sheet, axis)
	if err != nil {
		return 0, err
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	f, parseErr := strconv.ParseFloat(raw, 64)
	if parseErr != nil {
		// 兼容整数
		i, intErr := strconv.Atoi(raw)
		if intErr != nil {
			return 0, nil
		}
		return float64(i), nil
	}
	return f, nil
}

func setFloatPreserveBlank(wb *excelize.File, sheet string, axis string, v float64) error {
	// 关键：模板里大量单元格是“空值”而不是 0。若写入 0，会导致 1:1 对比失败。
	// 因此当 v==0 且模板当前单元格为空时，保持空值不写。
	if v == 0 {
		raw, err := wb.GetCellValue(sheet, axis)
		if err != nil {
			return err
		}
		if strings.TrimSpace(raw) == "" {
			return nil
		}
	}
	return wb.SetCellValue(sheet, axis, v)
}

func isAllBlankCells(wb *excelize.File, sheet string, axes ...string) bool {
	for _, axis := range axes {
		raw, err := wb.GetCellValue(sheet, axis)
		if err != nil {
			return false
		}
		if strings.TrimSpace(raw) != "" {
			return false
		}
	}
	return true
}

func isAllZeroCells(wb *excelize.File, sheet string, axes ...string) bool {
	for _, axis := range axes {
		v, err := getCellFloat(wb, sheet, axis)
		if err != nil {
			return false
		}
		if v != 0 {
			return false
		}
	}
	return true
}

func copyRowValues(wb *excelize.File, srcSheet string, srcRow int, dstSheet string, dstRow int) error {
	cols, err := wb.GetCols(srcSheet)
	if err != nil {
		return err
	}

	// 逐列复制：保留模板样式/合并，本函数只写值；
	// 同时尽量保持数值类型（避免把数字写成字符串导致 e2e value 对比失败）。
	for ci := 1; ci <= len(cols); ci++ {
		srcCell, _ := excelize.CoordinatesToCellName(ci, srcRow)
		v, err := wb.GetCellValue(srcSheet, srcCell)
		if err != nil {
			return err
		}
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}

		dstCell, _ := excelize.CoordinatesToCellName(ci, dstRow)
		if f, ok := tryParseFloat(v); ok {
			if err := wb.SetCellValue(dstSheet, dstCell, f); err != nil {
				return err
			}
			continue
		}
		if err := wb.SetCellValue(dstSheet, dstCell, v); err != nil {
			return err
		}
	}
	return nil
}

func companyInvariantKey(c *model.Company) string {
	if c == nil {
		return ""
	}
	return invariantKeyFromParts(c.IndustryType, c.IndustryCode, c.SalesLastYearMonth, c.SalesLastYearCumulative, c.RetailLastYearMonth, c.RetailLastYearCumulative)
}

func invariantKeyFromParts(t model.IndustryType, industryCode string, salesLast float64, salesLastCum float64, retailLast float64, retailLastCum float64) string {
	sb := strings.Builder{}
	sb.WriteString(string(t))
	sb.WriteString("|")
	sb.WriteString(strings.TrimSpace(industryCode))
	sb.WriteString("|")
	sb.WriteString(floatKey(salesLast))
	sb.WriteString("|")
	sb.WriteString(floatKey(salesLastCum))
	sb.WriteString("|")
	sb.WriteString(floatKey(retailLast))
	sb.WriteString("|")
	sb.WriteString(floatKey(retailLastCum))
	return sb.String()
}

func floatKey(v float64) string {
	// 避免 1 vs 1.0 的字符串差异
	if v == math.Trunc(v) {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

func tryParseFloat(s string) (float64, bool) {
	f, err := strconv.ParseFloat(s, 64)
	if err == nil {
		return f, true
	}
	i, err := strconv.Atoi(s)
	if err == nil {
		return float64(i), true
	}
	return 0, false
}
