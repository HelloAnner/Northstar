package dagcalc

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"northstar/internal/rules"
	"northstar/internal/store"
)

// Plan 计算计划
type Plan struct {
	Impact []ImpactNode
	Groups []IndicatorGroup
}

// OptimizeResult 是统一调整执行结果。
type OptimizeResult struct {
	Groups       []IndicatorGroup
	AppliedRules []AppliedRule
}

type orderedTarget struct {
	ID    string
	Value float64
}

// Engine DAG 计算引擎
type Engine struct {
	graph *Graph
	store *store.Store
	year  int
	month int
	rules *rules.RuleSet
	mu    sync.RWMutex
}

// NewEngine 创建引擎
func NewEngine(graph *Graph, st *store.Store, year, month int) *Engine {
	return &Engine{
		graph: graph,
		store: st,
		year:  year,
		month: month,
	}
}

// ForwardRecalc 正向重算
func (e *Engine) ForwardRecalc(anchor NodeID) (*Plan, error) {
	if e == nil || e.store == nil {
		return nil, fmt.Errorf("missing store")
	}
	year, month := e.getPeriod()
	groups, err := RecalcAll(e.store, year, month)
	if err != nil {
		return nil, err
	}
	impact := ImpactRange(e.graph, anchor)
	return &Plan{Impact: impact, Groups: groups}, nil
}

// ReverseAdjust 反向调整
func (e *Engine) ReverseAdjust(target NodeID, newValue float64) (*Plan, error) {
	if e == nil || e.store == nil {
		return nil, fmt.Errorf("missing store")
	}
	indicatorID := e.resolveIndicatorID(target)
	if indicatorID == "" {
		return nil, fmt.Errorf("unsupported target: %s", target)
	}
	year, month := e.getPeriod()
	if _, err := ApplyIndicatorTarget(e.store, year, month, indicatorID, newValue, e.getRules(), 0); err != nil {
		return nil, err
	}
	groups, err := RecalcAll(e.store, year, month)
	if err != nil {
		return nil, err
	}
	impact := ImpactRange(e.graph, target)
	return &Plan{Impact: impact, Groups: groups}, nil
}

// SetPeriod 更新引擎使用的年月。
func (e *Engine) SetPeriod(year, month int) {
	if e == nil {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	e.year = year
	e.month = month
}

// ReloadRules 从数据库重新加载硬约束。
func (e *Engine) ReloadRules() error {
	if e == nil {
		return fmt.Errorf("missing engine")
	}

	rs, err := rules.LoadFromStore(e.store)
	if err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules = rs
	return nil
}

// Optimize 使用统一规则引擎执行目标调整。
func (e *Engine) Optimize(targets map[string]float64) (*OptimizeResult, error) {
	if e == nil || e.store == nil {
		return nil, fmt.Errorf("missing store")
	}

	year, month := e.getPeriod()
	ruleSet := e.getRules()
	applied := make([]AppliedRule, 0, len(targets))
	for _, item := range orderEngineTargets(targets) {
		itemApplied, err := ApplyIndicatorTarget(e.store, year, month, item.ID, item.Value, ruleSet, 0)
		if err != nil {
			return nil, err
		}
		applied = append(applied, itemApplied...)
	}

	groups, err := RecalcAll(e.store, year, month)
	if err != nil {
		return nil, err
	}
	return &OptimizeResult{Groups: groups, AppliedRules: applied}, nil
}

func (e *Engine) resolveIndicatorID(target NodeID) string {
	if e.graph != nil {
		if id, ok := e.graph.IndicatorIDs[target]; ok {
			return id
		}
	}
	raw := strings.TrimSpace(string(target))
	if strings.HasPrefix(raw, "indicator:") {
		return strings.TrimPrefix(raw, "indicator:")
	}
	return ""
}

func (e *Engine) getPeriod() (int, int) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.year, e.month
}

func (e *Engine) getRules() *rules.RuleSet {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.rules == nil {
		return &rules.RuleSet{}
	}
	return e.rules
}

// ConstraintCount 返回当前已加载硬约束总数。
func (e *Engine) ConstraintCount() int {
	rs := e.getRules()
	if rs == nil {
		return 0
	}
	return len(rs.Clamps) + len(rs.Filters) + len(rs.Compensates)
}

// RuleCount is an alias for ConstraintCount for backward compatibility.
func (e *Engine) RuleCount() int {
	return e.ConstraintCount()
}

func orderEngineTargets(targets map[string]float64) []orderedTarget {
	knownOrder := []string{
		"limitAbove_month_value",
		"limitAbove_month_rate",
		"limitAbove_cumulative_value",
		"limitAbove_cumulative_rate",
		"eatWearUse_month_rate",
		"microSmall_month_rate",
		"wholesale_month_rate",
		"wholesale_cumulative_rate",
		"retail_month_rate",
		"retail_cumulative_rate",
		"accommodation_month_rate",
		"accommodation_cumulative_rate",
		"catering_month_rate",
		"catering_cumulative_rate",
		"totalSocial_cumulative_value",
		"totalSocial_cumulative_rate",
	}

	ordered := make([]orderedTarget, 0, len(targets))
	seen := map[string]bool{}
	for _, id := range knownOrder {
		value, ok := targets[id]
		if !ok {
			continue
		}
		ordered = append(ordered, orderedTarget{ID: id, Value: value})
		seen[id] = true
	}

	rest := make([]string, 0, len(targets))
	for id := range targets {
		if !seen[id] {
			rest = append(rest, id)
		}
	}
	sort.Strings(rest)
	for _, id := range rest {
		ordered = append(ordered, orderedTarget{ID: id, Value: targets[id]})
	}
	return ordered
}
