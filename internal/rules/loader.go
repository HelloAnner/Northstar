/**
 * 规则加载器
 *
 * @author Anner
 * Created on 2026/3/14
 */

package rules

import (
	"encoding/json"
	"os"
)

// RuleSet 是运行时规则集合。
type RuleSet struct {
	Clamps      []*ClampTargetConstraint
	Filters     []*FilterAllocationConstraint
	Compensates []*CompensateConstraint
}

type rawRule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Type        string   `json:"type"`
	Indicator   string   `json:"indicator,omitempty"`
	Min         *float64 `json:"min"`
	Max         *float64 `json:"max"`
	Filter      string   `json:"filter,omitempty"`
	Trigger     string   `json:"trigger,omitempty"`
	Ensure      string   `json:"ensure,omitempty"`
	Relation    string   `json:"relation,omitempty"`
	Tolerance   float64  `json:"tolerance,omitempty"`
}

type rawRoleJSON struct {
	Version    string    `json:"version"`
	UpdatedAt  string    `json:"updatedAt,omitempty"`
	SourceFile string    `json:"sourceFile,omitempty"`
	Rules      []rawRule `json:"rules"`
}

// Load 读取 role.json 并构建 RuleSet。
func Load(path string) (*RuleSet, error) {
	if path == "" {
		return emptyRuleSet(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyRuleSet(), nil
		}
		return nil, err
	}

	var raw rawRoleJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return buildRuleSet(raw.Rules), nil
}

func buildRuleSet(rawRules []rawRule) *RuleSet {
	rs := emptyRuleSet()
	for _, item := range rawRules {
		appendRule(rs, item)
	}
	return rs
}

func appendRule(rs *RuleSet, item rawRule) {
	if item.Type == "clamp_target" {
		rs.Clamps = append(rs.Clamps, &ClampTargetConstraint{
			ID:          item.ID,
			IndicatorID: item.Indicator,
			Min:         item.Min,
			Max:         item.Max,
		})
		return
	}
	if item.Type == "filter_allocation" {
		rs.Filters = append(rs.Filters, &FilterAllocationConstraint{
			ID:          item.ID,
			IndicatorID: item.Indicator,
			Filter:      item.Filter,
		})
		return
	}
	if item.Type == "compensate" {
		rs.Compensates = append(rs.Compensates, &CompensateConstraint{
			ID:        item.ID,
			TriggerID: item.Trigger,
			EnsureID:  item.Ensure,
			Relation:  item.Relation,
			Tolerance: item.Tolerance,
		})
	}
}

func emptyRuleSet() *RuleSet {
	return &RuleSet{
		Clamps:      []*ClampTargetConstraint{},
		Filters:     []*FilterAllocationConstraint{},
		Compensates: []*CompensateConstraint{},
	}
}
