/**
 * 规则约束定义
 *
 * @author Anner
 * Created on 2026/3/14
 */

package rules

// CompanyRow 是过滤规则需要的最小企业视图。
type CompanyRow struct {
	RowID        int64
	CompanyScale int
	IsSmallMicro bool
	CurrentValue float64
}

// ClampTargetConstraint 表示目标值裁剪规则。
type ClampTargetConstraint struct {
	ID          string
	IndicatorID string
	Min         *float64
	Max         *float64
}

// Clamp 返回裁剪后的目标值及是否发生变化。
func (c *ClampTargetConstraint) Clamp(indicatorID string, target float64) (float64, bool) {
	if c == nil || c.IndicatorID != indicatorID {
		return target, false
	}

	next := target
	if c.Min != nil && next < *c.Min {
		next = *c.Min
	}
	if c.Max != nil && next > *c.Max {
		next = *c.Max
	}
	return next, next != target
}

// FilterAllocationConstraint 表示企业参与范围过滤规则。
type FilterAllocationConstraint struct {
	ID          string
	IndicatorID string
	Filter      string
}

// Apply 返回过滤后的企业列表及是否有变化。
func (c *FilterAllocationConstraint) Apply(indicatorID string, companies []CompanyRow) ([]CompanyRow, bool) {
	if c == nil || c.IndicatorID != indicatorID {
		return companies, false
	}

	filtered := filterByMode(companies, c.Filter)
	if len(filtered) == len(companies) {
		return filtered, false
	}
	return filtered, true
}

// CompensateConstraint 表示指标间补偿关系。
type CompensateConstraint struct {
	ID        string
	TriggerID string
	EnsureID  string
	Relation  string
	Tolerance float64
}

// Check 返回是否需要补偿及补偿目标值。
func (c *CompensateConstraint) Check(triggerID string, indicators map[string]float64) (bool, float64) {
	if c == nil || c.TriggerID != triggerID {
		return false, 0
	}

	trigger := indicators[c.TriggerID]
	ensure := indicators[c.EnsureID]
	if c.Relation == "gte" {
		target := trigger - c.Tolerance
		if ensure < target {
			return true, target
		}
		return false, 0
	}
	if c.Relation == "lte" {
		target := trigger + c.Tolerance
		if ensure > target {
			return true, target
		}
		return false, 0
	}
	return false, 0
}
