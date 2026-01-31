package store

import (
	"database/sql"
	"fmt"
	"strings"

	"northstar/internal/model"
)

// BatchInsertWR 批量插入批零企业数据
func (s *Store) BatchInsertWR(records []*model.WholesaleRetail) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			sales_prev_month, sales_current_month, sales_last_year_month, sales_month_rate,
			sales_prev_cumulative, sales_last_year_prev_cumulative,
			sales_current_cumulative, sales_last_year_cumulative, sales_cumulative_rate,
			retail_prev_month, retail_current_month, retail_last_year_month, retail_month_rate,
			retail_prev_cumulative, retail_last_year_prev_cumulative,
			retail_current_cumulative, retail_last_year_cumulative, retail_cumulative_rate,
			retail_ratio,
			cat_grain_oil_food, cat_beverage, cat_tobacco_liquor,
			cat_clothing, cat_daily_use, cat_automobile,
			is_small_micro, is_eat_wear_use,
			first_report_ip, fill_ip, network_sales, opening_year, opening_month,
			original_sales_current_month, original_retail_current_month,
			source_sheet, source_file
		) VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?,
			?, ?, ?, ?,
			?, ?,
			?, ?, ?,
			?, ?, ?, ?,
			?, ?,
			?, ?, ?,
			?,
			?, ?, ?,
			?, ?, ?,
			?, ?,
			?, ?, ?, ?, ?,
			?, ?,
			?, ?
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, r := range records {
		_, err := stmt.Exec(
			r.CreditCode, r.Name, r.IndustryCode, r.IndustryType, r.CompanyScale, r.RowNo,
			r.DataYear, r.DataMonth,
			r.SalesPrevMonth, r.SalesCurrentMonth, r.SalesLastYearMonth, r.SalesMonthRate,
			r.SalesPrevCumulative, r.SalesLastYearPrevCumulative,
			r.SalesCurrentCumulative, r.SalesLastYearCumulative, r.SalesCumulativeRate,
			r.RetailPrevMonth, r.RetailCurrentMonth, r.RetailLastYearMonth, r.RetailMonthRate,
			r.RetailPrevCumulative, r.RetailLastYearPrevCumulative,
			r.RetailCurrentCumulative, r.RetailLastYearCumulative, r.RetailCumulativeRate,
			r.RetailRatio,
			r.CatGrainOilFood, r.CatBeverage, r.CatTobaccoLiquor,
			r.CatClothing, r.CatDailyUse, r.CatAutomobile,
			r.IsSmallMicro, r.IsEatWearUse,
			r.FirstReportIP, r.FillIP, r.NetworkSales, r.OpeningYear, r.OpeningMonth,
			r.OriginalSalesCurrentMonth, r.OriginalRetailCurrentMonth,
			r.SourceSheet, r.SourceFile,
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

// WRQueryOptions 批零企业查询选项
type WRQueryOptions struct {
	DataYear     *int
	DataMonth    *int
	IndustryType *string // wholesale/retail
	CompanyScale *int
	IsSmallMicro *int
	IsEatWearUse *int
	Limit        int
	Offset       int
}

// GetWRByYearMonth 获取指定年月的批零企业数据
func (s *Store) GetWRByYearMonth(opts WRQueryOptions) ([]*model.WholesaleRetail, error) {
	query := "SELECT * FROM wholesale_retail WHERE 1=1"
	args := []interface{}{}

	if opts.DataYear != nil {
		query += " AND data_year = ?"
		args = append(args, *opts.DataYear)
	}
	if opts.DataMonth != nil {
		query += " AND data_month = ?"
		args = append(args, *opts.DataMonth)
	}
	if opts.IndustryType != nil {
		query += " AND industry_type = ?"
		args = append(args, *opts.IndustryType)
	}
	if opts.CompanyScale != nil {
		query += " AND company_scale = ?"
		args = append(args, *opts.CompanyScale)
	}
	if opts.IsSmallMicro != nil {
		query += " AND is_small_micro = ?"
		args = append(args, *opts.IsSmallMicro)
	}
	if opts.IsEatWearUse != nil {
		query += " AND is_eat_wear_use = ?"
		args = append(args, *opts.IsEatWearUse)
	}

	query += " ORDER BY id"

	if opts.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, opts.Limit)
		if opts.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, opts.Offset)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %w", err)
	}
	defer rows.Close()

	return s.scanWRRows(rows)
}

// UpdateWR 更新批零企业数据
func (s *Store) UpdateWR(id int64, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setClauses := []string{}
	args := []interface{}{}

	for field, value := range updates {
		setClauses = append(setClauses, field+" = ?")
		args = append(args, value)
	}
	args = append(args, id)

	query := fmt.Sprintf("UPDATE wholesale_retail SET %s WHERE id = ?",
		strings.Join(setClauses, ", "))

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	return nil
}

// DeleteWRByYearMonth 删除指定年月的批零企业数据
func (s *Store) DeleteWRByYearMonth(year, month int) error {
	_, err := s.db.Exec("DELETE FROM wholesale_retail WHERE data_year = ? AND data_month = ?",
		year, month)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	return nil
}

// CountWR 统计批零企业数量
func (s *Store) CountWR(opts WRQueryOptions) (int, error) {
	query := "SELECT COUNT(*) FROM wholesale_retail WHERE 1=1"
	args := []interface{}{}

	if opts.DataYear != nil {
		query += " AND data_year = ?"
		args = append(args, *opts.DataYear)
	}
	if opts.DataMonth != nil {
		query += " AND data_month = ?"
		args = append(args, *opts.DataMonth)
	}
	if opts.IndustryType != nil {
		query += " AND industry_type = ?"
		args = append(args, *opts.IndustryType)
	}
	if opts.IsSmallMicro != nil {
		query += " AND is_small_micro = ?"
		args = append(args, *opts.IsSmallMicro)
	}
	if opts.IsEatWearUse != nil {
		query += " AND is_eat_wear_use = ?"
		args = append(args, *opts.IsEatWearUse)
	}

	var count int
	err := s.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count: %w", err)
	}

	return count, nil
}

// GetWRByID 根据 ID 获取批零企业
func (s *Store) GetWRByID(id int64) (*model.WholesaleRetail, error) {
	row := s.db.QueryRow("SELECT * FROM wholesale_retail WHERE id = ?", id)
	return s.scanWRRow(row)
}

// scanWRRows 扫描多行批零企业数据
func (s *Store) scanWRRows(rows *sql.Rows) ([]*model.WholesaleRetail, error) {
	var results []*model.WholesaleRetail

	for rows.Next() {
		r := &model.WholesaleRetail{}
		var firstReportIP sql.NullString
		var fillIP sql.NullString
		err := rows.Scan(
			&r.ID, &r.CreditCode, &r.Name, &r.IndustryCode, &r.IndustryType,
			&r.CompanyScale, &r.RowNo,
			&r.DataYear, &r.DataMonth,
			&r.SalesPrevMonth, &r.SalesCurrentMonth, &r.SalesLastYearMonth, &r.SalesMonthRate,
			&r.SalesPrevCumulative, &r.SalesLastYearPrevCumulative,
			&r.SalesCurrentCumulative, &r.SalesLastYearCumulative, &r.SalesCumulativeRate,
			&r.RetailPrevMonth, &r.RetailCurrentMonth, &r.RetailLastYearMonth, &r.RetailMonthRate,
			&r.RetailPrevCumulative, &r.RetailLastYearPrevCumulative,
			&r.RetailCurrentCumulative, &r.RetailLastYearCumulative, &r.RetailCumulativeRate,
			&r.RetailRatio,
			&r.CatGrainOilFood, &r.CatBeverage, &r.CatTobaccoLiquor,
			&r.CatClothing, &r.CatDailyUse, &r.CatAutomobile,
			&r.IsSmallMicro, &r.IsEatWearUse,
			&firstReportIP, &fillIP, &r.NetworkSales, &r.OpeningYear, &r.OpeningMonth,
			&r.OriginalSalesCurrentMonth, &r.OriginalRetailCurrentMonth,
			&r.SourceSheet, &r.SourceFile,
			&r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		r.FirstReportIP = firstReportIP.String
		r.FillIP = fillIP.String
		results = append(results, r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return results, nil
}

// scanWRRow 扫描单行批零企业数据
func (s *Store) scanWRRow(row *sql.Row) (*model.WholesaleRetail, error) {
	r := &model.WholesaleRetail{}
	var firstReportIP sql.NullString
	var fillIP sql.NullString
	err := row.Scan(
		&r.ID, &r.CreditCode, &r.Name, &r.IndustryCode, &r.IndustryType,
		&r.CompanyScale, &r.RowNo,
		&r.DataYear, &r.DataMonth,
		&r.SalesPrevMonth, &r.SalesCurrentMonth, &r.SalesLastYearMonth, &r.SalesMonthRate,
		&r.SalesPrevCumulative, &r.SalesLastYearPrevCumulative,
		&r.SalesCurrentCumulative, &r.SalesLastYearCumulative, &r.SalesCumulativeRate,
		&r.RetailPrevMonth, &r.RetailCurrentMonth, &r.RetailLastYearMonth, &r.RetailMonthRate,
		&r.RetailPrevCumulative, &r.RetailLastYearPrevCumulative,
		&r.RetailCurrentCumulative, &r.RetailLastYearCumulative, &r.RetailCumulativeRate,
		&r.RetailRatio,
		&r.CatGrainOilFood, &r.CatBeverage, &r.CatTobaccoLiquor,
		&r.CatClothing, &r.CatDailyUse, &r.CatAutomobile,
		&r.IsSmallMicro, &r.IsEatWearUse,
		&firstReportIP, &fillIP, &r.NetworkSales, &r.OpeningYear, &r.OpeningMonth,
		&r.OriginalSalesCurrentMonth, &r.OriginalRetailCurrentMonth,
		&r.SourceSheet, &r.SourceFile,
		&r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("record not found")
		}
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}
	r.FirstReportIP = firstReportIP.String
	r.FillIP = fillIP.String
	return r, nil
}
