package store

import (
	"fmt"

	"northstar/internal/model"
)

// BatchInsertACSnapshot 批量插入住餐快照数据
func (s *Store) BatchInsertACSnapshot(records []*model.ACSnapshot) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO ac_snapshot (
			snapshot_year, snapshot_month, snapshot_name,
			credit_code, name, industry_code, company_scale,
			revenue_current_month, revenue_current_cumulative,
			room_current_month, room_current_cumulative,
			food_current_month, food_current_cumulative,
			goods_current_month, goods_current_cumulative,
			source_sheet
		) VALUES (
			?, ?, ?,
			?, ?, ?, ?,
			?, ?,
			?, ?,
			?, ?,
			?, ?,
			?
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, r := range records {
		_, err := stmt.Exec(
			r.SnapshotYear, r.SnapshotMonth, r.SnapshotName,
			r.CreditCode, r.Name, r.IndustryCode, r.CompanyScale,
			r.RevenueCurrentMonth, r.RevenueCurrentCumulative,
			r.RoomCurrentMonth, r.RoomCurrentCumulative,
			r.FoodCurrentMonth, r.FoodCurrentCumulative,
			r.GoodsCurrentMonth, r.GoodsCurrentCumulative,
			r.SourceSheet,
		)
		if err != nil {
			return fmt.Errorf("failed to insert record: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// DeleteACSnapshotByYearMonth 删除指定年月的住餐快照数据
func (s *Store) DeleteACSnapshotByYearMonth(year, month int) error {
	_, err := s.db.Exec("DELETE FROM ac_snapshot WHERE snapshot_year = ? AND snapshot_month = ?", year, month)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	return nil
}

