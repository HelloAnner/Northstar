/**
 * 汇总类 Sheet 解析
 *
 * @author Anner
 * Created on 2026/2/5
 */

package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
	"northstar/internal/model"
)

type SummaryKind string

const (
	SummaryLimitAbove SummaryKind = "limit_above_retail"
	SummaryMicroSmall SummaryKind = "micro_small"
	SummaryEatWearUse SummaryKind = "eat_wear_use"
)

// RecognizeSummaryKind 识别汇总类 Sheet 类型
func RecognizeSummaryKind(sheetName string, file *excelize.File, maxCol int) SummaryKind {
	name := strings.TrimSpace(sheetName)
	switch {
	case strings.Contains(name, "限上零售额"):
		return SummaryLimitAbove
	case strings.Contains(name, "小微"):
		return SummaryMicroSmall
	case strings.Contains(name, "吃穿用"):
		return SummaryEatWearUse
	}

	// 兜底：通过首行字段位置判断
	colCurrent, colLast, _ := detectSummaryColumns(file, sheetName, maxCol)
	if colCurrent >= 10 {
		return SummaryMicroSmall
	}
	if colCurrent > 0 && colLast > 0 {
		return SummaryEatWearUse
	}
	return ""
}

// ParseSummarySheet 解析汇总类 Sheet
func ParseSummarySheet(file *excelize.File, sheetName string, year, month int) (
	limitAbove []model.SummaryLimitAboveRetail,
	micro []model.SummaryMicroSmall,
	eatWearUse []model.SummaryEatWearUse,
	err error,
) {
	dim, err := file.GetSheetDimension(sheetName)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get dimension failed: %w", err)
	}
	if strings.TrimSpace(dim) == "" {
		return nil, nil, nil, nil
	}
	maxRow, maxCol, err := parseDimension(dim)
	if err != nil || (maxRow == 1 && maxCol == 1) {
		rows, readErr := file.GetRows(sheetName)
		if readErr != nil {
			return nil, nil, nil, err
		}
		maxRow = len(rows)
		maxCol = 0
		for _, row := range rows {
			if len(row) > maxCol {
				maxCol = len(row)
			}
		}
	}

	kind := RecognizeSummaryKind(sheetName, file, maxCol)
	if kind == "" {
		return nil, nil, nil, nil
	}

	colCurrent, colLast, colRate := detectSummaryColumns(file, sheetName, maxCol)
	if colCurrent == 0 || colLast == 0 {
		return nil, nil, nil, nil
	}
	rowKeyCol := 1
	if kind == SummaryMicroSmall {
		rowKeyCol = maxInt(1, colCurrent-1)
	}

	for row := 2; row <= maxRow; row++ {
		rowKey := readCellRaw(file, sheetName, rowKeyCol, row)
		valCurrent := parseOptionalFloat(readCellCalc(file, sheetName, colCurrent, row))
		valLast := parseOptionalFloat(readCellCalc(file, sheetName, colLast, row))
		valRate := parseOptionalFloat(readCellCalc(file, sheetName, colRate, row))

		if strings.TrimSpace(rowKey) == "" && valCurrent == nil && valLast == nil && valRate == nil {
			continue
		}

		switch kind {
		case SummaryLimitAbove:
			limitAbove = append(limitAbove, model.SummaryLimitAboveRetail{
				DataYear:     year,
				DataMonth:    month,
				RowKey:       strings.TrimSpace(rowKey),
				RowNo:        row,
				ValueCurrent: valCurrent,
				ValueLast:    valLast,
				Rate:         valRate,
				SourceSheet:  sheetName,
				SourceCell:   fmt.Sprintf("B%d", row),
			})
		case SummaryMicroSmall:
			micro = append(micro, model.SummaryMicroSmall{
				DataYear:     year,
				DataMonth:    month,
				RowKey:       strings.TrimSpace(rowKey),
				RowNo:        row,
				ValueCurrent: valCurrent,
				ValueLast:    valLast,
				Rate:         valRate,
				SourceSheet:  sheetName,
				SourceCell:   fmt.Sprintf("%s%d", columnName(colCurrent), row),
			})
		case SummaryEatWearUse:
			eatWearUse = append(eatWearUse, model.SummaryEatWearUse{
				DataYear:     year,
				DataMonth:    month,
				RowKey:       strings.TrimSpace(rowKey),
				RowNo:        row,
				ValueCurrent: valCurrent,
				ValueLast:    valLast,
				Rate:         valRate,
				SourceSheet:  sheetName,
				SourceCell:   fmt.Sprintf("B%d", row),
			})
		}
	}

	return limitAbove, micro, eatWearUse, nil
}

func detectSummaryColumns(file *excelize.File, sheetName string, maxCol int) (current, last, rate int) {
	for col := 1; col <= maxCol; col++ {
		val := readCellRaw(file, sheetName, col, 1)
		val = strings.TrimSpace(val)
		switch val {
		case "45839":
			current = col
		case "45474":
			last = col
		case "45809":
			current = col
		case "45444":
			last = col
		case "增速":
			rate = col
		}
	}
	if rate == 0 {
		rate = last + 1
	}
	return current, last, rate
}

func readCellRaw(file *excelize.File, sheet string, col, row int) string {
	cell, _ := excelize.CoordinatesToCellName(col, row)
	v, _ := file.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true})
	return v
}

func readCellCalc(file *excelize.File, sheet string, col, row int) string {
	cell, _ := excelize.CoordinatesToCellName(col, row)
	v, _ := file.GetCellValue(sheet, cell)
	return v
}

func parseOptionalFloat(text string) *float64 {
	value := strings.TrimSpace(text)
	if value == "" {
		return nil
	}
	value = strings.ReplaceAll(value, ",", "")
	value = strings.ReplaceAll(value, "%", "")
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil
	}
	return &f
}

func parseDimension(dim string) (maxRow, maxCol int, err error) {
	parts := strings.Split(dim, ":")
	end := parts[0]
	if len(parts) == 2 {
		end = parts[1]
	}
	col, row, err := excelize.CellNameToCoordinates(end)
	if err != nil {
		return 0, 0, fmt.Errorf("parse dimension failed: %w", err)
	}
	return row, col, nil
}

func columnName(col int) string {
	name, _ := excelize.ColumnNumberToName(col)
	return name
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
