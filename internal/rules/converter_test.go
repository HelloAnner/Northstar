/**
 * 规则转换测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

package rules

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"northstar/internal/llm"
	"northstar/internal/store"
)

type fakeRuleChatClient struct {
	responses []llm.ChatResult
	err       error
	requests  []llm.ChatRequest
}

func (f *fakeRuleChatClient) Chat(_ context.Context, req llm.ChatRequest, _ func(string) error) (llm.ChatResult, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return llm.ChatResult{}, f.err
	}
	if len(f.responses) == 0 {
		return llm.ChatResult{}, nil
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	return resp, nil
}

type fakeRuleReloader struct {
	reloadCount int
}

func (f *fakeRuleReloader) ReloadRules() error {
	f.reloadCount++
	return nil
}

func TestExtractJSONPrefersJSONCodeBlock(t *testing.T) {
	content := "说明文字\n```json\n{\"version\":\"1.0\",\"rules\":[]}\n```\n补充说明"

	got, err := extractJSON(content)
	if err != nil {
		t.Fatalf("extract json: %v", err)
	}
	if got != "{\"version\":\"1.0\",\"rules\":[]}" {
		t.Fatalf("unexpected json: %s", got)
	}
}

func TestExtractJSONAcceptsPlainJSON(t *testing.T) {
	got, err := extractJSON("{\"version\":\"1.0\",\"rules\":[]}")
	if err != nil {
		t.Fatalf("extract json: %v", err)
	}
	if got != "{\"version\":\"1.0\",\"rules\":[]}" {
		t.Fatalf("unexpected json: %s", got)
	}
}

func TestValidateRoleJSONReturnsStructuredErrors(t *testing.T) {
	jsonStr := `{
  "version": "1.0",
  "rules": [
    {"id":"c1","type":"clamp_target","indicator":"bad_indicator"},
    {"id":"f1","type":"filter_allocation","indicator":"retail_month_rate","filter":"bad_filter"},
    {"id":"p1","type":"compensate","trigger":"retail_month_rate","ensure":"bad_indicator","relation":"bad_relation"},
    {"id":"x1","type":"unknown"}
  ]
}`

	errs := validateRoleJSON(jsonStr)
	if len(errs) != 6 {
		t.Fatalf("unexpected error count: %d %+v", len(errs), errs)
	}
	if errs[0].RuleID == "" || errs[0].Field == "" || errs[0].Message == "" {
		t.Fatalf("expected structured validation error: %+v", errs[0])
	}
}

func TestConvertRetriesWithValidationFeedback(t *testing.T) {
	client := &fakeRuleChatClient{
		responses: []llm.ChatResult{
			{Content: "{\"version\":\"1.0\",\"rules\":[{\"id\":\"c1\",\"type\":\"clamp_target\",\"indicator\":\"bad\",\"min\":1}]}"},
			{Content: "{\"version\":\"1.0\",\"rules\":[{\"id\":\"c1\",\"type\":\"clamp_target\",\"indicator\":\"retail_month_rate\",\"min\":1}]}"},
		},
	}
	converter := &Converter{llm: client}

	jsonStr, err := converter.convert("# 调整规则\n\n1. 零售增速不低于 1%")
	if err != nil {
		t.Fatalf("convert failed: %v", err)
	}
	if !strings.Contains(jsonStr, "retail_month_rate") {
		t.Fatalf("unexpected converted json: %s", jsonStr)
	}
	if len(client.requests) != 2 {
		t.Fatalf("expected 2 llm requests, got %d", len(client.requests))
	}
	lastReq := client.requests[1]
	if len(lastReq.Messages) < 3 {
		t.Fatalf("expected retry context to contain validation feedback: %+v", lastReq.Messages)
	}
	if !strings.Contains(lastReq.Messages[len(lastReq.Messages)-1].Content, "校验失败") {
		t.Fatalf("expected validation feedback in retry prompt: %+v", lastReq.Messages)
	}
}

func TestConvertAsyncWritesRoleFileAndStatus(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	rolePath := filepath.Join(t.TempDir(), "config", "role.json")
	reloader := &fakeRuleReloader{}
	converter := &Converter{
		llm: &fakeRuleChatClient{
			responses: []llm.ChatResult{
				{Content: "{\"version\":\"1.0\",\"rules\":[{\"id\":\"c1\",\"type\":\"clamp_target\",\"indicator\":\"retail_month_rate\",\"max\":15}]}"},
			},
		},
		rolePath: rolePath,
		mdPath:   filepath.Join(t.TempDir(), "config", "rules.md"),
		store:    st,
		engine:   reloader,
	}

	converter.ConvertAsync("# 调整规则\n\n1. 零售增速不高于 15%")

	waitForRuleStatus(t, st, "ok")
	data, err := os.ReadFile(rolePath)
	if err != nil {
		t.Fatalf("read role.json: %v", err)
	}
	if !strings.Contains(string(data), "retail_month_rate") {
		t.Fatalf("unexpected role.json: %s", string(data))
	}
	if reloader.reloadCount != 1 {
		t.Fatalf("expected reload rules once, got %d", reloader.reloadCount)
	}
}

func waitForRuleStatus(t *testing.T, st *store.Store, expected string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		value, err := st.GetConfig("rules_convert_status")
		if err == nil && value == expected {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	value, err := st.GetConfig("rules_convert_status")
	if err != nil {
		t.Fatalf("load rules status: %v", err)
	}
	t.Fatalf("unexpected rules status: %s", value)
}
