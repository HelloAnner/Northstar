/**
 * 指标与规则表结构测试
 *
 * @author Anner
 * Created on 2026/2/6
 */

package store

import (
	"path/filepath"
	"testing"
)

func TestSchema_IncludesIndicatorRuleTables(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	expect := []string{
		"indicator_definitions",
		"rule_definitions",
		"rule_indicator_links",
	}

	for _, name := range expect {
		if !tableExists(st.db, name) {
			t.Fatalf("missing table: %s", name)
		}
	}
}

func TestSchema_SeedsIndicatorAndRules(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	defs, err := st.ListIndicatorDefinitions(true)
	if err != nil {
		t.Fatalf("list indicator definitions failed: %v", err)
	}
	if len(defs) < 16 {
		t.Fatalf("expected >=16 indicator definitions, got %d", len(defs))
	}

	needIndicators := map[string]bool{
		"限上社零额_当月值":  false,
		"限上社零额增速_累计": false,
		"小微企业增速_当月":  false,
		"社零总额增速_累计":  false,
	}
	for _, def := range defs {
		if _, ok := needIndicators[def.Code]; ok {
			needIndicators[def.Code] = true
		}
	}
	for code, exists := range needIndicators {
		if !exists {
			t.Fatalf("missing seeded indicator definition: %s", code)
		}
	}

	rules, err := st.ListRuleDefinitions(true)
	if err != nil {
		t.Fatalf("list rule definitions failed: %v", err)
	}
	if len(rules) < 10 {
		t.Fatalf("expected >=10 rule definitions, got %d", len(rules))
	}

	needRules := map[string]bool{
		"规则P2-1 行业增速区间与差异约束": false,
		"规则P2-2 批发业零销比约束":    false,
		"规则P2-9 新进企业同期累计上限":  false,
		"规则P2-10 小微与吃穿用优先策略": false,
	}
	for _, rule := range rules {
		if _, ok := needRules[rule.RuleCode]; ok {
			needRules[rule.RuleCode] = true
		}
	}
	for code, exists := range needRules {
		if !exists {
			t.Fatalf("missing seeded rule definition: %s", code)
		}
	}

	links, err := st.ListRuleIndicatorLinks("")
	if err != nil {
		t.Fatalf("list rule indicator links failed: %v", err)
	}
	if len(links) < 15 {
		t.Fatalf("expected >=15 seeded rule links, got %d", len(links))
	}
}

func TestSchema_SeedsRuleThresholdConfig(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	keys := []string{
		"rule_growth_abs_limit",
		"rule_wholesale_ratio_limit",
		"rule_retail_big_growth_limit",
		"rule_new_company_wholesale_year_cap",
		"rule_priority_target",
	}

	for _, key := range keys {
		value, getErr := st.GetConfig(key)
		if getErr != nil {
			t.Fatalf("missing config key %s: %v", key, getErr)
		}
		if value == "" {
			t.Fatalf("config key %s should not be empty", key)
		}
	}
}
