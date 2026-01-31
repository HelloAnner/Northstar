package store

import (
	"database/sql"
	"fmt"
	"strings"

	"northstar/internal/model"
)

// BatchInsertAC 批量插入住餐企业数据
func (s *Store) BatchInsertAC(records []*model.AccommodationCatering) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO accommodation_catering (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			revenue_prev_month, revenue_current_month, revenue_last_year_month, revenue_month_rate,
			revenue_prev_cumulative, revenue_current_cumulative,
			revenue_last_year_cumulative, revenue_cumulative_rate,
			room_prev_month, room_current_month, room_last_year_month,
			room_prev_cumulative, room_current_cumulative, room_last_year_cumulative,
			food_prev_month, food_current_month, food_last_year_month,
			food_prev_cumulative, food_current_cumulative, food_last_year_cumulative,
			goods_prev_month, goods_current_month, goods_last_year_month,
			goods_prev_cumulative, goods_current_cumulative, goods_last_year_cumulative,
			retail_current_month, retail_last_year_month,
			is_small_micro, is_eat_wear_use,
			first_report_ip, fill_ip, network_sales, opening_year, opening_month,
			original_revenue_current_month, original_room_current_month,
			original_food_current_month, original_goods_current_month,
			source_sheet, source_file
		) VALUES (
			?, ?, ?, ?, ?, ?,
			?, ?,
			?, ?, ?, ?,
			?, ?,
			?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?, ?,
			?, ?,
			?, ?,
			?, ?, ?, ?, ?,
			?, ?,
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
			r.RevenuePrevMonth, r.RevenueCurrentMonth, r.RevenueLastYearMonth, r.RevenueMonthRate,
			r.RevenuePrevCumulative, r.RevenueCurrentCumulative,
			r.RevenueLastYearCumulative, r.RevenueCumulativeRate,
			r.RoomPrevMonth, r.RoomCurrentMonth, r.RoomLastYearMonth,
			r.RoomPrevCumulative, r.RoomCurrentCumulative, r.RoomLastYearCumulative,
			r.FoodPrevMonth, r.FoodCurrentMonth, r.FoodLastYearMonth,
			r.FoodPrevCumulative, r.FoodCurrentCumulative, r.FoodLastYearCumulative,
			r.GoodsPrevMonth, r.GoodsCurrentMonth, r.GoodsLastYearMonth,
			r.GoodsPrevCumulative, r.GoodsCurrentCumulative, r.GoodsLastYearCumulative,
			r.RetailCurrentMonth, r.RetailLastYearMonth,
			r.IsSmallMicro, r.IsEatWearUse,
			r.FirstReportIP, r.FillIP, r.NetworkSales, r.OpeningYear, r.OpeningMonth,
			r.OriginalRevenueCurrentMonth, r.OriginalRoomCurrentMonth,
			r.OriginalFoodCurrentMonth, r.OriginalGoodsCurrentMonth,
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

// ACQueryOptions 住餐企业查询选项
type ACQueryOptions struct {
	DataYear     *int
	DataMonth    *int
	IndustryType *string // accommodation/catering
	CompanyScale *int
	IsSmallMicro *int
	IsEatWearUse *int
	Limit        int
	Offset       int
}

// GetACByYearMonth 获取指定年月的住餐企业数据
func (s *Store) GetACByYearMonth(opts ACQueryOptions) ([]*model.AccommodationCatering, error) {
	query := "SELECT * FROM accommodation_catering WHERE 1=1"
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

	return s.scanACRows(rows)
}

// UpdateAC 更新住餐企业数据
func (s *Store) UpdateAC(id int64, updates map[string]interface{}) error {
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

	query := fmt.Sprintf("UPDATE accommodation_catering SET %s WHERE id = ?",
		strings.Join(setClauses, ", "))

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update: %w", err)
	}

	return nil
}

// DeleteACByYearMonth 删除指定年月的住餐企业数据
func (s *Store) DeleteACByYearMonth(year, month int) error {
	_, err := s.db.Exec("DELETE FROM accommodation_catering WHERE data_year = ? AND data_month = ?",
		year, month)
	if err != nil {
		return fmt.Errorf("failed to delete: %w", err)
	}
	return nil
}

// CountAC 统计住餐企业数量
func (s *Store) CountAC(opts ACQueryOptions) (int, error) {
	query := "SELECT COUNT(*) FROM accommodation_catering WHERE 1=1"
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

// GetACByID 根据 ID 获取住餐企业
func (s *Store) GetACByID(id int64) (*model.AccommodationCatering, error) {
	row := s.db.QueryRow("SELECT * FROM accommodation_catering WHERE id = ?", id)
	return s.scanACRow(row)
}

// scanACRows 扫描多行住餐企业数据
func (s *Store) scanACRows(rows *sql.Rows) ([]*model.AccommodationCatering, error) {
	var results []*model.AccommodationCatering

	for rows.Next() {
		r := &model.AccommodationCatering{}
		var firstReportIP sql.NullString
		var fillIP sql.NullString
		err := rows.Scan(
			&r.ID, &r.CreditCode, &r.Name, &r.IndustryCode, &r.IndustryType,
			&r.CompanyScale, &r.RowNo,
			&r.DataYear, &r.DataMonth,
			&r.RevenuePrevMonth, &r.RevenueCurrentMonth, &r.RevenueLastYearMonth, &r.RevenueMonthRate,
			&r.RevenuePrevCumulative, &r.RevenueCurrentCumulative,
			&r.RevenueLastYearCumulative, &r.RevenueCumulativeRate,
			&r.RoomPrevMonth, &r.RoomCurrentMonth, &r.RoomLastYearMonth,
			&r.RoomPrevCumulative, &r.RoomCurrentCumulative, &r.RoomLastYearCumulative,
			&r.FoodPrevMonth, &r.FoodCurrentMonth, &r.FoodLastYearMonth,
			&r.FoodPrevCumulative, &r.FoodCurrentCumulative, &r.FoodLastYearCumulative,
			&r.GoodsPrevMonth, &r.GoodsCurrentMonth, &r.GoodsLastYearMonth,
			&r.GoodsPrevCumulative, &r.GoodsCurrentCumulative, &r.GoodsLastYearCumulative,
			&r.RetailCurrentMonth, &r.RetailLastYearMonth,
			&r.IsSmallMicro, &r.IsEatWearUse,
			&firstReportIP, &fillIP, &r.NetworkSales, &r.OpeningYear, &r.OpeningMonth,
			&r.OriginalRevenueCurrentMonth, &r.OriginalRoomCurrentMonth,
			&r.OriginalFoodCurrentMonth, &r.OriginalGoodsCurrentMonth,
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

// scanACRow 扫描单行住餐企业数据
func (s *Store) scanACRow(row *sql.Row) (*model.AccommodationCatering, error) {
	r := &model.AccommodationCatering{}
	var firstReportIP sql.NullString
	var fillIP sql.NullString
	err := row.Scan(
		&r.ID, &r.CreditCode, &r.Name, &r.IndustryCode, &r.IndustryType,
		&r.CompanyScale, &r.RowNo,
		&r.DataYear, &r.DataMonth,
		&r.RevenuePrevMonth, &r.RevenueCurrentMonth, &r.RevenueLastYearMonth, &r.RevenueMonthRate,
		&r.RevenuePrevCumulative, &r.RevenueCurrentCumulative,
		&r.RevenueLastYearCumulative, &r.RevenueCumulativeRate,
		&r.RoomPrevMonth, &r.RoomCurrentMonth, &r.RoomLastYearMonth,
		&r.RoomPrevCumulative, &r.RoomCurrentCumulative, &r.RoomLastYearCumulative,
		&r.FoodPrevMonth, &r.FoodCurrentMonth, &r.FoodLastYearMonth,
		&r.FoodPrevCumulative, &r.FoodCurrentCumulative, &r.FoodLastYearCumulative,
		&r.GoodsPrevMonth, &r.GoodsCurrentMonth, &r.GoodsLastYearMonth,
		&r.GoodsPrevCumulative, &r.GoodsCurrentCumulative, &r.GoodsLastYearCumulative,
		&r.RetailCurrentMonth, &r.RetailLastYearMonth,
		&r.IsSmallMicro, &r.IsEatWearUse,
		&firstReportIP, &fillIP, &r.NetworkSales, &r.OpeningYear, &r.OpeningMonth,
		&r.OriginalRevenueCurrentMonth, &r.OriginalRoomCurrentMonth,
		&r.OriginalFoodCurrentMonth, &r.OriginalGoodsCurrentMonth,
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
