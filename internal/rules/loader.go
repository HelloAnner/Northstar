/**
 * 规则加载器
 *
 * @author Anner
 * Created on 2026/3/14
 */

package rules

import (
	"fmt"

	"northstar/internal/store"
)

// RuleSet 是运行时规则集合。
type RuleSet struct {
	Clamps      []*ClampTargetConstraint
	Filters     []*FilterAllocationConstraint
	Compensates []*CompensateConstraint
}

// ConstraintStore 是加载约束所需的最小存储接口。
type ConstraintStore interface {
	ListAdjustmentConstraints(enabledOnly bool) ([]store.AdjustmentConstraint, error)
}

// LoadFromStore 从数据库加载启用的约束，构建 RuleSet。
func LoadFromStore(st ConstraintStore) (*RuleSet, error) {
	if st == nil {
		return emptyRuleSet(), nil
	}

	constraints, err := st.ListAdjustmentConstraints(true)
	if err != nil {
		return nil, err
	}

	rs := emptyRuleSet()
	for _, c := range constraints {
		appendConstraint(rs, c)
	}
	return rs, nil
}

func appendConstraint(rs *RuleSet, c store.AdjustmentConstraint) {
	switch c.Type {
	case "clamp_target":
		rs.Clamps = append(rs.Clamps, &ClampTargetConstraint{
			ID:          idFromInt(c.ID),
			IndicatorID: c.IndicatorID,
			Min:         c.MinValue,
			Max:         c.MaxValue,
		})
	case "filter_allocation":
		rs.Filters = append(rs.Filters, &FilterAllocationConstraint{
			ID:          idFromInt(c.ID),
			IndicatorID: c.IndicatorID,
			Filter:      c.FilterMode,
		})
	case "compensate":
		rs.Compensates = append(rs.Compensates, &CompensateConstraint{
			ID:        idFromInt(c.ID),
			TriggerID: c.TriggerID,
			EnsureID:  c.EnsureID,
			Relation:  c.Relation,
			Tolerance: c.Tolerance,
		})
	}
}

func idFromInt(id int64) string {
	if id == 0 {
		return ""
	}
	return fmt.Sprintf("%d", id)
}

func emptyRuleSet() *RuleSet {
	return &RuleSet{
		Clamps:      []*ClampTargetConstraint{},
		Filters:     []*FilterAllocationConstraint{},
		Compensates: []*CompensateConstraint{},
	}
}
