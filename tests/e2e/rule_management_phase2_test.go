//go:build e2e

/**
 * Rule Management Phase 2 端到端测试
 *
 * 测试硬约束 CRUD 和自然语言规则 CRUD，以及约束生效后的 optimize 验证。
 *
 * @author Anner
 * Created on 2026/3/14
 */

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	v3 "northstar/internal/api/v3"
	"northstar/internal/config"
	"northstar/internal/dagcalc"
	"northstar/internal/server"
	"northstar/internal/store"
)

type phase2Env struct {
	baseURL string
	store   *store.Store
	handler *v3.Handler
	engine  *dagcalc.Engine
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

	testServer := httptest.NewServer(s.RouterForTest())
	t.Cleanup(testServer.Close)

	// 验证 constraints API 可用
	resp, err := http.Get(testServer.URL + "/api/v1/constraints")
	if err != nil {
		t.Fatalf("get constraints: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected constraints status: %d", resp.StatusCode)
	}

	var items []store.AdjustmentConstraint
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("decode constraints: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 default constraints, got %d", len(items))
	}

	// 验证 natural-rules API 可用
	rulesResp, err := http.Get(testServer.URL + "/api/v1/natural-rules")
	if err != nil {
		t.Fatalf("get natural-rules: %v", err)
	}
	defer rulesResp.Body.Close()
	if rulesResp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected natural-rules status: %d", rulesResp.StatusCode)
	}
}

func TestRuleManagementPhase2E2E_ConstraintCRUDAndOptimize(t *testing.T) {
	env := newPhase2Env(t)
	insertWRRateRow(t, env.store, "W1", "wholesale", 100, 100)

	// 创建约束：批发当月增速不超过 15%
	max := 15.0
	created := env.createConstraint(t, store.AdjustmentConstraint{
		Type:        "clamp_target",
		IndicatorID: "wholesale_month_rate",
		MaxValue:    &max,
	})
	if created.ID == 0 {
		t.Fatalf("expected non-zero constraint ID")
	}

	// 验证 optimize 生效
	resp := env.postOptimize(t, map[string]float64{"wholesale_month_rate": 30})
	rule := requireSingleRule(t, resp.AppliedRules)
	assertRuleType(t, rule.Type, "clamp_target")
	assertFloatEqual(t, env.indicatorValue(t, "wholesale_month_rate"), 15)

	// 更新约束：改为不超过 10%
	newMax := 10.0
	env.updateConstraint(t, created.ID, store.AdjustmentConstraint{
		Type:        "clamp_target",
		IndicatorID: "wholesale_month_rate",
		MaxValue:    &newMax,
		Enabled:     true,
	})

	resp = env.postOptimize(t, map[string]float64{"wholesale_month_rate": 30})
	rule = requireSingleRule(t, resp.AppliedRules)
	assertFloatEqual(t, env.indicatorValue(t, "wholesale_month_rate"), 10)

	// 删除约束后不再限制
	env.deleteConstraint(t, created.ID)
	resp = env.postOptimize(t, map[string]float64{"wholesale_month_rate": 30})
	if len(resp.AppliedRules) != 0 {
		t.Fatalf("expected no applied rules after delete, got %+v", resp.AppliedRules)
	}
	assertFloatEqual(t, env.indicatorValue(t, "wholesale_month_rate"), 30)
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

	engine := dagcalc.NewEngine(dagcalc.NewGraph(), st, 2025, 12)
	if err := engine.ReloadRules(); err != nil {
		t.Fatalf("reload rules: %v", err)
	}

	handler := v3.NewHandlerWithEngine(st, "", engine)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))
	testServer := httptest.NewServer(router)
	t.Cleanup(testServer.Close)

	return &phase2Env{
		baseURL: testServer.URL,
		store:   st,
		handler: handler,
		engine:  engine,
	}
}

func (e *phase2Env) createConstraint(t *testing.T, c store.AdjustmentConstraint) store.AdjustmentConstraint {
	t.Helper()

	body, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal constraint: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, e.baseURL+"/api/constraints", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("create constraint: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("unexpected create status: %d", resp.StatusCode)
	}
	return decodeJSON[store.AdjustmentConstraint](t, resp)
}

func (e *phase2Env) updateConstraint(t *testing.T, id int64, c store.AdjustmentConstraint) {
	t.Helper()

	body, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("marshal constraint: %v", err)
	}
	url := fmt.Sprintf("%s/api/constraints/%d", e.baseURL, id)
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update constraint: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected update status: %d", resp.StatusCode)
	}
}

func (e *phase2Env) deleteConstraint(t *testing.T, id int64) {
	t.Helper()

	url := fmt.Sprintf("%s/api/constraints/%d", e.baseURL, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("delete constraint: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected delete status: %d", resp.StatusCode)
	}
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

