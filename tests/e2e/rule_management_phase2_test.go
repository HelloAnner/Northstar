//go:build e2e

/**
 * Rule Management Phase 2 端到端测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	v3 "northstar/internal/api/v3"
	"northstar/internal/config"
	"northstar/internal/dagcalc"
	"northstar/internal/llm"
	"northstar/internal/rules"
	"northstar/internal/server"
	"northstar/internal/store"
)

type fakeE2ERuleClient struct {
	responses []llm.ChatResult
}

func (f *fakeE2ERuleClient) Chat(_ context.Context, _ llm.ChatRequest, _ func(string) error) (llm.ChatResult, error) {
	if len(f.responses) == 0 {
		return llm.ChatResult{Content: `{"version":"1.0","rules":[]}`}, nil
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	return resp, nil
}

type phase2RuleItem struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

type phase2RuleStatus struct {
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
	Error     string `json:"error"`
}

type phase2Env struct {
	baseURL   string
	store     *store.Store
	handler   *v3.Handler
	rolePath  string
	rulesPath string
	engine    *dagcalc.Engine
}

func TestRuleManagementPhase2E2E_ServerInit(t *testing.T) {
	dataDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.Data.DataDir = dataDir
	cfg.Server.DevMode = false

	s := server.NewServer(cfg)
	t.Cleanup(func() {
		_ = s.GetStore().Close()
	})

	rulesPath := filepath.Join(dataDir, "config", "rules.md")
	raw, err := os.ReadFile(rulesPath)
	if err != nil {
		t.Fatalf("read rules.md: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("rules.md should not be empty")
	}

	roleJSONPath := filepath.Join(dataDir, "config", "role.json")
	roleRaw, err := os.ReadFile(roleJSONPath)
	if err != nil {
		t.Fatalf("read role.json: %v", err)
	}
	if len(roleRaw) == 0 {
		t.Fatal("role.json should not be empty")
	}

	testServer := httptest.NewServer(s.RouterForTest())
	t.Cleanup(testServer.Close)

	resp, err := http.Get(testServer.URL + "/api/v1/rules/status")
	if err != nil {
		t.Fatalf("get rules status: %v", err)
	}
	defer resp.Body.Close()

	status := decodeJSON[phase2RuleStatus](t, resp)
	if status.Status != "idle" {
		t.Fatalf("unexpected rules status: %+v", status)
	}

	rulesResp, err := http.Get(testServer.URL + "/api/v1/rules")
	if err != nil {
		t.Fatalf("get rules: %v", err)
	}
	defer rulesResp.Body.Close()
	if rulesResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected rules status code: %d", rulesResp.StatusCode)
	}

	items := decodeJSON[[]phase2RuleItem](t, rulesResp)
	if len(items) != 17 {
		t.Fatalf("expected 17 default rules, got %d", len(items))
	}
}

func TestRuleManagementPhase2E2E_CRUDConvertAndFallback(t *testing.T) {
	env := newPhase2Env(t)
	insertWRRateRow(t, env.store, "W1", "wholesale", 100, 100)

	env.useConverter(&fakeE2ERuleClient{
		responses: []llm.ChatResult{
			{Content: `{"version":"1.0","rules":[{"id":"c1","name":"clamp","type":"clamp_target","indicator":"wholesale_month_rate","max":15}]}`},
		},
	})

	env.postRule(t, http.MethodPost, "/api/rules", map[string]string{"text": "批发当月增速不超过 15%"}, http.StatusCreated)
	waitRuleStatusE2E(t, env, "ok")

	items := env.fetchRules(t)
	if len(items) != 1 || items[0].Text != "批发当月增速不超过 15%" {
		t.Fatalf("unexpected rules: %+v", items)
	}

	resp := env.postOptimize(t, map[string]float64{"wholesale_month_rate": 30})
	rule := requireSingleRule(t, resp.AppliedRules)
	assertRuleType(t, rule.Type, "clamp_target")
	assertFloatEqual(t, env.indicatorValue(t, "wholesale_month_rate"), 15)

	roleBefore, err := os.ReadFile(env.rolePath)
	if err != nil {
		t.Fatalf("read role.json: %v", err)
	}

	env.useConverter(&fakeE2ERuleClient{
		responses: []llm.ChatResult{
			{Content: `{"version":"1.0","rules":[{"id":"c1","type":"clamp_target","indicator":"bad_indicator","max":15}]}`},
			{Content: `{"version":"1.0","rules":[{"id":"c1","type":"clamp_target","indicator":"bad_indicator","max":15}]}`},
			{Content: `{"version":"1.0","rules":[{"id":"c1","type":"clamp_target","indicator":"bad_indicator","max":15}]}`},
		},
	})

	env.postRule(t, http.MethodPut, "/api/rules/1", map[string]string{"text": "批发当月增速不超过 12%"}, http.StatusOK)
	waitRuleStatusE2E(t, env, "error")

	status := env.fetchRuleStatus(t)
	if !strings.Contains(status.Error, "3次重试仍失败") {
		t.Fatalf("unexpected error status: %+v", status)
	}

	roleAfter, err := os.ReadFile(env.rolePath)
	if err != nil {
		t.Fatalf("read role.json after error: %v", err)
	}
	if string(roleAfter) != string(roleBefore) {
		t.Fatalf("expected role.json to remain unchanged")
	}

	resp = env.postOptimize(t, map[string]float64{"wholesale_month_rate": 30})
	rule = requireSingleRule(t, resp.AppliedRules)
	assertRuleType(t, rule.Type, "clamp_target")
	assertFloatEqual(t, env.indicatorValue(t, "wholesale_month_rate"), 15)
}

func newPhase2Env(t *testing.T) *phase2Env {
	t.Helper()

	gin.SetMode(gin.ReleaseMode)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	if err := st.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}
	if err := st.SetConfig("rules_convert_status", "idle"); err != nil {
		t.Fatalf("set rules status: %v", err)
	}
	if err := st.SetConfig("rules_convert_at", ""); err != nil {
		t.Fatalf("set rules updated at: %v", err)
	}
	if err := st.SetConfig("rules_convert_error", ""); err != nil {
		t.Fatalf("set rules error: %v", err)
	}

	rulesPath := filepath.Join(tmpDir, "config", "rules.md")
	rolePath := filepath.Join(tmpDir, "config", "role.json")
	if err := os.MkdirAll(filepath.Dir(rulesPath), 0755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}
	if err := os.WriteFile(rulesPath, []byte("# 调整规则\n\n"), 0644); err != nil {
		t.Fatalf("write rules.md: %v", err)
	}

	engine := dagcalc.NewEngine(dagcalc.NewGraph(), st, 2025, 12, rolePath)
	if err := engine.ReloadRules(); err != nil {
		t.Fatalf("reload rules: %v", err)
	}

	handler := v3.NewHandlerWithEngine(st, "", engine)
	handler.ConfigureRuleManagement(rulesPath, rolePath)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))
	testServer := httptest.NewServer(router)
	t.Cleanup(testServer.Close)

	return &phase2Env{
		baseURL:   testServer.URL,
		store:     st,
		handler:   handler,
		rolePath:  rolePath,
		rulesPath: rulesPath,
		engine:    engine,
	}
}

func (e *phase2Env) useConverter(client rules.ConverterClient) {
	e.handler.SetRuleConverterFactory(func() (v3.RuleConverter, error) {
		return rules.NewConverterWithClient(client, e.store, e.engine, e.rolePath, e.rulesPath), nil
	})
}

func (e *phase2Env) postRule(t *testing.T, method string, path string, body any, expected int) {
	t.Helper()

	var payload []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal rule body: %v", err)
		}
		payload = data
	}
	req, err := http.NewRequest(method, e.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("call rules api: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != expected {
		t.Fatalf("unexpected rules status: %d", resp.StatusCode)
	}
}

func (e *phase2Env) fetchRules(t *testing.T) []phase2RuleItem {
	t.Helper()

	resp, err := http.Get(e.baseURL + "/api/rules")
	if err != nil {
		t.Fatalf("get rules: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected get rules status: %d", resp.StatusCode)
	}
	return decodeJSON[[]phase2RuleItem](t, resp)
}

func (e *phase2Env) fetchRuleStatus(t *testing.T) phase2RuleStatus {
	t.Helper()

	resp, err := http.Get(e.baseURL + "/api/rules/status")
	if err != nil {
		t.Fatalf("get rule status: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected rules status code: %d", resp.StatusCode)
	}
	return decodeJSON[phase2RuleStatus](t, resp)
}

func (e *phase2Env) postOptimize(t *testing.T, targets map[string]float64) optimizeResult {
	t.Helper()

	body, err := json.Marshal(optimizePayload{Targets: targets})
	if err != nil {
		t.Fatalf("marshal optimize payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, e.baseURL+"/api/optimize", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build optimize request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("call optimize: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected optimize status: %d", resp.StatusCode)
	}
	return decodeJSON[optimizeResult](t, resp)
}

func (e *phase2Env) indicatorValue(t *testing.T, id string) float64 {
	t.Helper()

	resp, err := http.Get(e.baseURL + "/api/indicators")
	if err != nil {
		t.Fatalf("get indicators: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected indicators status: %d", resp.StatusCode)
	}

	result := decodeJSON[indicatorsResult](t, resp)
	for _, group := range result.Groups {
		for _, indicator := range group.Indicators {
			if indicator.ID == id {
				return indicator.Value
			}
		}
	}
	t.Fatalf("indicator not found: %s", id)
	return 0
}

func waitRuleStatusE2E(t *testing.T, env *phase2Env, expected string) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		status := env.fetchRuleStatus(t)
		if status.Status == expected {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	status := env.fetchRuleStatus(t)
	t.Fatalf("unexpected final rules status: %+v", status)
}
