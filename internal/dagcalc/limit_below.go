package dagcalc

import "northstar/internal/store"

const (
	defaultWeightSmallMicro = 0.3
	defaultWeightEatWearUse = 0.3
	defaultWeightSample     = 0.4
)

// LimitBelowEstimate 限下估算结果
type LimitBelowEstimate struct {
	LastYearCumulative float64
	PrevRate           float64
	RateChange         float64
	CurrentRate        float64
	GrowthFactor       float64
	CurrentCumulative  float64
}

func estimateLimitBelowCumulative(st *store.Store, year, month int) (LimitBelowEstimate, error) {
	lastYearCumulative := configFloatOrDefault(st, "last_year_limit_below_cumulative", 0)
	smallMicroRatePrev := configFloatOrDefault(st, "small_micro_rate_prev", 0)
	eatWearUseRatePrev := configFloatOrDefault(st, "eat_wear_use_rate_prev", 0)
	sampleRatePrev := configFloatOrDefault(st, "sample_rate_prev", 0)
	sampleRateMonth := configFloatOrDefault(st, "sample_rate_month", 0)
	weightSmallMicro := configFloatOrDefault(st, "weight_small_micro", defaultWeightSmallMicro)
	weightEatWearUse := configFloatOrDefault(st, "weight_eat_wear_use", defaultWeightEatWearUse)
	weightSample := configFloatOrDefault(st, "weight_sample", defaultWeightSample)
	provinceRateChange := configFloatOrDefault(st, "province_limit_below_rate_change", 0)

	smallMicroRateMonth, err := computeMicroSmallRate(st, year, month)
	if err != nil {
		return LimitBelowEstimate{}, err
	}
	eatWearUseRateMonth, err := computeEatWearUseRate(st, year, month)
	if err != nil {
		return LimitBelowEstimate{}, err
	}

	prevRate := smallMicroRatePrev*weightSmallMicro +
		eatWearUseRatePrev*weightEatWearUse +
		sampleRatePrev*weightSample
	rateChange := (smallMicroRateMonth-smallMicroRatePrev)*weightSmallMicro +
		(eatWearUseRateMonth-eatWearUseRatePrev)*weightEatWearUse +
		(sampleRateMonth-sampleRatePrev)*weightSample +
		provinceRateChange
	currentRate := prevRate + rateChange
	growthFactor := 1 + currentRate/100
	currentCumulative := lastYearCumulative * growthFactor
	if currentCumulative < 0 {
		currentCumulative = 0
	}

	return LimitBelowEstimate{
		LastYearCumulative: lastYearCumulative,
		PrevRate:           prevRate,
		RateChange:         rateChange,
		CurrentRate:        currentRate,
		GrowthFactor:       growthFactor,
		CurrentCumulative:  currentCumulative,
	}, nil
}

func computeEatWearUseRate(st *store.Store, year, month int) (float64, error) {
	currentSum, _, err := sumAndCountWR(st, year, month, "", "is_eat_wear_use", "retail_current_month")
	if err != nil {
		return 0, err
	}
	lastYearSum, _, err := sumAndCountWR(st, year, month, "", "is_eat_wear_use", "retail_last_year_month")
	if err != nil {
		return 0, err
	}
	if lastYearSum == 0 {
		return 0, nil
	}
	return (currentSum - lastYearSum) / lastYearSum * 100, nil
}

func configFloatOrDefault(st *store.Store, key string, fallback float64) float64 {
	value, err := st.GetConfigFloat(key)
	if err != nil {
		return fallback
	}
	return value
}
