/**
 * 规则管理接口测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

package v3

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"northstar/internal/dagcalc"
	"northstar/internal/store"
)

type fakeAsyncRuleConverter struct {
	contents []string
}

func (f *fakeAsyncRuleConverter) ConvertAsync(mdContent string) {
	f.contents = append(f.contents, mdContent)
}

func TestRulesFileRepoReadWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rules.md")
	repo := &rulesFileRepo{path: path}

	if err := repo.WriteRules([]RuleItem{
		{Index: 1, Text: "零售增速不超过 15%"},
		{Index: 2, Text: "批发增速不低于 0%"},
	}); err != nil {
		t.Fatalf("write rules: %v", err)
	}

	items, err := repo.ReadRules()
	if err != nil {
		t.Fatalf("read rules: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("unexpected rule count: %d", len(items))
	}
	if items[0].Text != "零售增速不超过 15%" || items[1].Index != 2 {
		t.Fatalf("unexpected rules: %+v", items)
	}
}

func TestRulesCRUDAndStatusRoutes(t *testing.T) {
	handler, repo, converter := newRulesTestHandler(t)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	postRule(t, router, http.MethodPost, "/api/rules", map[string]string{"text": "零售增速不超过 15%"}, http.StatusCreated)
	postRule(t, router, http.MethodPost, "/api/rules", map[string]string{"text": "批发增速不低于 0%"}, http.StatusCreated)

	items := fetchRules(t, router)
	if len(items) != 2 {
		t.Fatalf("unexpected rule count: %d", len(items))
	}
	if len(converter.contents) != 2 {
		t.Fatalf("expected convert async to be triggered twice, got %d", len(converter.contents))
	}

	postRule(t, router, http.MethodPut, "/api/rules/1", map[string]string{"text": "零售增速不超过 10%"}, http.StatusOK)
	items = fetchRules(t, router)
	if items[0].Text != "零售增速不超过 10%" {
		t.Fatalf("unexpected updated rule: %+v", items[0])
	}

	postRule(t, router, http.MethodDelete, "/api/rules/2", nil, http.StatusOK)
	items = fetchRules(t, router)
	if len(items) != 1 || items[0].Index != 1 {
		t.Fatalf("unexpected rules after delete: %+v", items)
	}

	raw, err := os.ReadFile(repo.path)
	if err != nil {
		t.Fatalf("read rules file: %v", err)
	}
	expected := "# 调整规则\n\n1. 零售增速不超过 10%\n"
	if string(raw) != expected {
		t.Fatalf("unexpected rules file:\n%s", string(raw))
	}

	status := fetchRulesStatus(t, router)
	if status.Status != "idle" {
		t.Fatalf("unexpected rules status: %+v", status)
	}
}

func TestRulesConvertRouteReturnsRunning(t *testing.T) {
	handler, _, converter := newRulesTestHandler(t)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))

	postRule(t, router, http.MethodPost, "/api/rules", map[string]string{"text": "零售增速不超过 15%"}, http.StatusCreated)
	postRule(t, router, http.MethodPost, "/api/rules/convert", nil, http.StatusOK)
	if len(converter.contents) != 2 {
		t.Fatalf("expected manual convert trigger, got %d", len(converter.contents))
	}
}

func newRulesTestHandler(t *testing.T) (*Handler, *rulesFileRepo, *fakeAsyncRuleConverter) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	repo := &rulesFileRepo{path: filepath.Join(t.TempDir(), "rules.md")}
	converter := &fakeAsyncRuleConverter{}
	handler := NewHandlerWithEngine(st, "", dagcalc.NewEngine(dagcalc.NewGraph(), st, 2025, 12, ""))
	handler.rulesRepo = repo
	handler.ruleConverterFactory = func() (RuleConverter, error) {
		return converter, nil
	}
	if err := st.SetConfig("rules_convert_status", "idle"); err != nil {
		t.Fatalf("set default status: %v", err)
	}
	return handler, repo, converter
}

func postRule(t *testing.T, router http.Handler, method string, path string, body any, expected int) {
	t.Helper()

	var reqBody []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reqBody = data
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != expected {
		t.Fatalf("unexpected status %d body=%s", resp.Code, resp.Body.String())
	}
}

func fetchRules(t *testing.T, router http.Handler) []RuleItem {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/rules", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected rules status %d body=%s", resp.Code, resp.Body.String())
	}
	var items []RuleItem
	if err := json.Unmarshal(resp.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode rules: %v", err)
	}
	return items
}

type testRulesStatusResponse struct {
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
	Error     string `json:"error"`
}

func fetchRulesStatus(t *testing.T, router http.Handler) testRulesStatusResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/rules/status", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status response %d body=%s", resp.Code, resp.Body.String())
	}
	var status testRulesStatusResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &status); err != nil {
		t.Fatalf("decode status: %v", err)
	}
	return status
}
