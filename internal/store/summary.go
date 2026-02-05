/**
 * 汇总表写入
 *
 * @author Anner
 * Created on 2026/2/5
 */

package store

import (
	"fmt"

	"northstar/internal/model"
)

// BatchInsertSummaryLimitAbove 写入限上零售额汇总
func (s *Store) BatchInsertSummaryLimitAbove(rows []model.SummaryLimitAboveRetail) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
	}
	stmt, err := tx.Prepare(`
		INSERT INTO summary_limit_above_retail (
			data_year, data_month, row_key, row_no,
			value_current, value_last, rate,
			source_sheet, source_cell, import_log_id, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(data_year, data_month, row_key) DO UPDATE SET
			row_no = excluded.row_no,
			value_current = excluded.value_current,
			value_last = excluded.value_last,
			rate = excluded.rate,
			source_sheet = excluded.source_sheet,
			source_cell = excluded.source_cell,
			import_log_id = excluded.import_log_id,
			source_file = excluded.source_file
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare insert summary_limit_above_retail failed: %w", err)
	}
	defer stmt.Close()

	for _, r := range rows {
		if _, err := stmt.Exec(
			r.DataYear, r.DataMonth, r.RowKey, r.RowNo,
			r.ValueCurrent, r.ValueLast, r.Rate,
			r.SourceSheet, r.SourceCell, r.ImportLogID, r.SourceFile,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert summary_limit_above_retail failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit summary_limit_above_retail failed: %w", err)
	}
	return nil
}

// BatchInsertSummaryMicroSmall 写入小微汇总
func (s *Store) BatchInsertSummaryMicroSmall(rows []model.SummaryMicroSmall) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
	}
	stmt, err := tx.Prepare(`
		INSERT INTO summary_micro_small (
			data_year, data_month, row_key, row_no,
			value_current, value_last, rate,
			source_sheet, source_cell, import_log_id, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(data_year, data_month, row_key) DO UPDATE SET
			row_no = excluded.row_no,
			value_current = excluded.value_current,
			value_last = excluded.value_last,
			rate = excluded.rate,
			source_sheet = excluded.source_sheet,
			source_cell = excluded.source_cell,
			import_log_id = excluded.import_log_id,
			source_file = excluded.source_file
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare insert summary_micro_small failed: %w", err)
	}
	defer stmt.Close()

	for _, r := range rows {
		if _, err := stmt.Exec(
			r.DataYear, r.DataMonth, r.RowKey, r.RowNo,
			r.ValueCurrent, r.ValueLast, r.Rate,
			r.SourceSheet, r.SourceCell, r.ImportLogID, r.SourceFile,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert summary_micro_small failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit summary_micro_small failed: %w", err)
	}
	return nil
}

// BatchInsertSummaryEatWearUse 写入吃穿用汇总
func (s *Store) BatchInsertSummaryEatWearUse(rows []model.SummaryEatWearUse) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
	}
	stmt, err := tx.Prepare(`
		INSERT INTO summary_eat_wear_use (
			data_year, data_month, row_key, row_no,
			value_current, value_last, rate,
			source_sheet, source_cell, import_log_id, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(data_year, data_month, row_key) DO UPDATE SET
			row_no = excluded.row_no,
			value_current = excluded.value_current,
			value_last = excluded.value_last,
			rate = excluded.rate,
			source_sheet = excluded.source_sheet,
			source_cell = excluded.source_cell,
			import_log_id = excluded.import_log_id,
			source_file = excluded.source_file
	`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare insert summary_eat_wear_use failed: %w", err)
	}
	defer stmt.Close()

	for _, r := range rows {
		if _, err := stmt.Exec(
			r.DataYear, r.DataMonth, r.RowKey, r.RowNo,
			r.ValueCurrent, r.ValueLast, r.Rate,
			r.SourceSheet, r.SourceCell, r.ImportLogID, r.SourceFile,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("insert summary_eat_wear_use failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit summary_eat_wear_use failed: %w", err)
	}
	return nil
}
