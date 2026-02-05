/**
 * 汇总表查询
 *
 * @author Anner
 * Created on 2026/2/5
 */

package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// GetSummaryMicroSmallRate 获取小微增速
func (s *Store) GetSummaryMicroSmallRate(year, month int) (*float64, error) {
	return getSummaryRate(s, "summary_micro_small", year, month)
}

// GetSummaryEatWearUseRate 获取吃穿用增速
func (s *Store) GetSummaryEatWearUseRate(year, month int) (*float64, error) {
	return getSummaryRate(s, "summary_eat_wear_use", year, month)
}

func getSummaryRate(s *Store, table string, year, month int) (*float64, error) {
	row := s.QueryRow(fmt.Sprintf(`
		SELECT value_current, value_last, rate, row_key
		FROM %s
		WHERE data_year = ? AND data_month = ?
		ORDER BY row_no DESC
		LIMIT 5
	`, table), year, month)

	var current sql.NullFloat64
	var last sql.NullFloat64
	var rate sql.NullFloat64
	var rowKey sql.NullString
	if err := row.Scan(&current, &last, &rate, &rowKey); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if rate.Valid {
		v := rate.Float64
		return &v, nil
	}
	if current.Valid && last.Valid && last.Float64 != 0 {
		v := (current.Float64 - last.Float64) / last.Float64 * 100
		return &v, nil
	}

	// 兜底：若合计行在后续，允许再查一次
	if rowKey.Valid && strings.TrimSpace(rowKey.String) != "" {
		return nil, nil
	}
	return nil, nil
}
