package store

import "fmt"

// YearMonthStat 可用年月统计
type YearMonthStat struct {
	Year  int `json:"year"`
	Month int `json:"month"`

	WRCount int `json:"wrCount"`
	ACCount int `json:"acCount"`
	Total   int `json:"totalCompanies"`
}

// ListAvailableYearMonths 列出当前数据库中存在数据的年月（按年/月倒序）
func (s *Store) ListAvailableYearMonths() ([]YearMonthStat, error) {
	rows, err := s.db.Query(`
		WITH ym AS (
			SELECT DISTINCT data_year AS y, data_month AS m FROM wholesale_retail
			UNION
			SELECT DISTINCT data_year AS y, data_month AS m FROM accommodation_catering
		)
		SELECT
			ym.y,
			ym.m,
			(SELECT COUNT(1) FROM wholesale_retail WHERE data_year = ym.y AND data_month = ym.m) AS wr_count,
			(SELECT COUNT(1) FROM accommodation_catering WHERE data_year = ym.y AND data_month = ym.m) AS ac_count
		FROM ym
		ORDER BY ym.y DESC, ym.m DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query available months failed: %w", err)
	}
	defer rows.Close()

	var out []YearMonthStat
	for rows.Next() {
		var it YearMonthStat
		if err := rows.Scan(&it.Year, &it.Month, &it.WRCount, &it.ACCount); err != nil {
			return nil, fmt.Errorf("scan available months failed: %w", err)
		}
		it.Total = it.WRCount + it.ACCount
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate available months failed: %w", err)
	}
	return out, nil
}

