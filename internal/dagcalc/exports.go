package dagcalc

import (
	"northstar/internal/model"
	"northstar/internal/store"
)

// SumAndCountWR 对外封装
func SumAndCountWR(st *store.Store, year, month int, industryType string, flagField string, field string) (float64, int, error) {
	return sumAndCountWR(st, year, month, industryType, flagField, field)
}

// SumAndCountAC 对外封装
func SumAndCountAC(st *store.Store, year, month int, industryType string, field string) (float64, int, error) {
	return sumAndCountAC(st, year, month, industryType, field)
}

// SumAndCountACDerivedRetailMonth 对外封装
func SumAndCountACDerivedRetailMonth(st *store.Store, year, month int, industryType string, hasRows bool) (float64, int, error) {
	return sumAndCountACDerivedRetailMonth(st, year, month, industryType, hasRows)
}

// SumAndCountACDerivedRetailLastYearMonth 对外封装
func SumAndCountACDerivedRetailLastYearMonth(st *store.Store, year, month int, industryType string) (float64, int, error) {
	return sumAndCountACDerivedRetailLastYearMonth(st, year, month, industryType)
}

// SumAndCountACDerivedRetailCumulative 对外封装
func SumAndCountACDerivedRetailCumulative(st *store.Store, year, month int, industryType string, hasRows bool) (float64, int, error) {
	return sumAndCountACDerivedRetailCumulative(st, year, month, industryType, hasRows)
}

// SumAndCountACDerivedRetailLastYearCumulative 对外封装
func SumAndCountACDerivedRetailLastYearCumulative(st *store.Store, year, month int, industryType string) (float64, int, error) {
	return sumAndCountACDerivedRetailLastYearCumulative(st, year, month, industryType)
}

// ComputeMicroSmallRate 对外封装
func ComputeMicroSmallRate(st *store.Store, year, month int) (float64, error) {
	return computeMicroSmallRate(st, year, month)
}

// LoadWRRowsForAdjust 对外封装
func LoadWRRowsForAdjust(st *store.Store, year, month int, industryType, flagField string) ([]*model.WholesaleRetail, error) {
	return loadWRRowsForAdjust(st, year, month, industryType, flagField)
}

// LoadACRowsForAdjust 对外封装
func LoadACRowsForAdjust(st *store.Store, year, month int, industryType string) ([]*model.AccommodationCatering, error) {
	return loadACRowsForAdjust(st, year, month, industryType)
}

// PickWRFieldValue 对外封装
func PickWRFieldValue(r *model.WholesaleRetail, field string) float64 {
	return pickWRFieldValue(r, field)
}

// PickACFieldValue 对外封装
func PickACFieldValue(r *model.AccommodationCatering, field string) float64 {
	return pickACFieldValue(r, field)
}

// UpdateWRFieldValues 对外封装
func UpdateWRFieldValues(st *store.Store, field string, rows []*model.WholesaleRetail, values []float64) error {
	return updateWRFieldValues(st, field, rows, values)
}

// UpdateACFieldValues 对外封装
func UpdateACFieldValues(st *store.Store, field string, rows []*model.AccommodationCatering, values []float64) error {
	return updateACFieldValues(st, field, rows, values)
}

// UpdateACDerivedRetailValues 对外封装
func UpdateACDerivedRetailValues(st *store.Store, foodField, goodsField string, rows []*model.AccommodationCatering, totals []float64) error {
	return updateACDerivedRetailValues(st, foodField, goodsField, rows, totals)
}

// RandomizeAllocations 对外封装
func RandomizeAllocations(target float64, bases []float64, scales []int) []float64 {
	return randomizeAllocations(target, bases, scales)
}
