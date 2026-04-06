/**
 * 指标与规则标识迁移
 *
 * @author Anner
 * Created on 2026/2/6
 */

package store

import (
	"database/sql"
	"fmt"
)

type textReplace struct {
	from string
	to   string
}

type sqlExecer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func (s *Store) normalizeIndicatorRuleIdentifiers() error {
	// Skip migration if indicator_definitions table doesn't exist (fresh database)
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='indicator_definitions'`).Scan(&n); err != nil || n == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin normalize identifier tx failed: %w", err)
	}
	defer tx.Rollback()

	if err := normalizeIndicatorCodes(tx); err != nil {
		return err
	}
	if err := normalizeRuleCodes(tx); err != nil {
		return err
	}
	if err := normalizeIndicatorFormulaText(tx); err != nil {
		return err
	}
	if err := normalizeRuleExpressionText(tx); err != nil {
		return err
	}
	if err := ensureDefinitionUniqueIndexes(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit normalize identifier tx failed: %w", err)
	}
	return nil
}

func normalizeIndicatorCodes(exec sqlExecer) error {
	mappings := []textReplace{
		{from: "limitAbove_month_value", to: "限上社零额_当月值"},
		{from: "limitAbove_month_rate", to: "限上社零额增速_当月"},
		{from: "limitAbove_cumulative_value", to: "限上社零额_累计值"},
		{from: "limitAbove_cumulative_rate", to: "限上社零额增速_累计"},
		{from: "eatWearUse_month_rate", to: "吃穿用增速_当月"},
		{from: "microSmall_month_rate", to: "小微企业增速_当月"},
		{from: "wholesale_month_rate", to: "批发业销售额增速_当月"},
		{from: "wholesale_cumulative_rate", to: "批发业销售额增速_累计"},
		{from: "retail_month_rate", to: "零售业销售额增速_当月"},
		{from: "retail_cumulative_rate", to: "零售业销售额增速_累计"},
		{from: "accommodation_month_rate", to: "住宿业营业额增速_当月"},
		{from: "accommodation_cumulative_rate", to: "住宿业营业额增速_累计"},
		{from: "catering_month_rate", to: "餐饮业营业额增速_当月"},
		{from: "catering_cumulative_rate", to: "餐饮业营业额增速_累计"},
		{from: "limitBelow_prev_month_rate", to: "限下增速_上月"},
		{from: "limitBelow_delta_month_rate", to: "限下增速变动量_本月"},
		{from: "limitBelow_month_rate", to: "限下增速_本月"},
		{from: "limitBelow_cumulative_value", to: "限下累计估算值_累计"},
		{from: "totalSocial_cumulative_value", to: "社零总额_累计值"},
		{from: "totalSocial_cumulative_rate", to: "社零总额增速_累计"},
	}

	for _, item := range mappings {
		if _, err := exec.Exec(`
			DELETE FROM indicator_definitions
			WHERE code = ?
			  AND EXISTS (SELECT 1 FROM indicator_definitions legacy WHERE legacy.code = ?)
		`, item.to, item.from); err != nil {
			return fmt.Errorf("delete duplicate indicator before migrate failed: %w", err)
		}
		if _, err := exec.Exec(`UPDATE indicator_definitions SET code = ?, name = ? WHERE code = ?`, item.to, item.to, item.from); err != nil {
			return fmt.Errorf("migrate indicator code failed: %w", err)
		}
		if _, err := exec.Exec(`
			DELETE FROM rule_indicator_links
			WHERE indicator_code = ?
			  AND EXISTS (
				SELECT 1 FROM rule_indicator_links legacy
				WHERE legacy.rule_code = rule_indicator_links.rule_code
				  AND legacy.indicator_code = ?
			  )
		`, item.to, item.from); err != nil {
			return fmt.Errorf("delete duplicate rule links before indicator migrate failed: %w", err)
		}
		if _, err := exec.Exec(`UPDATE rule_indicator_links SET indicator_code = ? WHERE indicator_code = ?`, item.to, item.from); err != nil {
			return fmt.Errorf("migrate rule link indicator failed: %w", err)
		}
	}

	if _, err := exec.Exec(`UPDATE indicator_definitions SET name = code WHERE name <> code`); err != nil {
		return fmt.Errorf("normalize indicator name=code failed: %w", err)
	}
	if _, err := exec.Exec(`
		DELETE FROM indicator_definitions
		WHERE rowid NOT IN (
			SELECT MIN(rowid) FROM indicator_definitions GROUP BY name
		)
	`); err != nil {
		return fmt.Errorf("deduplicate indicator name failed: %w", err)
	}
	if _, err := exec.Exec(`
		DELETE FROM rule_indicator_links
		WHERE rowid NOT IN (
			SELECT MIN(rowid) FROM rule_indicator_links GROUP BY rule_code, indicator_code
		)
	`); err != nil {
		return fmt.Errorf("deduplicate rule indicator links failed: %w", err)
	}
	if _, err := exec.Exec(`
		UPDATE indicator_definitions
		SET group_code = group_name
		WHERE group_name <> ''
		  AND (group_code = '' OR group_code IN ('limit_above', 'special_rate', 'industry_rate', 'limit_below_model', 'total_social', 'custom'))
	`); err != nil {
		return fmt.Errorf("normalize indicator group code failed: %w", err)
	}
	return nil
}

func normalizeRuleCodes(exec sqlExecer) error {
	if _, err := exec.Exec(`
		UPDATE rule_indicator_links
		SET rule_code = (SELECT name FROM rule_definitions WHERE rule_definitions.rule_code = rule_indicator_links.rule_code)
		WHERE rule_code IN (SELECT rule_code FROM rule_definitions WHERE rule_code <> name)
	`); err != nil {
		return fmt.Errorf("migrate rule links rule code failed: %w", err)
	}
	if _, err := exec.Exec(`
		DELETE FROM rule_indicator_links
		WHERE rowid NOT IN (
			SELECT MIN(rowid) FROM rule_indicator_links GROUP BY rule_code, indicator_code
		)
	`); err != nil {
		return fmt.Errorf("deduplicate rule links after rule migrate failed: %w", err)
	}
	if _, err := exec.Exec(`UPDATE rule_definitions SET rule_code = name WHERE rule_code <> name`); err != nil {
		return fmt.Errorf("normalize rule code=name failed: %w", err)
	}
	if _, err := exec.Exec(`
		DELETE FROM rule_definitions
		WHERE rowid NOT IN (
			SELECT MIN(rowid) FROM rule_definitions GROUP BY name
		)
	`); err != nil {
		return fmt.Errorf("deduplicate rule name failed: %w", err)
	}
	return nil
}

func normalizeIndicatorFormulaText(exec sqlExecer) error {
	replacements := []textReplace{
		{from: "limitAbove_month_value", to: "限上社零额_当月值"},
		{from: "limitAbove_month_rate", to: "限上社零额增速_当月"},
		{from: "limitAbove_cumulative_value", to: "限上社零额_累计值"},
		{from: "limitAbove_cumulative_rate", to: "限上社零额增速_累计"},
		{from: "eatWearUse_month_rate", to: "吃穿用增速_当月"},
		{from: "microSmall_month_rate", to: "小微企业增速_当月"},
		{from: "wholesale_month_rate", to: "批发业销售额增速_当月"},
		{from: "wholesale_cumulative_rate", to: "批发业销售额增速_累计"},
		{from: "retail_month_rate", to: "零售业销售额增速_当月"},
		{from: "retail_cumulative_rate", to: "零售业销售额增速_累计"},
		{from: "accommodation_month_rate", to: "住宿业营业额增速_当月"},
		{from: "accommodation_cumulative_rate", to: "住宿业营业额增速_累计"},
		{from: "catering_month_rate", to: "餐饮业营业额增速_当月"},
		{from: "catering_cumulative_rate", to: "餐饮业营业额增速_累计"},
		{from: "limitBelow_prev_month_rate", to: "限下增速_上月"},
		{from: "limitBelow_delta_month_rate", to: "限下增速变动量_本月"},
		{from: "limitBelow_month_rate", to: "限下增速_本月"},
		{from: "limitBelow_cumulative_value", to: "限下累计估算值_累计"},
		{from: "totalSocial_cumulative_value", to: "社零总额_累计值"},
		{from: "totalSocial_cumulative_rate", to: "社零总额增速_累计"},
		{from: "percent_diff(", to: "同比增速("},
		{from: "wr_retail_current_month_sum", to: "批零零售额_当月汇总"},
		{from: "wr_retail_last_year_month_sum", to: "批零零售额_上年当月汇总"},
		{from: "wr_retail_current_cumulative_sum", to: "批零零售额_累计汇总"},
		{from: "wr_retail_last_year_cumulative_sum", to: "批零零售额_上年累计汇总"},
		{from: "ac_derived_retail_current_month_sum", to: "住餐折算零售额_当月汇总"},
		{from: "ac_derived_retail_last_year_month_sum", to: "住餐折算零售额_上年当月汇总"},
		{from: "ac_derived_retail_current_cumulative_sum", to: "住餐折算零售额_累计汇总"},
		{from: "ac_derived_retail_last_year_cumulative_sum", to: "住餐折算零售额_上年累计汇总"},
		{from: "wr_eat_wear_use_current_month_sum", to: "吃穿用零售额_当月汇总"},
		{from: "wr_eat_wear_use_last_year_month_sum", to: "吃穿用零售额_上年当月汇总"},
		{from: "wr_micro_small_current_month_sum", to: "小微零售额_当月汇总"},
		{from: "wr_micro_small_last_year_month_sum", to: "小微零售额_上年当月汇总"},
		{from: "wr_wholesale_sales_current_month_sum", to: "批发销售额_当月汇总"},
		{from: "wr_wholesale_sales_last_year_month_sum", to: "批发销售额_上年当月汇总"},
		{from: "wr_wholesale_sales_current_cumulative_sum", to: "批发销售额_累计汇总"},
		{from: "wr_wholesale_sales_last_year_cumulative_sum", to: "批发销售额_上年累计汇总"},
		{from: "wr_retail_sales_current_month_sum", to: "零售销售额_当月汇总"},
		{from: "wr_retail_sales_last_year_month_sum", to: "零售销售额_上年当月汇总"},
		{from: "wr_retail_sales_current_cumulative_sum", to: "零售销售额_累计汇总"},
		{from: "wr_retail_sales_last_year_cumulative_sum", to: "零售销售额_上年累计汇总"},
		{from: "ac_accommodation_revenue_current_month_sum", to: "住宿营业额_当月汇总"},
		{from: "ac_accommodation_revenue_last_year_month_sum", to: "住宿营业额_上年当月汇总"},
		{from: "ac_accommodation_revenue_current_cumulative_sum", to: "住宿营业额_累计汇总"},
		{from: "ac_accommodation_revenue_last_year_cumulative_sum", to: "住宿营业额_上年累计汇总"},
		{from: "ac_catering_revenue_current_month_sum", to: "餐饮营业额_当月汇总"},
		{from: "ac_catering_revenue_last_year_month_sum", to: "餐饮营业额_上年当月汇总"},
		{from: "ac_catering_revenue_current_cumulative_sum", to: "餐饮营业额_累计汇总"},
		{from: "ac_catering_revenue_last_year_cumulative_sum", to: "餐饮营业额_上年累计汇总"},
		{from: "small_micro_rate_prev", to: "小微增速_上月配置"},
		{from: "eat_wear_use_rate_prev", to: "吃穿用增速_上月配置"},
		{from: "sample_rate_prev", to: "抽样增速_上月配置"},
		{from: "small_micro_rate_month", to: "小微增速_本月配置"},
		{from: "eat_wear_use_rate_month", to: "吃穿用增速_本月配置"},
		{from: "sample_rate_month", to: "抽样增速_本月配置"},
		{from: "weight_small_micro", to: "小微权重_配置"},
		{from: "weight_eat_wear_use", to: "吃穿用权重_配置"},
		{from: "weight_sample", to: "抽样权重_配置"},
		{from: "province_limit_below_rate_change", to: "全省限下增速变动量_配置"},
		{from: "limit_below_last_cumulative", to: "限下累计估算_上年值"},
	}
	return applyTextReplacements(exec, "indicator_definitions", "formula", replacements)
}

func normalizeRuleExpressionText(exec sqlExecer) error {
	replacements := []textReplace{
		{from: "abs(", to: "绝对值("},
		{from: "industry_month_rate", to: "行业增速_当月"},
		{from: "industry_cumulative_rate", to: "行业增速_累计"},
		{from: "rule_growth_abs_limit", to: "行业增速绝对值上限"},
		{from: "wholesale_retail_ratio", to: "批发零销比"},
		{from: "rule_wholesale_ratio_limit", to: "批发零销比上限"},
		{from: "retail_growth_rate", to: "零售大个体增速"},
		{from: "rule_retail_big_growth_limit", to: "零售大个体增速上限"},
		{from: "has_food", to: "存在餐费"},
		{from: "room_income", to: "客房收入"},
		{from: "food_income", to: "餐费收入"},
		{from: "has_room", to: "存在客房"},
		{from: "rate_decimal_delta", to: "住餐增速小数变动"},
		{from: "rule_room_food_delta_limit", to: "住餐增速小数变动阈值"},
		{from: "big_individual_growth_rate", to: "大个体增速"},
		{from: "wholesale_rate_decimal_delta", to: "批发增速小数变动"},
		{from: "new_company_cap_check", to: "新进企业累计上限校验"},
		{from: "priority_target", to: "优先目标增速"},
		{from: "rule_priority_target", to: "优先目标增速阈值"},
		{from: " AND ", to: " && "},
	}
	return applyTextReplacements(exec, "rule_definitions", "expression", replacements)
}

func applyTextReplacements(exec sqlExecer, table string, column string, replacements []textReplace) error {
	for _, item := range replacements {
		stmt := fmt.Sprintf(`UPDATE %s SET %s = REPLACE(%s, ?, ?)`, table, column, column)
		if _, err := exec.Exec(stmt, item.from, item.to); err != nil {
			return fmt.Errorf("replace text failed on %s.%s: %w", table, column, err)
		}
	}
	return nil
}

func ensureDefinitionUniqueIndexes(exec sqlExecer) error {
	if _, err := exec.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uniq_indicator_definition_name ON indicator_definitions(name)`); err != nil {
		return fmt.Errorf("create uniq indicator name index failed: %w", err)
	}
	if _, err := exec.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uniq_rule_definition_name ON rule_definitions(name)`); err != nil {
		return fmt.Errorf("create uniq rule name index failed: %w", err)
	}
	return nil
}
