package importer

import "fmt"

func (c *Coordinator) backfillCalculableFields(year, month int) error {
	if err := c.backfillWRFromSnapshot(year, month); err != nil {
		return fmt.Errorf("backfill wr from snapshot: %w", err)
	}
	if err := c.backfillACFromSnapshot(year, month); err != nil {
		return fmt.Errorf("backfill ac from snapshot: %w", err)
	}
	if err := c.backfillWRByCumulativeDiff(year, month); err != nil {
		return fmt.Errorf("backfill wr by cumulative diff: %w", err)
	}
	if err := c.backfillACByCumulativeDiff(year, month); err != nil {
		return fmt.Errorf("backfill ac by cumulative diff: %w", err)
	}
	if err := c.backfillWRByRate(year, month); err != nil {
		return fmt.Errorf("backfill wr by rate: %w", err)
	}
	if err := c.backfillACByRate(year, month); err != nil {
		return fmt.Errorf("backfill ac by rate: %w", err)
	}
	return c.backfillACRetailByComponents(year, month)
}

func (c *Coordinator) backfillWRFromSnapshot(year, month int) error {
	return c.store.Exec(`
		UPDATE wholesale_retail SET
			sales_prev_month = CASE
				WHEN sales_prev_month <> 0 THEN sales_prev_month
				ELSE COALESCE((SELECT ws.sales_current_month FROM wr_snapshot ws
					WHERE ws.snapshot_year = ? AND ws.snapshot_month = ? AND ws.credit_code = wholesale_retail.credit_code
					ORDER BY ws.id DESC LIMIT 1), 0)
			END,
			sales_prev_cumulative = CASE
				WHEN sales_prev_cumulative <> 0 THEN sales_prev_cumulative
				ELSE COALESCE((SELECT ws.sales_current_cumulative FROM wr_snapshot ws
					WHERE ws.snapshot_year = ? AND ws.snapshot_month = ? AND ws.credit_code = wholesale_retail.credit_code
					ORDER BY ws.id DESC LIMIT 1), 0)
			END,
			retail_prev_month = CASE
				WHEN retail_prev_month <> 0 THEN retail_prev_month
				ELSE COALESCE((SELECT ws.retail_current_month FROM wr_snapshot ws
					WHERE ws.snapshot_year = ? AND ws.snapshot_month = ? AND ws.credit_code = wholesale_retail.credit_code
					ORDER BY ws.id DESC LIMIT 1), 0)
			END,
			retail_prev_cumulative = CASE
				WHEN retail_prev_cumulative <> 0 THEN retail_prev_cumulative
				ELSE COALESCE((SELECT ws.retail_current_cumulative FROM wr_snapshot ws
					WHERE ws.snapshot_year = ? AND ws.snapshot_month = ? AND ws.credit_code = wholesale_retail.credit_code
					ORDER BY ws.id DESC LIMIT 1), 0)
			END,

			sales_last_year_month = CASE
				WHEN sales_last_year_month <> 0 THEN sales_last_year_month
				ELSE COALESCE((SELECT ws.sales_current_month FROM wr_snapshot ws
					WHERE ws.snapshot_year = ? AND ws.snapshot_month = ? AND ws.credit_code = wholesale_retail.credit_code
					ORDER BY ws.id DESC LIMIT 1), 0)
			END,
			sales_last_year_cumulative = CASE
				WHEN sales_last_year_cumulative <> 0 THEN sales_last_year_cumulative
				ELSE COALESCE((SELECT ws.sales_current_cumulative FROM wr_snapshot ws
					WHERE ws.snapshot_year = ? AND ws.snapshot_month = ? AND ws.credit_code = wholesale_retail.credit_code
					ORDER BY ws.id DESC LIMIT 1), 0)
			END,
			sales_last_year_prev_cumulative = CASE
				WHEN sales_last_year_prev_cumulative <> 0 THEN sales_last_year_prev_cumulative
				ELSE COALESCE((SELECT ws.sales_current_cumulative FROM wr_snapshot ws
					WHERE ws.snapshot_year = ? AND ws.snapshot_month = ? AND ws.credit_code = wholesale_retail.credit_code
					ORDER BY ws.id DESC LIMIT 1), 0)
			END,

			retail_last_year_month = CASE
				WHEN retail_last_year_month <> 0 THEN retail_last_year_month
				ELSE COALESCE((SELECT ws.retail_current_month FROM wr_snapshot ws
					WHERE ws.snapshot_year = ? AND ws.snapshot_month = ? AND ws.credit_code = wholesale_retail.credit_code
					ORDER BY ws.id DESC LIMIT 1), 0)
			END,
			retail_last_year_cumulative = CASE
				WHEN retail_last_year_cumulative <> 0 THEN retail_last_year_cumulative
				ELSE COALESCE((SELECT ws.retail_current_cumulative FROM wr_snapshot ws
					WHERE ws.snapshot_year = ? AND ws.snapshot_month = ? AND ws.credit_code = wholesale_retail.credit_code
					ORDER BY ws.id DESC LIMIT 1), 0)
			END,
			retail_last_year_prev_cumulative = CASE
				WHEN retail_last_year_prev_cumulative <> 0 THEN retail_last_year_prev_cumulative
				ELSE COALESCE((SELECT ws.retail_current_cumulative FROM wr_snapshot ws
					WHERE ws.snapshot_year = ? AND ws.snapshot_month = ? AND ws.credit_code = wholesale_retail.credit_code
					ORDER BY ws.id DESC LIMIT 1), 0)
			END
		WHERE data_year = ? AND data_month = ?
	`, year, month-1, year, month-1, year, month-1, year, month-1, year-1, month, year-1, month, year-1, month-1, year-1, month, year-1, month, year-1, month-1, year, month)
}

func (c *Coordinator) backfillACFromSnapshot(year, month int) error {
	return c.store.Exec(`
		UPDATE accommodation_catering SET
			revenue_prev_month = CASE
				WHEN revenue_prev_month <> 0 THEN revenue_prev_month
				ELSE COALESCE((SELECT s.revenue_current_month FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			revenue_prev_cumulative = CASE
				WHEN revenue_prev_cumulative <> 0 THEN revenue_prev_cumulative
				ELSE COALESCE((SELECT s.revenue_current_cumulative FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			room_prev_month = CASE
				WHEN room_prev_month <> 0 THEN room_prev_month
				ELSE COALESCE((SELECT s.room_current_month FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			room_prev_cumulative = CASE
				WHEN room_prev_cumulative <> 0 THEN room_prev_cumulative
				ELSE COALESCE((SELECT s.room_current_cumulative FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			food_prev_month = CASE
				WHEN food_prev_month <> 0 THEN food_prev_month
				ELSE COALESCE((SELECT s.food_current_month FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			food_prev_cumulative = CASE
				WHEN food_prev_cumulative <> 0 THEN food_prev_cumulative
				ELSE COALESCE((SELECT s.food_current_cumulative FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			goods_prev_month = CASE
				WHEN goods_prev_month <> 0 THEN goods_prev_month
				ELSE COALESCE((SELECT s.goods_current_month FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			goods_prev_cumulative = CASE
				WHEN goods_prev_cumulative <> 0 THEN goods_prev_cumulative
				ELSE COALESCE((SELECT s.goods_current_cumulative FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,

			revenue_last_year_month = CASE
				WHEN revenue_last_year_month <> 0 THEN revenue_last_year_month
				ELSE COALESCE((SELECT s.revenue_current_month FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			revenue_last_year_cumulative = CASE
				WHEN revenue_last_year_cumulative <> 0 THEN revenue_last_year_cumulative
				ELSE COALESCE((SELECT s.revenue_current_cumulative FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			room_last_year_month = CASE
				WHEN room_last_year_month <> 0 THEN room_last_year_month
				ELSE COALESCE((SELECT s.room_current_month FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			room_last_year_cumulative = CASE
				WHEN room_last_year_cumulative <> 0 THEN room_last_year_cumulative
				ELSE COALESCE((SELECT s.room_current_cumulative FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			food_last_year_month = CASE
				WHEN food_last_year_month <> 0 THEN food_last_year_month
				ELSE COALESCE((SELECT s.food_current_month FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			food_last_year_cumulative = CASE
				WHEN food_last_year_cumulative <> 0 THEN food_last_year_cumulative
				ELSE COALESCE((SELECT s.food_current_cumulative FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			goods_last_year_month = CASE
				WHEN goods_last_year_month <> 0 THEN goods_last_year_month
				ELSE COALESCE((SELECT s.goods_current_month FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END,
			goods_last_year_cumulative = CASE
				WHEN goods_last_year_cumulative <> 0 THEN goods_last_year_cumulative
				ELSE COALESCE((SELECT s.goods_current_cumulative FROM ac_snapshot s
					WHERE s.snapshot_year = ? AND s.snapshot_month = ? AND s.credit_code = accommodation_catering.credit_code
					ORDER BY s.id DESC LIMIT 1), 0)
			END
		WHERE data_year = ? AND data_month = ?
	`, year, month-1, year, month-1, year, month-1, year, month-1, year, month-1, year, month-1, year, month-1, year, month-1,
		year-1, month, year-1, month, year-1, month, year-1, month, year-1, month, year-1, month, year-1, month, year-1, month, year, month)
}

func (c *Coordinator) backfillWRByCumulativeDiff(year, month int) error {
	return c.store.Exec(`
		UPDATE wholesale_retail SET
			sales_current_month = CASE
				WHEN sales_current_month <> 0 THEN sales_current_month
				WHEN (sales_current_cumulative - sales_prev_cumulative) < 0 THEN 0
				ELSE (sales_current_cumulative - sales_prev_cumulative)
			END,
			retail_current_month = CASE
				WHEN retail_current_month <> 0 THEN retail_current_month
				WHEN (retail_current_cumulative - retail_prev_cumulative) < 0 THEN 0
				ELSE (retail_current_cumulative - retail_prev_cumulative)
			END,
			sales_last_year_month = CASE
				WHEN sales_last_year_month <> 0 THEN sales_last_year_month
				WHEN (sales_last_year_cumulative - sales_last_year_prev_cumulative) < 0 THEN 0
				ELSE (sales_last_year_cumulative - sales_last_year_prev_cumulative)
			END,
			retail_last_year_month = CASE
				WHEN retail_last_year_month <> 0 THEN retail_last_year_month
				WHEN (retail_last_year_cumulative - retail_last_year_prev_cumulative) < 0 THEN 0
				ELSE (retail_last_year_cumulative - retail_last_year_prev_cumulative)
			END
		WHERE data_year = ? AND data_month = ?
			AND (sales_current_month = 0 OR retail_current_month = 0 OR sales_last_year_month = 0 OR retail_last_year_month = 0)
	`, year, month)
}

func (c *Coordinator) backfillACByCumulativeDiff(year, month int) error {
	return c.store.Exec(`
		UPDATE accommodation_catering SET
			revenue_current_month = CASE
				WHEN revenue_current_month <> 0 THEN revenue_current_month
				WHEN (revenue_current_cumulative - revenue_prev_cumulative) < 0 THEN 0
				ELSE (revenue_current_cumulative - revenue_prev_cumulative)
			END,
			room_current_month = CASE
				WHEN room_current_month <> 0 THEN room_current_month
				WHEN (room_current_cumulative - room_prev_cumulative) < 0 THEN 0
				ELSE (room_current_cumulative - room_prev_cumulative)
			END,
			food_current_month = CASE
				WHEN food_current_month <> 0 THEN food_current_month
				WHEN (food_current_cumulative - food_prev_cumulative) < 0 THEN 0
				ELSE (food_current_cumulative - food_prev_cumulative)
			END,
			goods_current_month = CASE
				WHEN goods_current_month <> 0 THEN goods_current_month
				WHEN (goods_current_cumulative - goods_prev_cumulative) < 0 THEN 0
				ELSE (goods_current_cumulative - goods_prev_cumulative)
			END
		WHERE data_year = ? AND data_month = ?
			AND (revenue_current_month = 0 OR room_current_month = 0 OR food_current_month = 0 OR goods_current_month = 0)
	`, year, month)
}

func (c *Coordinator) backfillACRetailByComponents(year, month int) error {
	return c.store.Exec(`
		UPDATE accommodation_catering SET
			retail_current_month = CASE
				WHEN retail_current_month <> 0 THEN retail_current_month
				WHEN (food_current_month + goods_current_month) < 0 THEN 0
				ELSE (food_current_month + goods_current_month)
			END,
			retail_last_year_month = CASE
				WHEN retail_last_year_month <> 0 THEN retail_last_year_month
				WHEN (food_last_year_month + goods_last_year_month) < 0 THEN 0
				ELSE (food_last_year_month + goods_last_year_month)
			END
		WHERE data_year = ? AND data_month = ? AND (retail_current_month = 0 OR retail_last_year_month = 0)
	`, year, month)
}

func (c *Coordinator) backfillWRByRate(year, month int) error {
	return c.store.Exec(`
		UPDATE wholesale_retail SET
			sales_current_month = CASE
				WHEN sales_current_month <> 0 THEN sales_current_month
				WHEN sales_last_year_month = 0 OR sales_month_rate IS NULL THEN sales_current_month
				ELSE sales_last_year_month * (1 + sales_month_rate / 100.0)
			END,
			sales_current_cumulative = CASE
				WHEN sales_current_cumulative <> 0 THEN sales_current_cumulative
				WHEN sales_last_year_cumulative = 0 OR sales_cumulative_rate IS NULL THEN sales_current_cumulative
				ELSE sales_last_year_cumulative * (1 + sales_cumulative_rate / 100.0)
			END,
			retail_current_month = CASE
				WHEN retail_current_month <> 0 THEN retail_current_month
				WHEN retail_last_year_month = 0 OR retail_month_rate IS NULL THEN retail_current_month
				ELSE retail_last_year_month * (1 + retail_month_rate / 100.0)
			END,
			retail_current_cumulative = CASE
				WHEN retail_current_cumulative <> 0 THEN retail_current_cumulative
				WHEN retail_last_year_cumulative = 0 OR retail_cumulative_rate IS NULL THEN retail_current_cumulative
				ELSE retail_last_year_cumulative * (1 + retail_cumulative_rate / 100.0)
			END
		WHERE data_year = ? AND data_month = ?
			AND (
				(sales_current_month = 0 AND sales_last_year_month <> 0 AND sales_month_rate IS NOT NULL) OR
				(sales_current_cumulative = 0 AND sales_last_year_cumulative <> 0 AND sales_cumulative_rate IS NOT NULL) OR
				(retail_current_month = 0 AND retail_last_year_month <> 0 AND retail_month_rate IS NOT NULL) OR
				(retail_current_cumulative = 0 AND retail_last_year_cumulative <> 0 AND retail_cumulative_rate IS NOT NULL)
			)
	`, year, month)
}

func (c *Coordinator) backfillACByRate(year, month int) error {
	return c.store.Exec(`
		UPDATE accommodation_catering SET
			revenue_current_month = CASE
				WHEN revenue_current_month <> 0 THEN revenue_current_month
				WHEN revenue_last_year_month = 0 OR revenue_month_rate IS NULL THEN revenue_current_month
				ELSE revenue_last_year_month * (1 + revenue_month_rate / 100.0)
			END,
			revenue_current_cumulative = CASE
				WHEN revenue_current_cumulative <> 0 THEN revenue_current_cumulative
				WHEN revenue_last_year_cumulative = 0 OR revenue_cumulative_rate IS NULL THEN revenue_current_cumulative
				ELSE revenue_last_year_cumulative * (1 + revenue_cumulative_rate / 100.0)
			END
		WHERE data_year = ? AND data_month = ?
			AND (
				(revenue_current_month = 0 AND revenue_last_year_month <> 0 AND revenue_month_rate IS NOT NULL) OR
				(revenue_current_cumulative = 0 AND revenue_last_year_cumulative <> 0 AND revenue_cumulative_rate IS NOT NULL)
			)
	`, year, month)
}
