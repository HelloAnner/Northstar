package calculator

import (
	"northstar/internal/dagcalc"
	"northstar/internal/store"
)

// Indicator 指标定义
type Indicator = dagcalc.Indicator

// IndicatorGroup 指标分组
type IndicatorGroup = dagcalc.IndicatorGroup

// Calculator 指标计算器
type Calculator struct {
	store *store.Store
}

// NewCalculator 创建计算器
func NewCalculator(store *store.Store) *Calculator {
	return &Calculator{store: store}
}

// CalculateAll 计算所有16个指标
func (c *Calculator) CalculateAll(year, month int) ([]IndicatorGroup, error) {
	return dagcalc.CalculateIndicators(c.store, year, month)
}
