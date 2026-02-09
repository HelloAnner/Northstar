/**
 * 指标中心与规则中心存储
 *
 * @author Anner
 * Created on 2026/2/6
 */

package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// IndicatorDefinition 指标定义
type IndicatorDefinition struct {
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	GroupCode    string  `json:"groupCode"`
	GroupName    string  `json:"groupName"`
	GroupOrder   int     `json:"groupOrder"`
	Description  string  `json:"description"`
	Formula      string  `json:"formula"`
	Unit         string  `json:"unit"`
	FloatMin     float64 `json:"floatMin"`
	FloatMax     float64 `json:"floatMax"`
	DisplayOrder int     `json:"displayOrder"`
	Enabled      bool    `json:"enabled"`
}

// RuleDefinition 规则定义
type RuleDefinition struct {
	RuleCode       string `json:"ruleCode"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Expression     string `json:"expression"`
	Severity       string `json:"severity"`
	Suggestion     string `json:"suggestion"`
	PreferenceJSON string `json:"preferenceJson"`
	DisplayOrder   int    `json:"displayOrder"`
	Enabled        bool   `json:"enabled"`
}

// RuleIndicatorLink 规则与指标关联
type RuleIndicatorLink struct {
	RuleCode      string  `json:"ruleCode"`
	IndicatorCode string  `json:"indicatorCode"`
	RelationLabel string  `json:"relationLabel"`
	Weight        float64 `json:"weight"`
	DisplayOrder  int     `json:"displayOrder"`
}

// ListIndicatorDefinitions 列出指标定义
func (s *Store) ListIndicatorDefinitions(enabledOnly bool) ([]IndicatorDefinition, error) {
	query := `
		SELECT
			code, name, group_code, group_name, group_order,
			description, formula, unit, float_min, float_max,
			display_order, enabled
		FROM indicator_definitions
	`
	if enabledOnly {
		query += " WHERE enabled = 1"
	}
	query += " ORDER BY group_order ASC, display_order ASC"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query indicator definitions failed: %w", err)
	}
	defer rows.Close()

	result := make([]IndicatorDefinition, 0)
	for rows.Next() {
		item, err := scanIndicatorDefinition(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate indicator definitions failed: %w", err)
	}
	return result, nil
}

func scanIndicatorDefinition(rows *sql.Rows) (IndicatorDefinition, error) {
	var item IndicatorDefinition
	var enabled int
	err := rows.Scan(
		&item.Code,
		&item.Name,
		&item.GroupCode,
		&item.GroupName,
		&item.GroupOrder,
		&item.Description,
		&item.Formula,
		&item.Unit,
		&item.FloatMin,
		&item.FloatMax,
		&item.DisplayOrder,
		&enabled,
	)
	if err != nil {
		return IndicatorDefinition{}, fmt.Errorf("scan indicator definition failed: %w", err)
	}
	item.Enabled = enabled == 1
	return item, nil
}

// UpsertIndicatorDefinition 新增或更新指标定义
func (s *Store) UpsertIndicatorDefinition(def IndicatorDefinition) error {
	def.Name = strings.TrimSpace(def.Name)
	def.Code = strings.TrimSpace(def.Code)
	if def.Name == "" {
		return fmt.Errorf("indicator name cannot be empty")
	}
	if def.Code == "" || def.Code != def.Name {
		def.Code = def.Name
	}
	if strings.TrimSpace(def.GroupCode) == "" {
		def.GroupCode = def.GroupName
	}

	enabled := 0
	if def.Enabled {
		enabled = 1
	}

	_, err := s.db.Exec(`
		INSERT INTO indicator_definitions (
			code, name, group_code, group_name, group_order,
			description, formula, unit, float_min, float_max,
			display_order, enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			code = excluded.code,
			group_code = excluded.group_code,
			group_name = excluded.group_name,
			group_order = excluded.group_order,
			description = excluded.description,
			formula = excluded.formula,
			unit = excluded.unit,
			float_min = excluded.float_min,
			float_max = excluded.float_max,
			display_order = excluded.display_order,
			enabled = excluded.enabled,
			updated_at = CURRENT_TIMESTAMP
	`,
		def.Code,
		def.Name,
		def.GroupCode,
		def.GroupName,
		def.GroupOrder,
		def.Description,
		def.Formula,
		def.Unit,
		def.FloatMin,
		def.FloatMax,
		def.DisplayOrder,
		enabled,
	)
	if err != nil {
		return fmt.Errorf("upsert indicator definition failed: %w", err)
	}
	return nil
}

// ListRuleDefinitions 列出规则定义
func (s *Store) ListRuleDefinitions(enabledOnly bool) ([]RuleDefinition, error) {
	query := `
		SELECT
			rule_code, name, description, expression, severity,
			suggestion, preference_json, display_order, enabled
		FROM rule_definitions
	`
	if enabledOnly {
		query += " WHERE enabled = 1"
	}
	query += " ORDER BY display_order ASC"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query rule definitions failed: %w", err)
	}
	defer rows.Close()

	result := make([]RuleDefinition, 0)
	for rows.Next() {
		item, err := scanRuleDefinition(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rule definitions failed: %w", err)
	}
	return result, nil
}

func scanRuleDefinition(rows *sql.Rows) (RuleDefinition, error) {
	var item RuleDefinition
	var enabled int
	err := rows.Scan(
		&item.RuleCode,
		&item.Name,
		&item.Description,
		&item.Expression,
		&item.Severity,
		&item.Suggestion,
		&item.PreferenceJSON,
		&item.DisplayOrder,
		&enabled,
	)
	if err != nil {
		return RuleDefinition{}, fmt.Errorf("scan rule definition failed: %w", err)
	}
	item.Enabled = enabled == 1
	return item, nil
}

// UpsertRuleDefinition 新增或更新规则定义
func (s *Store) UpsertRuleDefinition(rule RuleDefinition) error {
	rule.Name = strings.TrimSpace(rule.Name)
	rule.RuleCode = strings.TrimSpace(rule.RuleCode)
	if rule.Name == "" {
		return fmt.Errorf("rule name cannot be empty")
	}
	if rule.RuleCode == "" || rule.RuleCode != rule.Name {
		rule.RuleCode = rule.Name
	}

	enabled := 0
	if rule.Enabled {
		enabled = 1
	}

	_, err := s.db.Exec(`
		INSERT INTO rule_definitions (
			rule_code, name, description, expression, severity,
			suggestion, preference_json, display_order, enabled
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			rule_code = excluded.rule_code,
			description = excluded.description,
			expression = excluded.expression,
			severity = excluded.severity,
			suggestion = excluded.suggestion,
			preference_json = excluded.preference_json,
			display_order = excluded.display_order,
			enabled = excluded.enabled,
			updated_at = CURRENT_TIMESTAMP
	`,
		rule.RuleCode,
		rule.Name,
		rule.Description,
		rule.Expression,
		rule.Severity,
		rule.Suggestion,
		rule.PreferenceJSON,
		rule.DisplayOrder,
		enabled,
	)
	if err != nil {
		return fmt.Errorf("upsert rule definition failed: %w", err)
	}
	return nil
}

// ListRuleIndicatorLinks 列出规则关联指标
func (s *Store) ListRuleIndicatorLinks(ruleCode string) ([]RuleIndicatorLink, error) {
	query := `
		SELECT rule_code, indicator_code, relation_label, weight, display_order
		FROM rule_indicator_links
	`
	args := make([]interface{}, 0, 1)
	if ruleCode != "" {
		query += " WHERE rule_code = ?"
		args = append(args, ruleCode)
	}
	query += " ORDER BY rule_code ASC, display_order ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query rule indicator links failed: %w", err)
	}
	defer rows.Close()

	result := make([]RuleIndicatorLink, 0)
	for rows.Next() {
		var item RuleIndicatorLink
		if err := rows.Scan(&item.RuleCode, &item.IndicatorCode, &item.RelationLabel, &item.Weight, &item.DisplayOrder); err != nil {
			return nil, fmt.Errorf("scan rule indicator link failed: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rule indicator links failed: %w", err)
	}
	return result, nil
}

// ReplaceRuleIndicatorLinks 覆盖规则关联指标
func (s *Store) ReplaceRuleIndicatorLinks(ruleCode string, links []RuleIndicatorLink) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin replace rule links tx failed: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM rule_indicator_links WHERE rule_code = ?", ruleCode); err != nil {
		return fmt.Errorf("delete old rule links failed: %w", err)
	}

	for _, link := range links {
		if _, err := tx.Exec(`
			INSERT INTO rule_indicator_links (rule_code, indicator_code, relation_label, weight, display_order)
			VALUES (?, ?, ?, ?, ?)
		`, ruleCode, link.IndicatorCode, link.RelationLabel, link.Weight, link.DisplayOrder); err != nil {
			return fmt.Errorf("insert rule link failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace rule links tx failed: %w", err)
	}
	return nil
}
