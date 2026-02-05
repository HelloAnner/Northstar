/**
 * Sheet 原始数据采集
 *
 * @author Anner
 * Created on 2026/2/5
 */

package importer

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
	"northstar/internal/model"
	"northstar/internal/parser"
)

type rawSheetData struct {
	columns []model.SheetColumn
	rows    []model.SheetRow
	cells   []model.SheetCell
	merges  []model.SheetMerge
}

func collectSheetRaw(file *excelize.File, sheetName string, importLogID *int64, sourceFile string) (*rawSheetData, error) {
	dim, err := file.GetSheetDimension(sheetName)
	if err != nil {
		return nil, fmt.Errorf("get sheet dimension failed: %w", err)
	}
	if strings.TrimSpace(dim) == "" {
		return &rawSheetData{}, nil
	}

	maxRow, maxCol, err := parseDimension(dim)
	if err != nil {
		return nil, err
	}

	columns := make([]model.SheetColumn, 0, maxCol)
	for colIdx := 1; colIdx <= maxCol; colIdx++ {
		colName, _ := excelize.ColumnNumberToName(colIdx)
		cell, _ := excelize.CoordinatesToCellName(colIdx, 1)
		header, _ := file.GetCellValue(sheetName, cell, excelize.Options{RawCellValue: true})
		normalized := parser.NormalizeColumnName(header)
		width, _ := file.GetColWidth(sheetName, colName)
		columns = append(columns, model.SheetColumn{
			SheetName:        sheetName,
			ColIdx:           colIdx,
			HeaderText:       header,
			NormalizedHeader: normalized,
			ColWidth:         width,
			ImportLogID:      importLogID,
			SourceFile:       sourceFile,
		})
	}

	rows := make([]model.SheetRow, 0, maxRow)
	for rowIdx := 1; rowIdx <= maxRow; rowIdx++ {
		height, _ := file.GetRowHeight(sheetName, rowIdx)
		rows = append(rows, model.SheetRow{
			SheetName:  sheetName,
			RowIdx:     rowIdx,
			RowHeight:  height,
			ImportLogID: importLogID,
			SourceFile: sourceFile,
		})
	}

	merges, mergeMap, err := collectSheetMerges(file, sheetName, importLogID, sourceFile)
	if err != nil {
		return nil, err
	}

	cells := make([]model.SheetCell, 0, maxRow*maxCol)
	for rowIdx := 1; rowIdx <= maxRow; rowIdx++ {
		for colIdx := 1; colIdx <= maxCol; colIdx++ {
			cellName, _ := excelize.CoordinatesToCellName(colIdx, rowIdx)
			rawValue, _ := file.GetCellValue(sheetName, cellName, excelize.Options{RawCellValue: true})
			calcValue, _ := file.GetCellValue(sheetName, cellName)
			formula, _ := file.GetCellFormula(sheetName, cellName)
			cellType, _ := file.GetCellType(sheetName, cellName)
			styleID, _ := file.GetCellStyle(sheetName, cellName)
			numFormat := extractNumFormat(file, styleID)

			mergeRange := mergeMap[cellKey(rowIdx, colIdx)]
			isMerged := 0
			if mergeRange != "" {
				isMerged = 1
			}

			if rawValue == "" && calcValue == "" && formula == "" && styleID == 0 && isMerged == 0 {
				continue
			}

			cells = append(cells, model.SheetCell{
				SheetName:  sheetName,
				RowIdx:     rowIdx,
				ColIdx:     colIdx,
				A1:         cellName,
				CellType:   string(cellType),
				RawValue:   rawValue,
				Formula:    formula,
				CalcValue:  calcValue,
				NumFormat:  numFormat,
				StyleID:    styleID,
				IsMerged:   isMerged,
				MergeRange: mergeRange,
				ImportLogID: importLogID,
				SourceFile: sourceFile,
			})
		}
	}

	return &rawSheetData{
		columns: columns,
		rows:    rows,
		cells:   cells,
		merges:  merges,
	}, nil
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

func collectSheetMerges(file *excelize.File, sheetName string, importLogID *int64, sourceFile string) ([]model.SheetMerge, map[string]string, error) {
	mergeCells, err := file.GetMergeCells(sheetName)
	if err != nil {
		return nil, nil, fmt.Errorf("get merge cells failed: %w", err)
	}
	merges := make([]model.SheetMerge, 0, len(mergeCells))
	mergeMap := make(map[string]string)
	for _, m := range mergeCells {
		rangeText := m[0]
		start := m.GetStartAxis()
		end := m.GetEndAxis()
		startCol, startRow, err := excelize.CellNameToCoordinates(start)
		if err != nil {
			continue
		}
		endCol, endRow, err := excelize.CellNameToCoordinates(end)
		if err != nil {
			continue
		}
		merges = append(merges, model.SheetMerge{
			SheetName:  sheetName,
			MergeRange: rangeText,
			StartRow:   startRow,
			StartCol:   startCol,
			EndRow:     endRow,
			EndCol:     endCol,
			ImportLogID: importLogID,
			SourceFile: sourceFile,
		})
		for r := startRow; r <= endRow; r++ {
			for c := startCol; c <= endCol; c++ {
				mergeMap[cellKey(r, c)] = rangeText
			}
		}
	}
	return merges, mergeMap, nil
}

func cellKey(row, col int) string {
	return fmt.Sprintf("%d:%d", row, col)
}

func extractNumFormat(file *excelize.File, styleID int) string {
	if styleID <= 0 {
		return ""
	}
	style, err := file.GetStyle(styleID)
	if err != nil || style == nil {
		return ""
	}
	if style.CustomNumFmt != nil {
		return *style.CustomNumFmt
	}
	if style.NumFmt != 0 {
		return fmt.Sprintf("%d", style.NumFmt)
	}
	return ""
}
