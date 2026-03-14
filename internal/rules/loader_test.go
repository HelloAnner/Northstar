/**
 * 规则加载测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDispatchesRulesByType(t *testing.T) {
	rolePath := filepath.Join(t.TempDir(), "role.json")
	content := `{
  "version": "1.0",
  "rules": [
    {"id":"c1","name":"clamp","type":"clamp_target","indicator":"retail_month_rate","min":-5,"max":10},
    {"id":"f1","name":"filter","type":"filter_allocation","indicator":"retail_month_rate","filter":"positive_current"},
    {"id":"p1","name":"compensate","type":"compensate","trigger":"retail_month_rate","ensure":"wholesale_month_rate","relation":"gte","tolerance":1},
    {"id":"x1","name":"skip","type":"unknown_rule"}
  ]
}`
	if err := os.WriteFile(rolePath, []byte(content), 0644); err != nil {
		t.Fatalf("write role.json: %v", err)
	}

	rs, err := Load(rolePath)
	if err != nil {
		t.Fatalf("load rules: %v", err)
	}
	if len(rs.Clamps) != 1 {
		t.Fatalf("unexpected clamp count: %d", len(rs.Clamps))
	}
	if len(rs.Filters) != 1 {
		t.Fatalf("unexpected filter count: %d", len(rs.Filters))
	}
	if len(rs.Compensates) != 1 {
		t.Fatalf("unexpected compensate count: %d", len(rs.Compensates))
	}
	if rs.Clamps[0].IndicatorID != "retail_month_rate" {
		t.Fatalf("unexpected clamp indicator: %s", rs.Clamps[0].IndicatorID)
	}
	if rs.Filters[0].Filter != "positive_current" {
		t.Fatalf("unexpected filter mode: %s", rs.Filters[0].Filter)
	}
	if rs.Compensates[0].EnsureID != "wholesale_month_rate" {
		t.Fatalf("unexpected ensure indicator: %s", rs.Compensates[0].EnsureID)
	}
}

func TestLoadMissingFileReturnsEmptyRuleSet(t *testing.T) {
	rs, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("load missing file: %v", err)
	}
	if len(rs.Clamps) != 0 || len(rs.Filters) != 0 || len(rs.Compensates) != 0 {
		t.Fatalf("expected empty ruleset: %+v", rs)
	}
}
