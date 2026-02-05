/**
 * Sheet 原始结构与单元格写入
 *
 * @author Anner
 * Created on 2026/2/5
 */

package store

import (
	"fmt"

	"northstar/internal/model"
)

// InsertSheetColumns 写入 Sheet 列结构
func (s *Store) InsertSheetColumns(columns []model.SheetColumn) error {
	if len(columns) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
	}
	stmt, err := tx.Prepare(`
		INSERT INTO sheet_columns (
			sheet_name, col_idx, header_text, normalized_header, col_width,
			import_log_id, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare insert sheet_columns failed: %w", err)
	}
	defer stmt.Close()

	for _, col := range columns {
		if _, err := stmt.Exec(
			col.SheetName, col.ColIdx, col.HeaderText, col.NormalizedHeader, col.ColWidth,
			col.ImportLogID, col.SourceFile,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert sheet_columns failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sheet_columns failed: %w", err)
	}
	return nil
}

// InsertSheetRows 写入 Sheet 行结构
func (s *Store) InsertSheetRows(rows []model.SheetRow) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
	}
	stmt, err := tx.Prepare(`
		INSERT INTO sheet_rows (
			sheet_name, row_idx, row_height, import_log_id, source_file
		) VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare insert sheet_rows failed: %w", err)
	}
	defer stmt.Close()

	for _, row := range rows {
		if _, err := stmt.Exec(
			row.SheetName, row.RowIdx, row.RowHeight, row.ImportLogID, row.SourceFile,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert sheet_rows failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sheet_rows failed: %w", err)
	}
	return nil
}

// BatchInsertSheetCells 批量写入 Sheet 单元格
func (s *Store) BatchInsertSheetCells(cells []model.SheetCell) error {
	if len(cells) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
	}
	stmt, err := tx.Prepare(`
		INSERT INTO sheet_cells (
			sheet_name, row_idx, col_idx, a1,
			cell_type, raw_value, formula, calc_value,
			num_format, style_id, is_merged, merge_range,
			import_log_id, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare insert sheet_cells failed: %w", err)
	}
	defer stmt.Close()

	for _, cell := range cells {
		if _, err := stmt.Exec(
			cell.SheetName, cell.RowIdx, cell.ColIdx, cell.A1,
			cell.CellType, cell.RawValue, cell.Formula, cell.CalcValue,
			cell.NumFormat, cell.StyleID, cell.IsMerged, cell.MergeRange,
			cell.ImportLogID, cell.SourceFile,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert sheet_cells failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sheet_cells failed: %w", err)
	}
	return nil
}

// InsertSheetMerges 写入 Sheet 合并单元格
func (s *Store) InsertSheetMerges(merges []model.SheetMerge) error {
	if len(merges) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
	}
	stmt, err := tx.Prepare(`
		INSERT INTO sheet_merges (
			sheet_name, merge_range, start_row, start_col, end_row, end_col,
			import_log_id, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare insert sheet_merges failed: %w", err)
	}
	defer stmt.Close()

	for _, m := range merges {
		if _, err := stmt.Exec(
			m.SheetName, m.MergeRange, m.StartRow, m.StartCol, m.EndRow, m.EndCol,
			m.ImportLogID, m.SourceFile,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert sheet_merges failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sheet_merges failed: %w", err)
	}
	return nil
}
