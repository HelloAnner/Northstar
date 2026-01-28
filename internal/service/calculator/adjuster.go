package calculator

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"northstar/internal/model"
	"northstar/internal/service/store"
)

// Adjuster 指标调整器：把“编辑指标”转换为对底层数据（企业/配置）的修改，然后触发联动计算
type Adjuster struct {
	store  *store.MemoryStore
	engine *Engine
}

func NewAdjuster(store *store.MemoryStore, engine *Engine) *Adjuster {
	return &Adjuster{store: store, engine: engine}
}

func (a *Adjuster) Adjust(key string, value float64) (*model.Indicators, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errors.New("key is required")
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return nil, errors.New("value is invalid")
	}

	switch {
	case key == "limitAboveMonthValue":
		return a.adjustRetailMonthAll(value)
	case key == "limitAboveMonthRate":
		return a.adjustRetailMonthRateAll(value)
	case key == "limitAboveCumulativeValue":
		return a.adjustRetailCumulativeAll(value)
	case key == "limitAboveCumulativeRate":
		return a.adjustRetailCumulativeRateAll(value)
	case key == "eatWearUseMonthRate":
		return a.adjustRetailMonthRateEatWearUse(value)
	case key == "microSmallMonthRate":
		return a.adjustRetailMonthRateMicroSmall(value)
	case key == "totalSocialCumulativeValue":
		return a.adjustTotalSocialCumulativeValue(value)
	case key == "totalSocialCumulativeRate":
		return a.adjustTotalSocialCumulativeRate(value)
	case strings.HasPrefix(key, "industry."):
		return a.adjustIndustryRate(key, value)
	default:
		return nil, fmt.Errorf("unsupported key: %s", key)
	}
}

// ==================== Retail: All ====================

func (a *Adjuster) adjustRetailMonthAll(targetSum float64) (*model.Indicators, error) {
	return a.adjustCompaniesFloat(
		func(c *model.Company) bool { return true },
		func(c *model.Company) float64 { return c.RetailCurrentMonth },
		func(c *model.Company) float64 { return 0 },
		func(c *model.Company) float64 {
			if c.SalesCurrentMonth > 0 {
				return c.SalesCurrentMonth
			}
			return c.RetailCurrentMonth
		},
		func(updates map[string]float64) error { return a.store.BatchUpdateCompanyRetail(updates) },
		targetSum,
	)
}

func (a *Adjuster) adjustRetailMonthRateAll(targetRate float64) (*model.Indicators, error) {
	companies := a.store.GetAllCompanies()
	lastYearSum := 0.0
	for _, c := range companies {
		lastYearSum += c.RetailLastYearMonth
	}
	targetSum := (1 + targetRate) * lastYearSum
	return a.adjustRetailMonthAll(targetSum)
}

func (a *Adjuster) adjustRetailCumulativeAll(targetSum float64) (*model.Indicators, error) {
	return a.adjustCompaniesFloat(
		func(c *model.Company) bool { return true },
		func(c *model.Company) float64 { return c.RetailCurrentCumulative },
		func(c *model.Company) float64 { return 0 },
		func(c *model.Company) float64 {
			if c.SalesCurrentCumulative > 0 {
				return c.SalesCurrentCumulative
			}
			return c.RetailCurrentCumulative
		},
		func(updates map[string]float64) error { return a.store.BatchUpdateCompanyRetailCumulative(updates) },
		targetSum,
	)
}

func (a *Adjuster) adjustRetailCumulativeRateAll(targetRate float64) (*model.Indicators, error) {
	companies := a.store.GetAllCompanies()
	lastYearSum := 0.0
	for _, c := range companies {
		lastYearSum += c.RetailLastYearCumulative
	}
	targetSum := (1 + targetRate) * lastYearSum
	return a.adjustRetailCumulativeAll(targetSum)
}

// ==================== Retail: Subsets ====================

func (a *Adjuster) adjustRetailMonthRateEatWearUse(targetRate float64) (*model.Indicators, error) {
	companies := a.store.GetAllCompanies()
	lastYearSum := 0.0
	for _, c := range companies {
		if c.IsEatWearUse {
			lastYearSum += c.RetailLastYearMonth
		}
	}
	targetSum := (1 + targetRate) * lastYearSum
	return a.adjustCompaniesFloat(
		func(c *model.Company) bool { return c.IsEatWearUse },
		func(c *model.Company) float64 { return c.RetailCurrentMonth },
		func(c *model.Company) float64 { return 0 },
		func(c *model.Company) float64 {
			if c.SalesCurrentMonth > 0 {
				return c.SalesCurrentMonth
			}
			return c.RetailCurrentMonth
		},
		func(updates map[string]float64) error { return a.store.BatchUpdateCompanyRetail(updates) },
		targetSum,
	)
}

func (a *Adjuster) adjustRetailMonthRateMicroSmall(targetRate float64) (*model.Indicators, error) {
	companies := a.store.GetAllCompanies()
	lastYearSum := 0.0
	for _, c := range companies {
		if c.IsMicroSmall() {
			lastYearSum += c.RetailLastYearMonth
		}
	}
	targetSum := (1 + targetRate) * lastYearSum
	return a.adjustCompaniesFloat(
		func(c *model.Company) bool { return c.IsMicroSmall() },
		func(c *model.Company) float64 { return c.RetailCurrentMonth },
		func(c *model.Company) float64 { return 0 },
		func(c *model.Company) float64 {
			if c.SalesCurrentMonth > 0 {
				return c.SalesCurrentMonth
			}
			return c.RetailCurrentMonth
		},
		func(updates map[string]float64) error { return a.store.BatchUpdateCompanyRetail(updates) },
		targetSum,
	)
}

// ==================== Industry (Sales) ====================

func (a *Adjuster) adjustIndustryRate(key string, targetRate float64) (*model.Indicators, error) {
	parts := strings.Split(key, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid industry key: %s", key)
	}
	industry := model.IndustryType(parts[1])
	period := parts[2]
	if period != "monthRate" && period != "cumulativeRate" {
		return nil, fmt.Errorf("invalid industry period: %s", period)
	}

	companies := a.store.GetAllCompanies()
	if period == "monthRate" {
		lastYearSum := 0.0
		for _, c := range companies {
			if c.IndustryType == industry {
				lastYearSum += c.SalesLastYearMonth
			}
		}
		targetSum := (1 + targetRate) * lastYearSum
		return a.adjustCompaniesFloat(
			func(c *model.Company) bool { return c.IndustryType == industry },
			func(c *model.Company) float64 { return c.SalesCurrentMonth },
			func(c *model.Company) float64 { return c.RetailCurrentMonth },
			func(c *model.Company) float64 { return c.SalesCurrentMonth + math.Abs(targetSum) + 1 },
			func(updates map[string]float64) error { return a.store.BatchUpdateCompanySales(updates) },
			targetSum,
		)
	}

	lastYearSum := 0.0
	for _, c := range companies {
		if c.IndustryType == industry {
			lastYearSum += c.SalesLastYearCumulative
		}
	}
	targetSum := (1 + targetRate) * lastYearSum
	return a.adjustCompaniesFloat(
		func(c *model.Company) bool { return c.IndustryType == industry },
		func(c *model.Company) float64 { return c.SalesCurrentCumulative },
		func(c *model.Company) float64 { return c.RetailCurrentCumulative },
		func(c *model.Company) float64 { return c.SalesCurrentCumulative + math.Abs(targetSum) + 1 },
		func(updates map[string]float64) error { return a.store.BatchUpdateCompanySalesCumulative(updates) },
		targetSum,
	)
}

// ==================== Total Social ====================

func (a *Adjuster) adjustTotalSocialCumulativeValue(targetTotal float64) (*model.Indicators, error) {
	indicators := a.engine.Calculate()
	limitAbove := indicators.LimitAboveCumulativeValue
	microRate := indicators.MicroSmallMonthRate

	estimatedLimitBelow := targetTotal - limitAbove
	if estimatedLimitBelow < 0 {
		estimatedLimitBelow = 0
	}

	denom := 1 + microRate
	lastYearLimitBelow := 0.0
	if denom > 0.0000001 {
		lastYearLimitBelow = estimatedLimitBelow / denom
	}
	if lastYearLimitBelow < 0 {
		lastYearLimitBelow = 0
	}

	a.store.UpdateConfig(map[string]interface{}{
		"lastYearLimitBelowCumulative": lastYearLimitBelow,
	})
	return a.engine.Calculate(), nil
}

func (a *Adjuster) adjustTotalSocialCumulativeRate(targetRate float64) (*model.Indicators, error) {
	sums := a.engine.GetSums()
	indicators := a.engine.Calculate()

	A := indicators.LimitAboveCumulativeValue
	B := sums.AllRetailLastYearCumulative
	r := indicators.MicroSmallMonthRate

	denom := targetRate - r
	if math.Abs(denom) < 0.0000001 {
		return nil, errors.New("targetRate is too close to microSmallMonthRate")
	}

	x := (A - B*(1+targetRate)) / denom
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return nil, errors.New("invalid computed config")
	}
	if x < 0 {
		x = 0
	}

	a.store.UpdateConfig(map[string]interface{}{
		"lastYearLimitBelowCumulative": x,
	})
	return a.engine.Calculate(), nil
}

// ==================== Core Adjust Algorithm ====================

type companyAdjustItem struct {
	ID  string
	Cur float64
	Min float64
	Max float64
}

func (a *Adjuster) adjustCompaniesFloat(
	filter func(*model.Company) bool,
	get func(*model.Company) float64,
	getMin func(*model.Company) float64,
	getMax func(*model.Company) float64,
	applyBatch func(map[string]float64) error,
	targetSum float64,
) (*model.Indicators, error) {
	companies := a.store.GetAllCompanies()
	items := make([]companyAdjustItem, 0, len(companies))
	currentSum := 0.0
	for _, c := range companies {
		if !filter(c) {
			continue
		}
		cur := get(c)
		minV := getMin(c)
		maxV := getMax(c)
		if maxV < minV {
			maxV = minV
		}
		items = append(items, companyAdjustItem{
			ID:  c.ID,
			Cur: cur,
			Min: minV,
			Max: maxV,
		})
		currentSum += cur
	}

	if len(items) == 0 {
		return a.engine.Calculate(), nil
	}

	delta := targetSum - currentSum
	if math.Abs(delta) < 0.0001 {
		return a.engine.Calculate(), nil
	}

	updates := make(map[string]float64)
	if delta > 0 {
		sort.SliceStable(items, func(i, j int) bool {
			return (items[i].Max - items[i].Cur) > (items[j].Max - items[j].Cur)
		})
		remaining := delta
		for _, it := range items {
			capacity := it.Max - it.Cur
			if capacity <= 0 {
				continue
			}
			add := math.Min(capacity, remaining)
			if add <= 0 {
				continue
			}
			updates[it.ID] = it.Cur + add
			remaining -= add
			if remaining <= 0.0001 {
				break
			}
		}
		if remaining > 0.01 {
			return nil, errors.New("not enough capacity to increase")
		}
	} else {
		sort.SliceStable(items, func(i, j int) bool {
			return (items[i].Cur - items[i].Min) > (items[j].Cur - items[j].Min)
		})
		remaining := -delta
		for _, it := range items {
			capacity := it.Cur - it.Min
			if capacity <= 0 {
				continue
			}
			sub := math.Min(capacity, remaining)
			if sub <= 0 {
				continue
			}
			updates[it.ID] = it.Cur - sub
			remaining -= sub
			if remaining <= 0.0001 {
				break
			}
		}
		if remaining > 0.01 {
			return nil, errors.New("not enough capacity to decrease")
		}
	}

	if err := applyBatch(updates); err != nil {
		return nil, err
	}
	return a.engine.Calculate(), nil
}
