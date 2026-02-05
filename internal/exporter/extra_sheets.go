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

	// 汇总表优先：若存在汇总增速，覆盖默认指标值
	if rate, err := st.GetSummaryMicroSmallRate(year, month); err == nil && rate != nil {
		if it, ok := m["microSmall_month_rate"]; ok {
			it.Value = math.Round(*rate)
			m["microSmall_month_rate"] = it
		}
	}
	if rate, err := st.GetSummaryEatWearUseRate(year, month); err == nil && rate != nil {
		if it, ok := m["eatWearUse_month_rate"]; ok {
			it.Value = math.Round(*rate)
			m["eatWearUse_month_rate"] = it
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
		if err := applyRowLogics(f, sheet, row, r, eatWearUseRowLogics); err != nil {
			return err
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
		if err := applyRowLogics(f, sheet, row, r, microSmallRowLogics); err != nil {
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
	ctx := newSocialContext(st, year, month, indicators)
	return applyCellLogics(f, ctx, socialSheetLogics())
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
	ctx := newSummaryContext(year, month, wh, re, acc, cat, indicators, wrRecords, acRecords)
	return applyCellLogics(f, ctx, summarySheetLogics())
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

func setCellValueIfNoFormula(f *excelize.File, sheet, cell string, value interface{}) error {
	formula, err := f.GetCellFormula(sheet, cell)
	if err != nil {
		return err
	}
	if strings.TrimSpace(formula) != "" {
		// 目标 sheet 强约束：保留模板公式（社零额（定）/汇总表（定）依赖模板公式）。
		return nil
	}
	return f.SetCellValue(sheet, cell, value)
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
