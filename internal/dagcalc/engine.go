package dagcalc

import (
	"fmt"
	"strings"

	"northstar/internal/store"
)

// Plan 计算计划
type Plan struct {
	Impact []ImpactNode
	Groups []IndicatorGroup
}

// Engine DAG 计算引擎
type Engine struct {
	graph *Graph
	store *store.Store
	year  int
	month int
}

// NewEngine 创建引擎
func NewEngine(graph *Graph, st *store.Store, year, month int) *Engine {
	return &Engine{graph: graph, store: st, year: year, month: month}
}

// ForwardRecalc 正向重算
func (e *Engine) ForwardRecalc(anchor NodeID) (*Plan, error) {
	if e == nil || e.store == nil {
		return nil, fmt.Errorf("missing store")
	}
	groups, err := RecalcAll(e.store, e.year, e.month)
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
	if err := ApplyIndicatorTarget(e.store, e.year, e.month, indicatorID, newValue); err != nil {
		return nil, err
	}
	groups, err := RecalcAll(e.store, e.year, e.month)
	if err != nil {
		return nil, err
	}
	impact := ImpactRange(e.graph, target)
	return &Plan{Impact: impact, Groups: groups}, nil
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
