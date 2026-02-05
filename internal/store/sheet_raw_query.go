/**
 * Sheet 原始结构查询
 *
 * @author Anner
 * Created on 2026/2/5
 */

package store

import (
	"fmt"

	"northstar/internal/model"
)

// GetSheetColumnsByImportLog 查询 Sheet 列结构
func (s *Store) GetSheetColumnsByImportLog(importLogID int64, sheetName string) ([]model.SheetColumn, error) {
	rows, err := s.db.Query(`
		SELECT id, sheet_name, col_idx, header_text, normalized_header, col_width, import_log_id, source_file, created_at
		FROM sheet_columns
		WHERE import_log_id = ? AND sheet_name = ?
		ORDER BY col_idx ASC
	`, importLogID, sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to query sheet_columns: %w", err)
	}
	defer rows.Close()

	result := make([]model.SheetColumn, 0)
	for rows.Next() {
		var col model.SheetColumn
		if err := rows.Scan(
			&col.ID,
			&col.SheetName,
			&col.ColIdx,
			&col.HeaderText,
			&col.NormalizedHeader,
			&col.ColWidth,
			&col.ImportLogID,
			&col.SourceFile,
			&col.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sheet_columns: %w", err)
		}
		result = append(result, col)
	}
	return result, nil
}

// GetSheetRowsByImportLog 查询 Sheet 行结构
func (s *Store) GetSheetRowsByImportLog(importLogID int64, sheetName string) ([]model.SheetRow, error) {
	rows, err := s.db.Query(`
		SELECT id, sheet_name, row_idx, row_height, import_log_id, source_file, created_at
		FROM sheet_rows
		WHERE import_log_id = ? AND sheet_name = ?
		ORDER BY row_idx ASC
	`, importLogID, sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to query sheet_rows: %w", err)
	}
	defer rows.Close()

	result := make([]model.SheetRow, 0)
	for rows.Next() {
		var row model.SheetRow
		if err := rows.Scan(
			&row.ID,
			&row.SheetName,
			&row.RowIdx,
			&row.RowHeight,
			&row.ImportLogID,
			&row.SourceFile,
			&row.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sheet_rows: %w", err)
		}
		result = append(result, row)
	}
	return result, nil
}

// GetSheetCellsByImportLog 查询 Sheet 单元格
func (s *Store) GetSheetCellsByImportLog(importLogID int64, sheetName string, maxRow, maxCol int) ([]model.SheetCell, error) {
	query := `
		SELECT id, sheet_name, row_idx, col_idx, a1,
		       cell_type, raw_value, formula, calc_value,
		       num_format, style_id, is_merged, merge_range,
		       import_log_id, source_file, created_at
		FROM sheet_cells
		WHERE import_log_id = ? AND sheet_name = ?
	`
	args := []interface{}{importLogID, sheetName}
	if maxRow > 0 {
		query += " AND row_idx <= ?"
		args = append(args, maxRow)
	}
	if maxCol > 0 {
		query += " AND col_idx <= ?"
		args = append(args, maxCol)
	}
	query += " ORDER BY row_idx ASC, col_idx ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query sheet_cells: %w", err)
	}
	defer rows.Close()

	result := make([]model.SheetCell, 0)
	for rows.Next() {
		var cell model.SheetCell
		if err := rows.Scan(
			&cell.ID,
			&cell.SheetName,
			&cell.RowIdx,
			&cell.ColIdx,
			&cell.A1,
			&cell.CellType,
			&cell.RawValue,
			&cell.Formula,
			&cell.CalcValue,
			&cell.NumFormat,
			&cell.StyleID,
			&cell.IsMerged,
			&cell.MergeRange,
			&cell.ImportLogID,
			&cell.SourceFile,
			&cell.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sheet_cells: %w", err)
		}
		result = append(result, cell)
	}
	return result, nil
}

// GetSheetMergesByImportLog 查询 Sheet 合并单元格
func (s *Store) GetSheetMergesByImportLog(importLogID int64, sheetName string) ([]model.SheetMerge, error) {
	rows, err := s.db.Query(`
		SELECT id, sheet_name, merge_range, start_row, start_col, end_row, end_col,
		       import_log_id, source_file, created_at
		FROM sheet_merges
		WHERE import_log_id = ? AND sheet_name = ?
		ORDER BY id ASC
	`, importLogID, sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to query sheet_merges: %w", err)
	}
	defer rows.Close()

	result := make([]model.SheetMerge, 0)
	for rows.Next() {
		var m model.SheetMerge
		if err := rows.Scan(
			&m.ID,
			&m.SheetName,
			&m.MergeRange,
			&m.StartRow,
			&m.StartCol,
			&m.EndRow,
			&m.EndCol,
			&m.ImportLogID,
			&m.SourceFile,
			&m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sheet_merges: %w", err)
		}
		result = append(result, m)
	}
	return result, nil
}

// CountSheetCellsByImportLog 统计指定导入批次的单元格数量
func (s *Store) CountSheetCellsByImportLog(importLogID int64) (int, error) {
	row := s.db.QueryRow(`
		SELECT COUNT(*)
		FROM sheet_cells
		WHERE import_log_id = ?
	`, importLogID)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count sheet_cells: %w", err)
	}
	return count, nil
}
