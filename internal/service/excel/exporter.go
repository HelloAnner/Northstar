package excel

import (
	"fmt"

	"github.com/xuri/excelize/v2"

	"northstar/internal/model"
)

// Exporter Excel导出器
type Exporter struct{}

// NewExporter 创建导出器
func NewExporter() *Exporter {
	return &Exporter{}
}

// Export 导出企业数据到Excel
func (e *Exporter) Export(companies []*model.Company, indicators *model.Indicators, includeChanges bool) (*excelize.File, error) {
	f := excelize.NewFile()

	// 创建企业数据表
	sheetName := "企业数据"
	f.SetSheetName("Sheet1", sheetName)

	// 设置表头
	headers := []string{
		"企业名称", "统一社会信用代码", "行业代码", "行业类型", "单位规模",
		"本期零售额", "上年同期零售额", "当月增速",
		"本年累计零售额", "上年累计零售额", "累计增速",
		"本期销售额", "上年同期销售额",
	}
	if includeChanges {
		headers = append(headers, "原始零售额", "调整幅度")
	}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, h)
	}

	// 设置表头样式
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E2E8F0"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetRowStyle(sheetName, 1, 1, headerStyle)

	// 写入数据
	for i, c := range companies {
		row := i + 2
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), c.Name)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), c.CreditCode)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), c.IndustryCode)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), string(c.IndustryType))
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), c.CompanyScale)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), c.RetailCurrentMonth)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), c.RetailLastYearMonth)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), fmt.Sprintf("%.2f%%", c.MonthGrowthRate()*100))
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), c.RetailCurrentCumulative)
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), c.RetailLastYearCumulative)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", row), fmt.Sprintf("%.2f%%", c.CumulativeGrowthRate()*100))
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", row), c.SalesCurrentMonth)
		f.SetCellValue(sheetName, fmt.Sprintf("M%d", row), c.SalesLastYearMonth)

		if includeChanges {
			f.SetCellValue(sheetName, fmt.Sprintf("N%d", row), c.OriginalRetailCurrentMonth)
			change := c.RetailCurrentMonth - c.OriginalRetailCurrentMonth
			f.SetCellValue(sheetName, fmt.Sprintf("O%d", row), change)
		}
	}

	// 创建指标汇总表
	if indicators != nil {
		indicatorSheet := "指标汇总"
		f.NewSheet(indicatorSheet)

		indicatorData := [][]interface{}{
			{"指标名称", "数值"},
			{"限上社零额(当月值)", indicators.LimitAboveMonthValue},
			{"限上社零额增速(当月)", fmt.Sprintf("%.2f%%", indicators.LimitAboveMonthRate*100)},
			{"限上社零额(累计值)", indicators.LimitAboveCumulativeValue},
			{"限上社零额增速(累计)", fmt.Sprintf("%.2f%%", indicators.LimitAboveCumulativeRate*100)},
			{"吃穿用增速(当月)", fmt.Sprintf("%.2f%%", indicators.EatWearUseMonthRate*100)},
			{"小微企业增速(当月)", fmt.Sprintf("%.2f%%", indicators.MicroSmallMonthRate*100)},
			{"社零总额(累计值)", indicators.TotalSocialCumulativeValue},
			{"社零总额增速(累计)", fmt.Sprintf("%.2f%%", indicators.TotalSocialCumulativeRate*100)},
		}

		// 四大行业增速
		for industryType, rate := range indicators.IndustryRates {
			indicatorData = append(indicatorData, []interface{}{
				fmt.Sprintf("%s销售额增速(当月)", industryType), fmt.Sprintf("%.2f%%", rate.MonthRate*100),
			})
			indicatorData = append(indicatorData, []interface{}{
				fmt.Sprintf("%s销售额增速(累计)", industryType), fmt.Sprintf("%.2f%%", rate.CumulativeRate*100),
			})
		}

		for i, row := range indicatorData {
			for j, val := range row {
				cell, _ := excelize.CoordinatesToCellName(j+1, i+1)
				f.SetCellValue(indicatorSheet, cell, val)
			}
		}

		// 设置表头样式
		f.SetRowStyle(indicatorSheet, 1, 1, headerStyle)
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 30)
	f.SetColWidth(sheetName, "B", "B", 25)
	f.SetColWidth(sheetName, "C", "M", 15)

	return f, nil
}
