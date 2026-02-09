/**
 * 规则校验接口测试
 *
 * @author Anner
 * Created on 2026/2/6
 */

package v3

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"northstar/internal/store"
)

func TestRulesAPI_Evaluate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	defer func() { _ = st.Close() }()

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	req := httptest.NewRequest(http.MethodGet, "/api/rules/evaluate?enabledOnly=false", nil)
	resp := httptest.NewRecorder()
	r.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("evaluate rules status=%d body=%s", resp.Code, resp.Body.String())
	}

	var payload listRuleEvaluationsResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode evaluate response failed: %v", err)
	}
	if len(payload.Items) < 10 {
		t.Fatalf("expected >=10 rule evaluations, got %d", len(payload.Items))
	}

	found := false
	for _, item := range payload.Items {
		if item.RuleCode == "规则P2-1 行业增速区间与差异约束" {
			found = true
			if item.Status == "" {
				t.Fatalf("rule status should not be empty")
			}
		}
	}
	if !found {
		t.Fatalf("missing industry growth evaluation")
	}
}
