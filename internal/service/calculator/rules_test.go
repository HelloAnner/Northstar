package calculator

import (
	"testing"

	"northstar/internal/model"
)

func TestRuleRetailNotExceedSales(t *testing.T) {
	c := &model.Company{
		RetailCurrentMonth: 100,
		SalesCurrentMonth:  90,
	}
	errs := ValidateCompany(c)
	if !containsString(errs, "零售额不能超过销售额") {
		t.Fatalf("expected rule error, got: %v", errs)
	}
}

func containsString(items []string, want string) bool {
	for _, it := range items {
		if it == want {
			return true
		}
	}
	return false
}
