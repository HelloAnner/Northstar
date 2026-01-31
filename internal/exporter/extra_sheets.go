package exporter

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"northstar/internal/calculator"
	"northstar/internal/model"
	"northstar/internal/store"
)

type indicatorIndex map[string]calculator.Indicator

func calculateIndicatorIndex(st *store.Store, year, month int) (indicatorIndex, error) {
	calc := calculator.NewCalculator(st)
	groups, err := calc.CalculateAll(year, month)
	if err != nil {
		return nil, fmt.Errorf("计算指标失败: %w", err)
	}

	m := indicatorIndex{}
	for _, g := range groups {
		for _, it := range g.Indicators {
			it.Value = math.Round(it.Value)
			m[it.ID] = it
		}
	}
	return m, nil
}

func fillEatWearUseSheetByRowOrder(f *excelize.File, sheet string, records []*model.WholesaleRetail) error {
	maxCol, maxRow, err := getSheetMaxColRow(f, sheet)
	if err != nil {
		return fmt.Errorf("读取 %s 维度失败: %w", sheet, err)
	}
	if err := clearSheetArea(f, sheet, 2, maxRow, 1, maxCol); err != nil {
		return fmt.Errorf("清空 %s 失败: %w", sheet, err)
	}

	list := make([]*model.WholesaleRetail, 0, len(records))
	list = append(list, records...)
	sort.Slice(list, func(i, j int) bool {
		ai := strings.TrimSpace(list[i].IndustryType)
		aj := strings.TrimSpace(list[j].IndustryType)
		if ai != aj {
			return ai < aj
		}
		if list[i].RowNo != list[j].RowNo {
			return list[i].RowNo < list[j].RowNo
		}
		return list[i].ID < list[j].ID
	})

	capacity := maxRow - 1
	if len(list) > capacity {
		return fmt.Errorf("%s 容量不足（rows=%d, records=%d）", sheet, capacity, len(list))
	}

	for i, r := range list {
		row := 2 + i
		if err := setCellValue(f, sheet, fmt.Sprintf("B%d", row), strings.TrimSpace(r.CreditCode)); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("C%d", row), strings.TrimSpace(r.Name)); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("D%d", row), normalizeCodeText(r.IndustryCode)); err != nil {
			return err
		}

		salesCur := math.Round(r.SalesCurrentMonth)
		salesLast := math.Round(r.SalesLastYearMonth)
		salesCurCum := math.Round(r.SalesCurrentCumulative)
		salesLastCum := math.Round(r.SalesLastYearCumulative)
		retailCur := math.Round(r.RetailCurrentMonth)
		retailLast := math.Round(r.RetailLastYearMonth)
		retailCurCum := math.Round(r.RetailCurrentCumulative)
		retailLastCum := math.Round(r.RetailLastYearCumulative)

		if err := setCellValue(f, sheet, fmt.Sprintf("E%d", row), salesCur); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("F%d", row), salesLast); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("G%d", row), ratePercent(salesCur, salesLast)); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("H%d", row), salesCurCum); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("I%d", row), salesLastCum); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("J%d", row), ratePercent(salesCurCum, salesLastCum)); err != nil {
			return err
		}

		if err := setCellValue(f, sheet, fmt.Sprintf("K%d", row), retailCur); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("L%d", row), retailLast); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("M%d", row), ratePercent(retailCur, retailLast)); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("N%d", row), retailCurCum); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("O%d", row), retailLastCum); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("P%d", row), ratePercent(retailCurCum, retailLastCum)); err != nil {
			return err
		}

		ewuCur := 0.0
		ewuLast := 0.0
		if r.IsEatWearUse == 1 {
			ewuCur = retailCur
			ewuLast = retailLast
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("Q%d", row), ewuCur); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("R%d", row), ewuLast); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("S%d", row), ratePercent(ewuCur, ewuLast)); err != nil {
			return err
		}

		if err := setCellValue(f, sheet, fmt.Sprintf("T%d", row), r.CompanyScale); err != nil {
			return err
		}

		microCur := 0.0
		microLast := 0.0
		if r.IsSmallMicro == 1 {
			microCur = retailCur
			microLast = retailLast
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("U%d", row), microCur); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("V%d", row), microLast); err != nil {
			return err
		}

		if err := setCellValue(f, sheet, fmt.Sprintf("W%d", row), math.Round(r.NetworkSales)); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("X%d", row), 0); err != nil {
			return err
		}
		if r.OpeningYear != nil {
			if err := setCellValue(f, sheet, fmt.Sprintf("Y%d", row), *r.OpeningYear); err != nil {
				return err
			}
		}
		if r.OpeningMonth != nil {
			if err := setCellValue(f, sheet, fmt.Sprintf("Z%d", row), *r.OpeningMonth); err != nil {
				return err
			}
		}
	}

	return nil
}

func fillMicroSmallSheetByRowOrder(f *excelize.File, sheet string, records []*model.WholesaleRetail) error {
	maxCol, maxRow, err := getSheetMaxColRow(f, sheet)
	if err != nil {
		return fmt.Errorf("读取 %s 维度失败: %w", sheet, err)
	}
	if err := clearSheetArea(f, sheet, 2, maxRow, 1, maxCol); err != nil {
		return fmt.Errorf("清空 %s 失败: %w", sheet, err)
	}

	var list []*model.WholesaleRetail
	for _, r := range records {
		if r.IsSmallMicro == 1 {
			list = append(list, r)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		ai := strings.TrimSpace(list[i].IndustryType)
		aj := strings.TrimSpace(list[j].IndustryType)
		if ai != aj {
			return ai < aj
		}
		if list[i].RowNo != list[j].RowNo {
			return list[i].RowNo < list[j].RowNo
		}
		return list[i].ID < list[j].ID
	})

	capacity := maxRow - 1
	if len(list) > capacity {
		return fmt.Errorf("%s 容量不足（rows=%d, records=%d）", sheet, capacity, len(list))
	}

	for i, r := range list {
		row := 2 + i
		if err := setCellValue(f, sheet, fmt.Sprintf("B%d", row), strings.TrimSpace(r.CreditCode)); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("C%d", row), strings.TrimSpace(r.Name)); err != nil {
			return err
		}
		cur := math.Round(r.RetailCurrentMonth)
		last := math.Round(r.RetailLastYearMonth)
		if err := setCellValue(f, sheet, fmt.Sprintf("D%d", row), cur); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("E%d", row), last); err != nil {
			return err
		}
		if err := setCellValue(f, sheet, fmt.Sprintf("F%d", row), ratePercent(cur, last)); err != nil {
			return err
		}
	}

	return nil
}

func fillEatWearUseExcludedSheetByRowOrder(f *excelize.File, sheet string, records []*model.WholesaleRetail) error {
	maxCol, maxRow, err := getSheetMaxColRow(f, sheet)
	if err != nil {
		return fmt.Errorf("读取 %s 维度失败: %w", sheet, err)
	}
	return clearSheetArea(f, sheet, 2, maxRow, 1, maxCol)
}

func fillSocialRetailSheetAndMaterialize(
	f *excelize.File,
	st *store.Store,
	year int,
	month int,
	indicators indicatorIndex,
) error {
	sheet := "社零额（定）"

	prevYear, prevMonth := prevYearMonth(year, month)
	prevIndicators, err := calculateIndicatorIndex(st, prevYear, prevMonth)
	if err != nil {
		prevIndicators = indicatorIndex{}
	}

	getFloat := func(key string) float64 {
		v, err := st.GetConfigFloat(key)
		if err != nil {
			return 0
		}
		return v
	}

	if err := setCellValueAndClearFormula(f, sheet, "J2", month); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "B4", indicators["microSmall_month_rate"].Value); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "C4", indicators["eatWearUse_month_rate"].Value); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "D4", getFloat("sample_rate_month")); err != nil {
		return err
	}

	if err := setCellValueAndClearFormula(f, sheet, "B6", prevIndicators["microSmall_month_rate"].Value); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "C6", prevIndicators["eatWearUse_month_rate"].Value); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "D6", getFloat("sample_rate_prev")); err != nil {
		return err
	}

	if err := setCellValueAndClearFormula(f, sheet, "B12", getFloat("weight_small_micro")); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "C12", getFloat("weight_eat_wear_use")); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "D12", getFloat("weight_sample")); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "I3", getFloat("province_limit_below_rate_change")); err != nil {
		return err
	}

	if err := setCellValueAndClearFormula(f, sheet, "E18", getFloat("history_social_e18")); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "E19", getFloat("history_social_e19")); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "E20", getFloat("history_social_e20")); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "E21", getFloat("history_social_e21")); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "E22", getFloat("history_social_e22")); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, sheet, "E23", getFloat("history_social_e23")); err != nil {
		return err
	}

	if err := materializeFormulasInSheet(f, sheet); err != nil {
		return fmt.Errorf("社零额（定）公式计算失败: %w", err)
	}
	return nil
}

func rewriteFixedSummarySheet(
	f *excelize.File,
	year int,
	month int,
	wh wrSums,
	re wrSums,
	acc wrSums,
	cat wrSums,
	indicators indicatorIndex,
	wrRecords []*model.WholesaleRetail,
	acRecords []*model.AccommodationCatering,
) error {
	summary := "汇总表（定）"

	totalCompanies := len(wrRecords) + len(acRecords)
	reportedCompanies := totalCompanies
	negativeGrowthCount := 0
	for _, r := range wrRecords {
		if r.SalesLastYearMonth > 0 && r.SalesCurrentMonth < r.SalesLastYearMonth {
			negativeGrowthCount++
		}
	}
	for _, r := range acRecords {
		if r.RevenueLastYearMonth > 0 && r.RevenueCurrentMonth < r.RevenueLastYearMonth {
			negativeGrowthCount++
		}
	}

	reportRate := 0.0
	if totalCompanies > 0 {
		reportRate = math.Round(float64(reportedCompanies) / float64(totalCompanies) * 100.0)
	}
	negativeRatio := 0.0
	if reportedCompanies > 0 {
		negativeRatio = math.Round(float64(negativeGrowthCount) / float64(reportedCompanies) * 100.0)
	}

	overallRetailCur := wh.retailCur + re.retailCur + acc.retailCur + cat.retailCur
	overallRetailLast := wh.retailLast + re.retailLast + acc.retailLast + cat.retailLast
	overallRetailCurCum := wh.retailCurCum + re.retailCurCum + acc.retailCurCum + cat.retailCurCum
	overallRetailLastCum := wh.retailLastCum + re.retailLastCum + acc.retailLastCum + cat.retailLastCum

	// 汇总表（定）口径为“万元”，行业表/总表口径为“千元”
	limitAboveMonthWan := math.Round(overallRetailCur / 10.0)
	limitAboveLastYearMonthWan := math.Round(overallRetailLast / 10.0)
	limitAboveCumulativeWan := math.Round(overallRetailCurCum / 10.0)
	limitAboveLastYearCumulativeWan := math.Round(overallRetailLastCum / 10.0)

	if err := setCellValueAndClearFormula(f, summary, "B4", totalCompanies); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "C4", reportedCompanies); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "D4", reportRate); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "E4", negativeGrowthCount); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "F4", negativeRatio); err != nil {
		return err
	}

	if err := setCellValueAndClearFormula(f, summary, "G4", limitAboveMonthWan); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "H4", limitAboveLastYearMonthWan); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "I4", limitAboveCumulativeWan); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "J4", limitAboveLastYearCumulativeWan); err != nil {
		return err
	}

	if err := setCellValueAndClearFormula(f, summary, "K4", ratePercent(wh.salesCur, wh.salesLast)); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "L4", ratePercent(wh.salesCurCum, wh.salesLastCum)); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "M4", ratePercent(re.salesCur, re.salesLast)); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "N4", ratePercent(re.salesCurCum, re.salesLastCum)); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "O4", ratePercent(acc.salesCur, acc.salesLast)); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "P4", ratePercent(acc.salesCurCum, acc.salesLastCum)); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "Q4", ratePercent(cat.salesCur, cat.salesLast)); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "R4", ratePercent(cat.salesCurCum, cat.salesLastCum)); err != nil {
		return err
	}

	if err := setCellValueAndClearFormula(f, summary, "S4", ratePercent(overallRetailCur, overallRetailLast)); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "T4", ratePercent(overallRetailCurCum, overallRetailLastCum)); err != nil {
		return err
	}

	if err := setCellValueAndClearFormula(f, summary, "U4", indicators["eatWearUse_month_rate"].Value); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "V4", indicators["microSmall_month_rate"].Value); err != nil {
		return err
	}

	statusText := "已全部上报"
	if totalCompanies != reportedCompanies {
		statusText = fmt.Sprintf("已上报%d家，上报进度%d%%", reportedCompanies, int(reportRate))
	}
	if err := setCellValueAndClearFormula(f, summary, "D10", statusText); err != nil {
		return err
	}

	totalSocialYi := roundHalfUp(indicators["totalSocial_cumulative_value"].Value/10000.0, 2)
	totalSocialRate := indicators["totalSocial_cumulative_rate"].Value
	if err := setCellValueAndClearFormula(f, summary, "N10", totalSocialYi); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "S10", totalSocialRate); err != nil {
		return err
	}

	monthRetailYi := formatTrimFloat(limitAboveMonthWan/10000.0, 2)
	cumRetailYi := formatTrimFloat(limitAboveCumulativeWan/10000.0, 2)
	totalSocialYiText := formatTrimFloat(totalSocialYi, 2)

	period := fmt.Sprintf("1-%d月", month)
	summaryText := fmt.Sprintf(
		"社会消费品零售总额：全县%d家限上商贸单位%s。%d月，批发、零售、住宿、餐饮业销售额(营业额)同比分别增长%d%%、%d%%、%d%%、%d%%。当月上报零售额%s亿元，同比增长%d%%；累计上报零售额%s亿元，同比增长%d%%。%s，全社会消费品零售总额预计完成%s亿元，同比增长%d%%。",
		totalCompanies,
		statusText,
		month,
		int(ratePercent(wh.salesCur, wh.salesLast)),
		int(ratePercent(re.salesCur, re.salesLast)),
		int(ratePercent(acc.salesCur, acc.salesLast)),
		int(ratePercent(cat.salesCur, cat.salesLast)),
		monthRetailYi,
		int(ratePercent(overallRetailCur, overallRetailLast)),
		cumRetailYi,
		int(ratePercent(overallRetailCurCum, overallRetailLastCum)),
		period,
		totalSocialYiText,
		int(totalSocialRate),
	)

	if err := setCellValueAndClearFormula(f, summary, "A11", summaryText); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "X3", summaryText); err != nil {
		return err
	}
	if err := setCellValueAndClearFormula(f, summary, "W4", fmt.Sprintf("%d-%02d", year, month)); err != nil {
		return err
	}
	return nil
}

func prevYearMonth(year, month int) (int, int) {
	if month <= 1 {
		return year - 1, 12
	}
	return year, month - 1
}

func getSheetMaxColRow(f *excelize.File, sheet string) (int, int, error) {
	dim, err := f.GetSheetDimension(sheet)
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Split(dim, ":")
	maxCell := parts[len(parts)-1]
	maxCol, maxRow, err := excelize.CellNameToCoordinates(maxCell)
	if err != nil {
		return 0, 0, err
	}
	return maxCol, maxRow, nil
}

func clearSheetArea(f *excelize.File, sheet string, fromRow, toRow, fromCol, toCol int) error {
	if fromRow > toRow || fromCol > toCol {
		return nil
	}
	for r := fromRow; r <= toRow; r++ {
		for c := fromCol; c <= toCol; c++ {
			cell, err := excelize.CoordinatesToCellName(c, r)
			if err != nil {
				return err
			}
			if err := f.SetCellValue(sheet, cell, ""); err != nil {
				return err
			}
			_ = f.SetCellFormula(sheet, cell, "")
		}
	}
	return nil
}

func setCellValueAndClearFormula(f *excelize.File, sheet, cell string, value interface{}) error {
	if err := f.SetCellValue(sheet, cell, value); err != nil {
		return err
	}
	_ = f.SetCellFormula(sheet, cell, "")
	return nil
}

func materializeFormulasInSheet(f *excelize.File, sheet string) error {
	maxCol, maxRow, err := getSheetMaxColRow(f, sheet)
	if err != nil {
		return err
	}
	for r := 1; r <= maxRow; r++ {
		for c := 1; c <= maxCol; c++ {
			cell, err := excelize.CoordinatesToCellName(c, r)
			if err != nil {
				return err
			}
			formula, err := f.GetCellFormula(sheet, cell)
			if err != nil {
				return err
			}
			if strings.TrimSpace(formula) == "" {
				continue
			}

			val, err := f.CalcCellValue(sheet, cell)
			if err != nil {
				// 公式计算可能因缺少“社零额(定)”输入导致出现 #DIV/0! 等错误。
				// 为保证导出稳定且“结果由代码写入”，这里将错误结果落为 0 并清除公式。
				if err := f.SetCellValue(sheet, cell, 0); err != nil {
					return err
				}
				if err := f.SetCellFormula(sheet, cell, ""); err != nil {
					return err
				}
				continue
			}
			s := strings.TrimSpace(val)
			if n, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", ""), 64); err == nil {
				if err := f.SetCellValue(sheet, cell, n); err != nil {
					return err
				}
			} else {
				if err := f.SetCellValue(sheet, cell, s); err != nil {
					return err
				}
			}
			if err := f.SetCellFormula(sheet, cell, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

func formatTrimFloat(v float64, digits int) string {
	if digits < 0 {
		return fmt.Sprintf("%v", v)
	}
	s := strconv.FormatFloat(roundHalfUp(v, digits), 'f', digits, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}
