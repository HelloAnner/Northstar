package v3

import (
	"math"

	"northstar/internal/calculator"
)

func roundIndicatorGroupsInPlace(groups []calculator.IndicatorGroup) {
	for gi := range groups {
		for ii := range groups[gi].Indicators {
			groups[gi].Indicators[ii].Value = math.Round(groups[gi].Indicators[ii].Value)
		}
	}
}
