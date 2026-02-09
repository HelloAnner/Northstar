/**
 * 可扩展规则引擎测试
 *
 * @author Anner
 * Created on 2026/2/6
 */

package ruleengine

import (
	"path/filepath"
	"testing"

	"northstar/internal/store"
)

func TestEvaluateRules_BaseStatus(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	items, err := EvaluateRules(st, 2025, 12, true)
	if err != nil {
		t.Fatalf("evaluate rules failed: %v", err)
	}
	if len(items) < 10 {
		t.Fatalf("expected >=10 evaluations, got %d", len(items))
	}

	industryFound := false
	priorityFound := false
	for _, item := range items {
		if item.RuleCode == "规则P2-1 行业增速区间与差异约束" {
			industryFound = true
			if item.Status != StatusPass {
				t.Fatalf("industry rule expect pass, got %s", item.Status)
			}
		}
		if item.RuleCode == "规则P2-10 小微与吃穿用优先策略" {
			priorityFound = true
			if item.Status != StatusFail {
				t.Fatalf("priority rule expect fail, got %s", item.Status)
			}
		}
	}
	if !industryFound {
		t.Fatalf("missing industry growth rule evaluation")
	}
	if !priorityFound {
		t.Fatalf("missing priority maximize rule evaluation")
	}
}

func TestEvaluateRules_UnknownExpressionSkipped(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	err = st.UpsertRuleDefinition(store.RuleDefinition{
		RuleCode:       "未知表达式规则",
		Name:           "未知表达式规则",
		Description:    "用于测试缺少上下文变量",
		Expression:     "unknown_metric > 0",
		Severity:       "warn",
		Suggestion:     "补齐变量",
		PreferenceJSON: "{}",
		DisplayOrder:   999,
		Enabled:        true,
	})
	if err != nil {
		t.Fatalf("upsert custom rule failed: %v", err)
	}

	items, err := EvaluateRules(st, 2025, 12, false)
	if err != nil {
		t.Fatalf("evaluate rules failed: %v", err)
	}

	found := false
	for _, item := range items {
		if item.RuleCode == "未知表达式规则" {
			found = true
			if item.Status != StatusSkipped {
				t.Fatalf("expect skipped, got %s", item.Status)
			}
			if item.SkippedReason == "" {
				t.Fatalf("expect skipped reason")
			}
		}
	}
	if !found {
		t.Fatalf("custom rule evaluation not found")
	}
}
