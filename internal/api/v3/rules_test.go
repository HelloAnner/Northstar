/**
 * 约束与自然语言规则接口测试
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
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"northstar/internal/dagcalc"
	"northstar/internal/store"
)

func TestConstraintsCRUD(t *testing.T) {
	handler, router, _ := newRulesTestRouter(t)
	_ = handler

	// Create
	max := 15.0
	createBody := store.AdjustmentConstraint{
		Type:        "clamp_target",
		IndicatorID: "wholesale_month_rate",
		MaxValue:    &max,
	}
	resp := postJSON(t, router, http.MethodPost, "/api/constraints", createBody)
	if resp.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: %d body=%s", resp.Code, resp.Body.String())
	}
	var created store.AdjustmentConstraint
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.ID == 0 || created.IndicatorID != "wholesale_month_rate" {
		t.Fatalf("unexpected created constraint: %+v", created)
	}

	// List
	listResp := postJSON(t, router, http.MethodGet, "/api/constraints", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d", listResp.Code)
	}
	var items []store.AdjustmentConstraint
	if err := json.Unmarshal(listResp.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 constraint, got %d", len(items))
	}

	// Update
	newMax := 10.0
	updateBody := store.AdjustmentConstraint{
		Type:        "clamp_target",
		IndicatorID: "wholesale_month_rate",
		MaxValue:    &newMax,
		Enabled:     true,
	}
	updateResp := postJSON(t, router, http.MethodPut, "/api/constraints/1", updateBody)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("unexpected update status: %d body=%s", updateResp.Code, updateResp.Body.String())
	}

	// Delete
	deleteResp := postJSON(t, router, http.MethodDelete, "/api/constraints/1", nil)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}

	// Verify empty
	listResp2 := postJSON(t, router, http.MethodGet, "/api/constraints", nil)
	var afterDelete []store.AdjustmentConstraint
	if err := json.Unmarshal(listResp2.Body.Bytes(), &afterDelete); err != nil {
		t.Fatalf("decode after delete: %v", err)
	}
	if len(afterDelete) != 0 {
		t.Fatalf("expected 0 constraints after delete, got %d", len(afterDelete))
	}
}

func TestConstraintsValidation(t *testing.T) {
	_, router, _ := newRulesTestRouter(t)

	// Missing indicatorId for clamp_target
	bad := store.AdjustmentConstraint{Type: "clamp_target"}
	resp := postJSON(t, router, http.MethodPost, "/api/constraints", bad)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d body=%s", resp.Code, resp.Body.String())
	}

	// Unknown type
	bad2 := store.AdjustmentConstraint{Type: "unknown"}
	resp2 := postJSON(t, router, http.MethodPost, "/api/constraints", bad2)
	if resp2.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request for unknown type, got %d", resp2.Code)
	}
}

func TestNaturalRulesCRUD(t *testing.T) {
	_, router, _ := newRulesTestRouter(t)

	// Create
	resp := postJSON(t, router, http.MethodPost, "/api/natural-rules", naturalRuleRequest{Text: "零售增速不超过 15%"})
	if resp.Code != http.StatusCreated {
		t.Fatalf("unexpected create status: %d body=%s", resp.Code, resp.Body.String())
	}
	var created store.NaturalRule
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created: %v", err)
	}
	if created.ID == 0 || created.Text != "零售增速不超过 15%" {
		t.Fatalf("unexpected created rule: %+v", created)
	}

	// Create another
	postJSON(t, router, http.MethodPost, "/api/natural-rules", naturalRuleRequest{Text: "批发增速不低于 0%"})

	// List
	listResp := postJSON(t, router, http.MethodGet, "/api/natural-rules", nil)
	if listResp.Code != http.StatusOK {
		t.Fatalf("unexpected list status: %d", listResp.Code)
	}
	var items []store.NaturalRule
	if err := json.Unmarshal(listResp.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(items))
	}

	// Update
	updateResp := postJSON(t, router, http.MethodPut, "/api/natural-rules/1", naturalRuleRequest{Text: "零售增速不超过 10%"})
	if updateResp.Code != http.StatusOK {
		t.Fatalf("unexpected update status: %d body=%s", updateResp.Code, updateResp.Body.String())
	}

	// Delete
	deleteResp := postJSON(t, router, http.MethodDelete, "/api/natural-rules/2", nil)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: %d body=%s", deleteResp.Code, deleteResp.Body.String())
	}

	// Verify
	listResp2 := postJSON(t, router, http.MethodGet, "/api/natural-rules", nil)
	var afterOps []store.NaturalRule
	if err := json.Unmarshal(listResp2.Body.Bytes(), &afterOps); err != nil {
		t.Fatalf("decode after ops: %v", err)
	}
	if len(afterOps) != 1 || afterOps[0].Text != "零售增速不超过 10%" {
		t.Fatalf("unexpected rules after ops: %+v", afterOps)
	}
}

func TestNaturalRulesRejectsEmptyText(t *testing.T) {
	_, router, _ := newRulesTestRouter(t)

	resp := postJSON(t, router, http.MethodPost, "/api/natural-rules", naturalRuleRequest{Text: ""})
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func newRulesTestRouter(t *testing.T) (*Handler, http.Handler, *store.Store) {
	t.Helper()

	gin.SetMode(gin.ReleaseMode)
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	handler := NewHandlerWithEngine(st, "", dagcalc.NewEngine(dagcalc.NewGraph(), st, 2025, 12))
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))
	return handler, router, st
}

func postJSON(t *testing.T, router http.Handler, method string, path string, body any) *httptest.ResponseRecorder {
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
	return resp
}
