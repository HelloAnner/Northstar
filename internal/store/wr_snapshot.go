package store

import (
	"fmt"

	"northstar/internal/model"
)

// BatchInsertWRSnapshot 批量插入批零快照数据
func (s *Store) BatchInsertWRSnapshot(records []*model.WRSnapshot) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO wr_snapshot (
			snapshot_year, snapshot_month, snapshot_name,
			credit_code, name, industry_code, company_scale,
			sales_current_month, sales_current_cumulative, sales_last_year_month, sales_last_year_cumulative,
			retail_current_month, retail_current_cumulative, retail_last_year_month, retail_last_year_cumulative,
			cat_grain_oil_food, cat_beverage, cat_tobacco_liquor, cat_clothing, cat_daily_use, cat_automobile,
			source_sheet
		) VALUES (
			?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?, ?, ?,
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
			r.SalesCurrentMonth, r.SalesCurrentCumulative, r.SalesLastYearMonth, r.SalesLastYearCumulative,
			r.RetailCurrentMonth, r.RetailCurrentCumulative, r.RetailLastYearMonth, r.RetailLastYearCumulative,
			r.CatGrainOilFood, r.CatBeverage, r.CatTobaccoLiquor, r.CatClothing, r.CatDailyUse, r.CatAutomobile,
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

// DeleteWRSnapshotByYearMonth 删除指定年月的批零快照数据
func (s *Store) DeleteWRSnapshotByYearMonth(year, month int) error {
	_, err := s.db.Exec("DELETE FROM wr_snapshot WHERE snapshot_year = ? AND snapshot_month = ?", year, month)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	return nil
}

