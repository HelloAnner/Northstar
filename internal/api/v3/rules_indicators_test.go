package v3

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"northstar/internal/store"
)

func TestIndicatorDefinitionsAPI_ListAndUpsert(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	listReq := httptest.NewRequest(http.MethodGet, "/api/indicator-definitions?enabledOnly=false", nil)
	listResp := httptest.NewRecorder()
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list indicator status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	var before listIndicatorDefinitionsResponse
	if err := json.Unmarshal(listResp.Body.Bytes(), &before); err != nil {
		t.Fatalf("decode list indicator: %v", err)
	}
	if len(before.Items) < 16 {
		t.Fatalf("expected seeded indicators, got=%d", len(before.Items))
	}

	payload := map[string]interface{}{
		"name":         "测试指标",
		"groupCode":    "custom",
		"groupName":    "自定义",
		"groupOrder":   9,
		"description":  "用于测试新增指标接口",
		"formula":      "percent_diff(限上社零额_当月值, 限上社零额_累计值)",
		"unit":         "%",
		"floatMin":     -9,
		"floatMax":     9,
		"displayOrder": 999,
		"enabled":      true,
	}
	body, _ := json.Marshal(payload)
	upsertReq := httptest.NewRequest(http.MethodPatch, "/api/indicator-definitions/测试指标", bytes.NewReader(body))
	upsertReq.Header.Set("Content-Type", "application/json")
	upsertResp := httptest.NewRecorder()
	r.ServeHTTP(upsertResp, upsertReq)
	if upsertResp.Code != http.StatusOK {
		t.Fatalf("upsert indicator status=%d body=%s", upsertResp.Code, upsertResp.Body.String())
	}

	defs, err := st.ListIndicatorDefinitions(false)
	if err != nil {
		t.Fatalf("list indicator from store: %v", err)
	}
	found := false
	for _, def := range defs {
		if def.Code == "测试指标" {
			found = true
			if def.FloatMin != -9 || def.FloatMax != 9 {
				t.Fatalf("unexpected indicator range: %.2f ~ %.2f", def.FloatMin, def.FloatMax)
			}
		}
	}
	if !found {
		t.Fatalf("upserted indicator not found")
	}
}

func TestRulesAPI_ListAndUpsert(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	listReq := httptest.NewRequest(http.MethodGet, "/api/rules?enabledOnly=false", nil)
	listResp := httptest.NewRecorder()
	r.ServeHTTP(listResp, listReq)
	if listResp.Code != http.StatusOK {
		t.Fatalf("list rules status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	var before listRulesResponse
	if err := json.Unmarshal(listResp.Body.Bytes(), &before); err != nil {
		t.Fatalf("decode list rules: %v", err)
	}
	if len(before.Items) < 10 {
		t.Fatalf("expected seeded rules, got=%d", len(before.Items))
	}

	payload := map[string]interface{}{
		"name":           "测试规则",
		"description":    "用于测试规则编辑接口",
		"expression":     "test_value > 0",
		"severity":       "warn",
		"suggestion":     "检查测试规则输入",
		"preferenceJson": "{\"threshold\":1}",
		"displayOrder":   999,
		"enabled":        true,
		"links": []map[string]interface{}{
			{
				"indicatorCode": "小微企业增速_当月",
				"relationLabel": "测试联动",
				"weight":        0.8,
				"displayOrder":  10,
			},
		},
	}
	body, _ := json.Marshal(payload)
	upsertReq := httptest.NewRequest(http.MethodPatch, "/api/rules/测试规则", bytes.NewReader(body))
	upsertReq.Header.Set("Content-Type", "application/json")
	upsertResp := httptest.NewRecorder()
	r.ServeHTTP(upsertResp, upsertReq)
	if upsertResp.Code != http.StatusOK {
		t.Fatalf("upsert rule status=%d body=%s", upsertResp.Code, upsertResp.Body.String())
	}

	rules, err := st.ListRuleDefinitions(false)
	if err != nil {
		t.Fatalf("list rule from store: %v", err)
	}
	foundRule := false
	for _, rule := range rules {
		if rule.RuleCode == "测试规则" {
			foundRule = true
			if rule.Name != "测试规则" {
				t.Fatalf("unexpected rule name: %s", rule.Name)
			}
		}
	}
	if !foundRule {
		t.Fatalf("upserted rule not found")
	}

	links, err := st.ListRuleIndicatorLinks("测试规则")
	if err != nil {
		t.Fatalf("list rule links: %v", err)
	}
	if len(links) != 1 {
		t.Fatalf("expected 1 rule link, got %d", len(links))
	}
	if links[0].IndicatorCode != "小微企业增速_当月" || links[0].Weight != 0.8 {
		t.Fatalf("unexpected rule link: %+v", links[0])
	}
}

func TestRulesAPI_RejectAssignmentExpression(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	payload := map[string]interface{}{
		"name":           "错误规则",
		"description":    "赋值型表达式应被拒绝",
		"expression":     "a = b + c",
		"severity":       "warn",
		"suggestion":     "请改成比较表达式",
		"preferenceJson": "{}",
		"displayOrder":   1,
		"enabled":        true,
		"links":          []map[string]interface{}{},
	}
	body, _ := json.Marshal(payload)
	upsertReq := httptest.NewRequest(http.MethodPatch, "/api/rules/rule_assign_test", bytes.NewReader(body))
	upsertReq.Header.Set("Content-Type", "application/json")
	upsertResp := httptest.NewRecorder()
	r.ServeHTTP(upsertResp, upsertReq)
	if upsertResp.Code != http.StatusBadRequest {
		t.Fatalf("expect bad request, got=%d body=%s", upsertResp.Code, upsertResp.Body.String())
	}
}
