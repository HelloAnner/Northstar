package dagcalc

import "northstar/internal/store"

// RecalcAll 统一重算入口（派生字段 + 指标）
func RecalcAll(st *store.Store, year, month int) ([]IndicatorGroup, error) {
	if err := RecalcDerivedFields(st, year, month); err != nil {
		return nil, err
	}
	return CalculateIndicators(st, year, month)
}
