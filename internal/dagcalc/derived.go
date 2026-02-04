package dagcalc

import "northstar/internal/store"

// RecalcDerivedFields 重算派生字段（增速/零售额等）
func RecalcDerivedFields(st *store.Store, year, month int) error {
	if err := st.Exec(`
		UPDATE wholesale_retail SET
			sales_month_rate = CASE
				WHEN sales_last_year_month = 0 THEN -100
				ELSE (sales_current_month - sales_last_year_month) / sales_last_year_month * 100
			END,
			sales_cumulative_rate = CASE
				WHEN sales_last_year_cumulative = 0 THEN -100
				ELSE (sales_current_cumulative - sales_last_year_cumulative) / sales_last_year_cumulative * 100
			END,
			retail_month_rate = CASE
				WHEN retail_last_year_month = 0 THEN -100
				ELSE (retail_current_month - retail_last_year_month) / retail_last_year_month * 100
			END,
			retail_cumulative_rate = CASE
				WHEN retail_last_year_cumulative = 0 THEN -100
				ELSE (retail_current_cumulative - retail_last_year_cumulative) / retail_last_year_cumulative * 100
			END,
			retail_ratio = CASE
				WHEN sales_current_month = 0 THEN NULL
				ELSE retail_current_month / sales_current_month * 100
			END
		WHERE data_year = ? AND data_month = ?
	`, year, month); err != nil {
		return err
	}

	if err := st.Exec(`
		UPDATE accommodation_catering SET
			revenue_month_rate = CASE
				WHEN revenue_last_year_month = 0 THEN -100
				ELSE (revenue_current_month - revenue_last_year_month) / revenue_last_year_month * 100
			END,
			revenue_cumulative_rate = CASE
				WHEN revenue_last_year_cumulative = 0 THEN -100
				ELSE (revenue_current_cumulative - revenue_last_year_cumulative) / revenue_last_year_cumulative * 100
			END,
			retail_current_month = food_current_month + goods_current_month,
			retail_last_year_month = food_last_year_month + goods_last_year_month
		WHERE data_year = ? AND data_month = ?
	`, year, month); err != nil {
		return err
	}
	return nil
}
