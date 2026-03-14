/**
 * AI 对话 Phase 3 接口测试
 *
 * @author Anner
 * Created on 2026/3/14
 */
package v3

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

	"github.com/gin-gonic/gin"

	"northstar/internal/dagcalc"
	"northstar/internal/llm"
	"northstar/internal/store"
)

type scriptedLLMClient struct {
	result  llm.ChatResult
	err     error
	chunks  []string
	lastReq llm.ChatRequest
}

func (c *scriptedLLMClient) Chat(_ context.Context, req llm.ChatRequest, stream func(string) error) (llm.ChatResult, error) {
	c.lastReq = req
	for _, chunk := range c.chunks {
		if stream != nil {
			if err := stream(chunk); err != nil {
				return llm.ChatResult{}, err
			}
		}
	}
	return c.result, c.err
}

type scriptedClientFactory struct {
	prompts []string
	clients []LLMChatClient
}

func (f *scriptedClientFactory) newClient(prompt string) (LLMChatClient, error) {
	f.prompts = append(f.prompts, prompt)
	if len(f.clients) == 0 {
		return nil, nil
	}
	client := f.clients[0]
	f.clients = f.clients[1:]
	return client, nil
}

func TestLLMChatPhase3ChatModeIncludesUserPrompt(t *testing.T) {
	handler, router, _ := newLLMChatPhase3Router(t, "")
	client := &scriptedLLMClient{
		chunks: []string{"当前", "批发增速偏高"},
		result: llm.ChatResult{Content: "当前批发增速偏高"},
	}
	factory := &scriptedClientFactory{clients: []LLMChatClient{client}}
	handler.SetLLMClientFactory(factory.newClient)

	if err := handler.store.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}
	if err := handler.store.SetConfig("llm_user_prompt", "请先给结论，再解释。"); err != nil {
		t.Fatalf("set user prompt: %v", err)
	}

	resp := performSSERequest(t, router, llmChatRequest{
		Mode: "chat",
		Messages: []llmChatMessage{
			{Role: "user", Content: "解释当前批发增速"},
		},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", resp.Code, resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), `"type":"message_delta"`) {
		t.Fatalf("expected stream delta, got %s", resp.Body.String())
	}
	if !strings.Contains(resp.Body.String(), `"mode":"chat"`) {
		t.Fatalf("expected final mode=chat, got %s", resp.Body.String())
	}
	if len(factory.prompts) != 1 || !strings.Contains(factory.prompts[0], "请先给结论，再解释。") {
		t.Fatalf("expected prompt to include user prompt, got %+v", factory.prompts)
	}
}

func TestLLMChatPhase3AdjustModeRunsOptimize(t *testing.T) {
	rolePath := filepath.Join(t.TempDir(), "role.json")
	if err := os.WriteFile(rolePath, []byte(`{
  "version": "1.0",
  "rules": [
    {"id":"c1","name":"limit","type":"clamp_target","indicator":"wholesale_month_rate","max":10}
  ]
}`), 0644); err != nil {
		t.Fatalf("write role.json: %v", err)
	}

	handler, router, st := newLLMChatPhase3Router(t, rolePath)
	insertChatPhase3WRRow(t, st)

	intentClient := &scriptedLLMClient{
		result: llm.ChatResult{
			Content: `{"actions":[{"type":"set_target","indicatorId":"wholesale_month_rate","value":15}]}`,
		},
	}
	summaryClient := &scriptedLLMClient{
		chunks: []string{"已完成调整"},
		result: llm.ChatResult{Content: "已完成调整"},
	}
	factory := &scriptedClientFactory{clients: []LLMChatClient{intentClient, summaryClient}}
	handler.SetLLMClientFactory(factory.newClient)

	resp := performSSERequest(t, router, llmChatRequest{
		Mode: "adjust",
		Messages: []llmChatMessage{
			{Role: "user", Content: "把批发当月增速调到 15%"},
		},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", resp.Code, resp.Body.String())
	}

	result := readResultEvent(t, resp.Body.String())
	if result.Mode != "adjust" {
		t.Fatalf("unexpected mode: %+v", result)
	}
	if len(result.AppliedRules) == 0 || result.AppliedRules[0].Type != "clamp_target" {
		t.Fatalf("expected clamp applied rule, got %+v", result.AppliedRules)
	}
	value := findChatPhase3IndicatorValue(t, st, "wholesale_month_rate")
	if value != 10 {
		t.Fatalf("expected clamped indicator value 10, got %.2f", value)
	}
}

func TestLLMChatPhase3AdjustModeFallsBackToChat(t *testing.T) {
	handler, router, st := newLLMChatPhase3Router(t, "")
	intentClient := &scriptedLLMClient{
		result: llm.ChatResult{Content: `{"actions":[]}`},
	}
	chatClient := &scriptedLLMClient{
		chunks: []string{"这是咨询回复"},
		result: llm.ChatResult{Content: "这是咨询回复"},
	}
	factory := &scriptedClientFactory{clients: []LLMChatClient{intentClient, chatClient}}
	handler.SetLLMClientFactory(factory.newClient)

	before := findChatPhase3IndicatorValue(t, st, "limitAbove_month_rate")
	resp := performSSERequest(t, router, llmChatRequest{
		Mode: "adjust",
		Messages: []llmChatMessage{
			{Role: "user", Content: "当前零售增速为什么偏低？"},
		},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", resp.Code, resp.Body.String())
	}

	result := readResultEvent(t, resp.Body.String())
	if result.Mode != "chat" {
		t.Fatalf("expected fallback chat mode, got %+v", result)
	}
	if len(result.AppliedRules) != 0 {
		t.Fatalf("expected no applied rules, got %+v", result.AppliedRules)
	}
	after := findChatPhase3IndicatorValue(t, st, "limitAbove_month_rate")
	if before != after {
		t.Fatalf("expected indicators unchanged, before=%.2f after=%.2f", before, after)
	}
}

func newLLMChatPhase3Router(t *testing.T, rolePath string) (*Handler, http.Handler, *store.Store) {
	t.Helper()

	gin.SetMode(gin.ReleaseMode)
	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}
	if err := st.SetConfig("llm_user_prompt", ""); err != nil {
		t.Fatalf("set user prompt: %v", err)
	}

	engine := dagcalc.NewEngine(dagcalc.NewGraph(), st, 2025, 12, rolePath)
	if rolePath != "" {
		if err := engine.ReloadRules(); err != nil {
			t.Fatalf("reload rules: %v", err)
		}
	}
	handler := NewHandlerWithEngine(st, "", engine)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))
	return handler, router, st
}

func insertChatPhase3WRRow(t *testing.T, st *store.Store) {
	t.Helper()

	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			sales_current_month, sales_last_year_month,
			sales_current_cumulative, sales_last_year_cumulative,
			retail_current_month, retail_last_year_month,
			retail_current_cumulative, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AAA", "企业A", "5101", "wholesale", 1, 1, 2025, 12, 100, 100, 100, 100, 100, 100, 100, 100, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}
}

func performSSERequest(t *testing.T, router http.Handler, body llmChatRequest) *httptest.ResponseRecorder {
	t.Helper()

	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/llm/chat/stream", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func readResultEvent(t *testing.T, raw string) llmResultPayload {
	t.Helper()

	events := strings.Split(raw, "\n\n")
	for _, event := range events {
		lines := strings.Split(event, "\n")
		for _, line := range lines {
			if !strings.HasPrefix(line, "data:") {
				continue
			}
			payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if payload == "" {
				continue
			}
			var streamEvent llmStreamEvent
			if err := json.Unmarshal([]byte(payload), &streamEvent); err != nil {
				t.Fatalf("decode stream event: %v", err)
			}
			if streamEvent.Type == "result" && streamEvent.Result != nil {
				return *streamEvent.Result
			}
		}
	}
	t.Fatalf("missing result event in %s", raw)
	return llmResultPayload{}
}

func findChatPhase3IndicatorValue(t *testing.T, st *store.Store, id string) float64 {
	t.Helper()

	groups, err := dagcalc.CalculateIndicators(st, 2025, 12)
	if err != nil {
		t.Fatalf("calculate indicators: %v", err)
	}
	for _, group := range groups {
		for _, indicator := range group.Indicators {
			if indicator.ID == id {
				return indicator.Value
			}
		}
	}
	t.Fatalf("indicator not found: %s", id)
	return 0
}
