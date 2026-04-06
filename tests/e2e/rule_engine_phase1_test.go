//go:build e2e

/**
 * Rule Engine Phase 1 端到端测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	v3 "northstar/internal/api/v3"
	"northstar/internal/dagcalc"
	"northstar/internal/store"
)

type phase1Env struct {
	baseURL string
	store   *store.Store
}

type optimizePayload struct {
	Targets map[string]float64 `json:"targets"`
}

type optimizeResult struct {
	AppliedRules []appliedRule `json:"appliedRules"`
}

type appliedRule struct {
	Type        string  `json:"type"`
	IndicatorID string  `json:"indicatorId"`
	TriggerID   string  `json:"triggerId"`
	EnsureID    string  `json:"ensureId"`
	BeforeValue float64 `json:"beforeValue"`
	AfterValue  float64 `json:"afterValue"`
	BeforeCount int     `json:"beforeCount"`
	AfterCount  int     `json:"afterCount"`
}

type indicatorsResult struct {
	Groups []indicatorGroup `json:"groups"`
}

type indicatorGroup struct {
	Indicators []indicatorValue `json:"indicators"`
}

type indicatorValue struct {
	ID    string  `json:"id"`
	Value float64 `json:"value"`
}

func TestRuleEnginePhase1E2E_ClampTarget(t *testing.T) {
	env := newPhase1Env(t, clampConstraint("wholesale_month_rate", nil, float64Ptr(15)))
	insertWRRateRow(t, env.store, "W1", "wholesale", 100, 100)

	resp := env.postOptimize(t, map[string]float64{"wholesale_month_rate": 20})
	rule := requireSingleRule(t, resp.AppliedRules)
	assertRuleType(t, rule.Type, "clamp_target")
	assertFloatEqual(t, rule.BeforeValue, 20)
	assertFloatEqual(t, rule.AfterValue, 15)
	assertFloatEqual(t, env.indicatorValue(t, "wholesale_month_rate"), 15)
}

func TestRuleEnginePhase1E2E_FilterAllocation(t *testing.T) {
	env := newPhase1Env(t, filterConstraint("retail_month_rate", "positive_current"))
	insertWRRateRow(t, env.store, "R1", "retail", 110, 100)
	insertWRRateRow(t, env.store, "R2", "retail", 80, 100)

	resp := env.postOptimize(t, map[string]float64{"retail_month_rate": 10})
	rule := requireSingleRule(t, resp.AppliedRules)
	assertRuleType(t, rule.Type, "filter_allocation")
	if rule.BeforeCount != 2 || rule.AfterCount != 1 {
		t.Fatalf("unexpected filter counts: %+v", rule)
	}
	assertFloatEqual(t, queryCurrentSales(t, env.store, "R2", "retail"), 80)
}

func TestRuleEnginePhase1E2E_Compensate(t *testing.T) {
	env := newPhase1Env(t, compensateConstraint("retail_month_rate", "wholesale_month_rate", "gte"))
	insertWRRateRow(t, env.store, "W1", "wholesale", 100, 100)
	insertWRRateRow(t, env.store, "R1", "retail", 100, 100)

	resp := env.postOptimize(t, map[string]float64{"retail_month_rate": 30})
	if !containsCompensateRule(resp.AppliedRules) {
		t.Fatalf("expected compensate rule, got: %+v", resp.AppliedRules)
	}
	retail := env.indicatorValue(t, "retail_month_rate")
	wholesale := env.indicatorValue(t, "wholesale_month_rate")
	if wholesale < retail {
		t.Fatalf("expected wholesale >= retail, got wholesale=%.2f retail=%.2f", wholesale, retail)
	}
}

func TestRuleEnginePhase1E2E_NoConstraintsFallsBack(t *testing.T) {
	env := newPhase1Env(t)
	insertWRRateRow(t, env.store, "W1", "wholesale", 100, 100)

	resp := env.postOptimize(t, map[string]float64{"wholesale_month_rate": 10})
	if len(resp.AppliedRules) != 0 {
		t.Fatalf("expected empty applied rules, got: %+v", resp.AppliedRules)
	}
	assertFloatEqual(t, env.indicatorValue(t, "wholesale_month_rate"), 10)
}

func newPhase1Env(t *testing.T, constraints ...store.AdjustmentConstraint) *phase1Env {
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

	for _, c := range constraints {
		if _, err := st.CreateAdjustmentConstraint(c); err != nil {
			t.Fatalf("create constraint: %v", err)
		}
	}

	engine := dagcalc.NewEngine(dagcalc.NewGraph(), st, 2025, 12)
	if err := engine.ReloadRules(); err != nil {
		t.Fatalf("reload rules: %v", err)
	}

	router := gin.New()
	v3.NewHandlerWithEngine(st, "", engine).RegisterRoutes(router.Group("/api"))
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &phase1Env{baseURL: server.URL, store: st}
}

func float64Ptr(v float64) *float64 { return &v }

func clampConstraint(indicator string, min *float64, max *float64) store.AdjustmentConstraint {
	return store.AdjustmentConstraint{
		Type:        "clamp_target",
		IndicatorID: indicator,
		MinValue:    min,
		MaxValue:    max,
		Enabled:     true,
	}
}

func filterConstraint(indicator string, filter string) store.AdjustmentConstraint {
	return store.AdjustmentConstraint{
		Type:        "filter_allocation",
		IndicatorID: indicator,
		FilterMode:  filter,
		Enabled:     true,
	}
}

func compensateConstraint(trigger string, ensure string, relation string) store.AdjustmentConstraint {
	return store.AdjustmentConstraint{
		Type:      "compensate",
		TriggerID: trigger,
		EnsureID:  ensure,
		Relation:  relation,
		Enabled:   true,
	}
}

func insertWRRateRow(t *testing.T, st *store.Store, code string, industry string, current float64, lastYear float64) {
	t.Helper()

	err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			sales_current_month, sales_last_year_month,
			sales_current_cumulative, sales_last_year_cumulative,
			retail_current_month, retail_last_year_month,
			retail_current_cumulative, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, code, code, "5101", industry, 1, 1, 2025, 12, current, lastYear, current, lastYear, current, lastYear, current, lastYear, "test", "test.xlsx")
	if err != nil {
		t.Fatalf("insert wr row: %v", err)
	}
}

func queryCurrentSales(t *testing.T, st *store.Store, code string, industry string) float64 {
	t.Helper()

	var value float64
	err := st.QueryRow(
		"SELECT sales_current_month FROM wholesale_retail WHERE credit_code = ? AND industry_type = ?",
		code,
		industry,
	).Scan(&value)
	if err != nil {
		t.Fatalf("query current sales: %v", err)
	}
	return value
}

func (e *phase1Env) postOptimize(t *testing.T, targets map[string]float64) optimizeResult {
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

func (e *phase1Env) indicatorValue(t *testing.T, id string) float64 {
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

func decodeJSON[T any](t *testing.T, resp *http.Response) T {
	t.Helper()

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	return result
}

func requireSingleRule(t *testing.T, rules []appliedRule) appliedRule {
	t.Helper()

	if len(rules) != 1 {
		t.Fatalf("expected single applied rule, got: %+v", rules)
	}
	return rules[0]
}

func containsCompensateRule(rules []appliedRule) bool {
	for _, rule := range rules {
		if rule.Type == "compensate" {
			return true
		}
	}
	return false
}

func assertRuleType(t *testing.T, got string, want string) {
	t.Helper()

	if got != want {
		t.Fatalf("unexpected rule type: got=%s want=%s", got, want)
	}
}

func assertFloatEqual(t *testing.T, got float64, want float64) {
	t.Helper()

	if got != want {
		t.Fatalf("unexpected value: got=%.2f want=%.2f", got, want)
	}
}
