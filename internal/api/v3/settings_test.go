/**
 * AI 偏好设置接口测试
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
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"northstar/internal/dagcalc"
	"northstar/internal/store"
)

func TestUserPromptRoutes(t *testing.T) {
	handler, router, st := newSettingsTestRouter(t)
	_ = handler

	if err := st.SetConfig("llm_user_prompt", ""); err != nil {
		t.Fatalf("set config: %v", err)
	}

	resp := performJSONRequest(t, router, http.MethodGet, "/api/settings/user-prompt", nil)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected get status: %d", resp.Code)
	}
	var getBody userPromptResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &getBody); err != nil {
		t.Fatalf("decode get response: %v", err)
	}
	if getBody.Content != "" {
		t.Fatalf("expected empty content, got %+v", getBody)
	}

	saveBody := userPromptRequest{Content: "回答简洁，先写结论。"}
	resp = performJSONRequest(t, router, http.MethodPut, "/api/settings/user-prompt", saveBody)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected put status: %d body=%s", resp.Code, resp.Body.String())
	}

	value, err := st.GetConfig("llm_user_prompt")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if value != saveBody.Content {
		t.Fatalf("unexpected stored value: %s", value)
	}
}

func TestUserPromptRoutesRejectContentTooLong(t *testing.T) {
	_, router, _ := newSettingsTestRouter(t)
	tooLong := userPromptRequest{Content: strings.Repeat("长", 501)}

	resp := performJSONRequest(t, router, http.MethodPut, "/api/settings/user-prompt", tooLong)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected bad request, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func newSettingsTestRouter(t *testing.T) (*Handler, http.Handler, *store.Store) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	handler := NewHandlerWithEngine(st, "", dagcalc.NewEngine(dagcalc.NewGraph(), st, 2025, 12, ""))
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))
	return handler, router, st
}

func performJSONRequest(t *testing.T, router http.Handler, method string, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var payload []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request: %v", err)
		}
		payload = data
	}

	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}
