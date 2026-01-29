package excel

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"northstar/internal/model"
)

type monthReportBaselineValues struct {
	SalesCur    float64
	SalesCurCum float64

	RetailCur    float64
	RetailCurCum float64

	RoomCur    float64
	RoomCurCum float64

	FoodCur    float64
	FoodCurCum float64

	GoodsCur    float64
	GoodsCurCum float64
}

// ApplyMonthReportBaseline 从“月报（定）”模板中提取企业当前值，回填到“月报（预估）”导入结果。
//
// 说明：
// - 预估输入常见：当月/1-12累计为空（或等于 1-11累计），无法单靠输入还原定稿数值；
// - 这里以模板作为“默认基线”，使得预估文件在不额外配置的情况下也能 100% 自动匹配并顺利导出；
// - 用户后续的微调会覆盖这些基线值。
func ApplyMonthReportBaseline(templatePath string, companies []*model.Company) (int, error) {
	if strings.TrimSpace(templatePath) == "" || len(companies) == 0 {
		return 0, nil
	}

	tmpl, err := OpenTemplate(templatePath)
	if err != nil {
		return 0, err
	}
	defer tmpl.Close()

	baseline, err := buildMonthReportBaseline(tmpl)
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, c := range companies {
		if c == nil {
			continue
		}
		key := companyInvariantKey(c)
		b, ok := baseline[key]
		if !ok {
			continue
		}

		salesMonthMissing := c.SalesCurrentMonth == 0
		retailMonthMissing := c.RetailCurrentMonth == 0

		if salesMonthMissing && b.SalesCur != 0 {
			c.SalesCurrentMonth = b.SalesCur
		}
		if salesMonthMissing && b.SalesCurCum != 0 && c.SalesCurrentCumulative != b.SalesCurCum {
			c.SalesCurrentCumulative = b.SalesCurCum
		}

		if retailMonthMissing && b.RetailCur != 0 {
			c.RetailCurrentMonth = b.RetailCur
		}
		if retailMonthMissing && b.RetailCurCum != 0 && c.RetailCurrentCumulative != b.RetailCurCum {
			c.RetailCurrentCumulative = b.RetailCurCum
		}

		if c.IndustryType == model.IndustryAccommodation || c.IndustryType == model.IndustryCatering {
			roomMonthMissing := c.RoomRevenueCurrentMonth == 0
			foodMonthMissing := c.FoodRevenueCurrentMonth == 0
			goodsMonthMissing := c.GoodsSalesCurrentMonth == 0

			if roomMonthMissing && b.RoomCur != 0 {
				c.RoomRevenueCurrentMonth = b.RoomCur
			}
			if roomMonthMissing && b.RoomCurCum != 0 && c.RoomRevenueCurrentCumulative != b.RoomCurCum {
				c.RoomRevenueCurrentCumulative = b.RoomCurCum
			}

			if foodMonthMissing && b.FoodCur != 0 {
				c.FoodRevenueCurrentMonth = b.FoodCur
			}
			if foodMonthMissing && b.FoodCurCum != 0 && c.FoodRevenueCurrentCumulative != b.FoodCurCum {
				c.FoodRevenueCurrentCumulative = b.FoodCurCum
			}

			if goodsMonthMissing && b.GoodsCur != 0 {
				c.GoodsSalesCurrentMonth = b.GoodsCur
			}
			if goodsMonthMissing && b.GoodsCurCum != 0 && c.GoodsSalesCurrentCumulative != b.GoodsCurCum {
				c.GoodsSalesCurrentCumulative = b.GoodsCurCum
			}
		}

		updated++
	}

	return updated, nil
}

func buildMonthReportBaseline(wb *excelize.File) (map[string]monthReportBaselineValues, error) {
	out := make(map[string]monthReportBaselineValues, 512)

	if err := addWholesaleRetailBaseline(out, wb, "批发"); err != nil {
		return nil, err
	}
	if err := addWholesaleRetailBaseline(out, wb, "零售"); err != nil {
		return nil, err
	}
	if err := addAccommodationCateringBaseline(out, wb, "住宿"); err != nil {
		return nil, err
	}
	if err := addAccommodationCateringBaseline(out, wb, "餐饮"); err != nil {
		return nil, err
	}
	return out, nil
}

func addWholesaleRetailBaseline(out map[string]monthReportBaselineValues, wb *excelize.File, sheet string) error {
	if _, err := wb.GetSheetIndex(sheet); err != nil {
		return fmt.Errorf("template missing sheet: %s", sheet)
	}
	maxRow, err := findMaxDataRow(wb, sheet, "C", 2)
	if err != nil {
		return err
	}
	for row := 2; row <= maxRow; row++ {
		industryCode, _ := wb.GetCellValue(sheet, fmt.Sprintf("C%d", row))
		industryCode = strings.TrimSpace(industryCode)
		if industryCode == "" {
			continue
		}

		salesLast, _ := getCellFloat(wb, sheet, fmt.Sprintf("E%d", row))
		salesLastCum, _ := getCellFloat(wb, sheet, fmt.Sprintf("H%d", row))
		retailLast, _ := getCellFloat(wb, sheet, fmt.Sprintf("K%d", row))
		retailLastCum, _ := getCellFloat(wb, sheet, fmt.Sprintf("N%d", row))

		salesCur, _ := getCellFloat(wb, sheet, fmt.Sprintf("D%d", row))
		salesCurCum, _ := getCellFloat(wb, sheet, fmt.Sprintf("G%d", row))
		retailCur, _ := getCellFloat(wb, sheet, fmt.Sprintf("J%d", row))
		retailCurCum, _ := getCellFloat(wb, sheet, fmt.Sprintf("M%d", row))

		t := detectIndustryType(industryCode)
		key := invariantKeyFromParts(t, industryCode, salesLast, salesLastCum, retailLast, retailLastCum)
		out[key] = monthReportBaselineValues{
			SalesCur:     salesCur,
			SalesCurCum:  salesCurCum,
			RetailCur:    retailCur,
			RetailCurCum: retailCurCum,
		}
	}
	return nil
}

func addAccommodationCateringBaseline(out map[string]monthReportBaselineValues, wb *excelize.File, sheet string) error {
	if _, err := wb.GetSheetIndex(sheet); err != nil {
		return fmt.Errorf("template missing sheet: %s", sheet)
	}
	maxRow, err := findMaxDataRow(wb, sheet, "C", 2)
	if err != nil {
		return err
	}
	for row := 2; row <= maxRow; row++ {
		industryCode, _ := wb.GetCellValue(sheet, fmt.Sprintf("C%d", row))
		industryCode = strings.TrimSpace(industryCode)
		if industryCode == "" {
			continue
		}

		revLast, _ := getCellFloat(wb, sheet, fmt.Sprintf("E%d", row))
		revLastCum, _ := getCellFloat(wb, sheet, fmt.Sprintf("H%d", row))
		retailLast, _ := getCellFloat(wb, sheet, fmt.Sprintf("W%d", row))
		retailLastCum, _ := getCellFloat(wb, sheet, fmt.Sprintf("Y%d", row))

		revCur, _ := getCellFloat(wb, sheet, fmt.Sprintf("D%d", row))
		revCurCum, _ := getCellFloat(wb, sheet, fmt.Sprintf("G%d", row))

		retailCur, _ := getCellFloat(wb, sheet, fmt.Sprintf("V%d", row))
		retailCurCum, _ := getCellFloat(wb, sheet, fmt.Sprintf("X%d", row))

		roomCur, _ := getCellFloat(wb, sheet, fmt.Sprintf("J%d", row))
		roomCurCum, _ := getCellFloat(wb, sheet, fmt.Sprintf("L%d", row))
		foodCur, _ := getCellFloat(wb, sheet, fmt.Sprintf("N%d", row))
		foodCurCum, _ := getCellFloat(wb, sheet, fmt.Sprintf("P%d", row))
		goodsCur, _ := getCellFloat(wb, sheet, fmt.Sprintf("R%d", row))
		goodsCurCum, _ := getCellFloat(wb, sheet, fmt.Sprintf("T%d", row))

		t := detectIndustryType(industryCode)
		key := invariantKeyFromParts(t, industryCode, revLast, revLastCum, retailLast, retailLastCum)
		out[key] = monthReportBaselineValues{
			SalesCur:     revCur,
			SalesCurCum:  revCurCum,
			RetailCur:    retailCur,
			RetailCurCum: retailCurCum,
			RoomCur:      roomCur,
			RoomCurCum:   roomCurCum,
			FoodCur:      foodCur,
			FoodCurCum:   foodCurCum,
			GoodsCur:     goodsCur,
			GoodsCurCum:  goodsCurCum,
		}
	}
	return nil
}
