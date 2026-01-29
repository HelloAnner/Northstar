package excel

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"northstar/internal/model"
)

func TestApplyMonthReportBaseline_FillsWholesaleFromTemplate(t *testing.T) {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root not found from: %s", filename)
		}
		dir = parent
	}
	templatePath := filepath.Join(dir, "prd", "12月月报（定）.xlsx")

	companies := []*model.Company{
		{
			ID:                    "c1",
			IndustryType:          model.IndustryWholesale,
			IndustryCode:          "5152",
			SalesLastYearMonth:    848,
			SalesLastYearCumulative: 13698,
			RetailLastYearMonth:   251,
			RetailLastYearCumulative: 4403,
			SalesCurrentMonth:     0,
			SalesCurrentCumulative: 11289,
			RetailCurrentMonth:    0,
			RetailCurrentCumulative: 3066,
		},
	}

	updated, err := ApplyMonthReportBaseline(templatePath, companies)
	if err != nil {
		t.Fatalf("ApplyMonthReportBaseline err: %v", err)
	}
	if updated != 1 {
		t.Fatalf("updated=%d, want 1", updated)
	}

	c := companies[0]
	if c.SalesCurrentMonth != 493 {
		t.Fatalf("SalesCurrentMonth=%v, want 493", c.SalesCurrentMonth)
	}
	// 模板应补齐 1-12累计
	if c.SalesCurrentCumulative != 11782 {
		t.Fatalf("SalesCurrentCumulative=%v, want 11782", c.SalesCurrentCumulative)
	}
}
