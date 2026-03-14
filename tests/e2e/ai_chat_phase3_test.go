//go:build e2e

/**
 * AI Chat Phase 3 端到端测试
 *
 * @author Anner
 * Created on 2026/3/14
 */

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	v3 "northstar/internal/api/v3"
	"northstar/internal/dagcalc"
	"northstar/internal/llm"
	"northstar/internal/store"
)

type fakePhase3LLMClient struct {
	result  llm.ChatResult
	err     error
	chunks  []string
	lastReq llm.ChatRequest
}

func (c *fakePhase3LLMClient) Chat(_ context.Context, req llm.ChatRequest, stream func(string) error) (llm.ChatResult, error) {
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

type phase3ClientFactory struct {
	prompts []string
	clients []v3.LLMChatClient
}

func (f *phase3ClientFactory) newClient(prompt string) (v3.LLMChatClient, error) {
	f.prompts = append(f.prompts, prompt)
	if len(f.clients) == 0 {
		return nil, nil
	}
	client := f.clients[0]
	f.clients = f.clients[1:]
	return client, nil
}

type phase3ChatRequest struct {
	Mode     string              `json:"mode"`
	Messages []phase3ChatMessage `json:"messages"`
}

type phase3ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type phase3ResultPayload struct {
	Mode         string        `json:"mode"`
	Reply        string        `json:"reply"`
	AppliedRules []appliedRule `json:"appliedRules"`
}

type phase3StreamEvent struct {
	Type   string               `json:"type"`
	Result *phase3ResultPayload `json:"result"`
	Error  string               `json:"error"`
}

type phase3UserPromptPayload struct {
	Content string `json:"content"`
}

type phase3Env struct {
	baseURL string
	store   *store.Store
	handler *v3.Handler
	factory *phase3ClientFactory
}

func TestAIChatPhase3E2E_SettingsAndChatMode(t *testing.T) {
	env := newPhase3Env(t, "")
	env.factory.clients = []v3.LLMChatClient{
		&fakePhase3LLMClient{
			chunks: []string{"当前批发增速偏高"},
			result: llm.ChatResult{Content: "当前批发增速偏高"},
		},
	}

	env.putUserPrompt(t, "回答尽量直接，先说结论。")
	result := env.postChat(t, "chat", "解释当前批发增速")

	if result.Mode != "chat" {
		t.Fatalf("unexpected mode: %+v", result)
	}
	if !strings.Contains(result.Reply, "批发增速") {
		t.Fatalf("unexpected reply: %+v", result)
	}
	if len(env.factory.prompts) != 1 || !strings.Contains(env.factory.prompts[0], "回答尽量直接") {
		t.Fatalf("expected user prompt in system prompt, got %+v", env.factory.prompts)
	}
}

func TestAIChatPhase3E2E_AdjustAndFallback(t *testing.T) {
	env := newPhase3Env(t, `{
  "version": "1.0",
  "rules": [
    {"id":"c1","name":"limit","type":"clamp_target","indicator":"wholesale_month_rate","max":10}
  ]
}`)
	insertWRRateRow(t, env.store, "W1", "wholesale", 100, 100)

	env.factory.clients = []v3.LLMChatClient{
		&fakePhase3LLMClient{
			result: llm.ChatResult{Content: `{"actions":[{"type":"set_target","indicatorId":"wholesale_month_rate","value":15}]}`},
		},
		&fakePhase3LLMClient{
			chunks: []string{"已完成调整"},
			result: llm.ChatResult{Content: "已完成调整"},
		},
	}
	result := env.postChat(t, "adjust", "把批发当月增速调到 15%")
	if result.Mode != "adjust" {
		t.Fatalf("unexpected adjust result: %+v", result)
	}
	rule := requireSingleRule(t, result.AppliedRules)
	assertRuleType(t, rule.Type, "clamp_target")
	assertFloatEqual(t, env.indicatorValue(t, "wholesale_month_rate"), 10)

	env.factory.clients = []v3.LLMChatClient{
		&fakePhase3LLMClient{
			result: llm.ChatResult{Content: `{"actions":[]}`},
		},
		&fakePhase3LLMClient{
			chunks: []string{"这是咨询回复"},
			result: llm.ChatResult{Content: "这是咨询回复"},
		},
	}
	fallback := env.postChat(t, "adjust", "当前零售增速为什么偏低？")
	if fallback.Mode != "chat" {
		t.Fatalf("expected fallback chat mode, got %+v", fallback)
	}
	if len(fallback.AppliedRules) != 0 {
		t.Fatalf("expected no applied rules, got %+v", fallback.AppliedRules)
	}
}

func newPhase3Env(t *testing.T, roleJSON string) *phase3Env {
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
	if err := st.SetConfig("llm_user_prompt", ""); err != nil {
		t.Fatalf("set llm_user_prompt: %v", err)
	}

	rolePath := filepath.Join(tmpDir, "config", "role.json")
	if strings.TrimSpace(roleJSON) != "" {
		writeRoleJSON(t, rolePath, roleJSON)
	}

	engine := dagcalc.NewEngine(dagcalc.NewGraph(), st, 2025, 12, rolePath)
	if err := engine.ReloadRules(); err != nil {
		t.Fatalf("reload rules: %v", err)
	}

	handler := v3.NewHandlerWithEngine(st, "", engine)
	factory := &phase3ClientFactory{}
	handler.SetLLMClientFactory(factory.newClient)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api"))
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &phase3Env{
		baseURL: server.URL,
		store:   st,
		handler: handler,
		factory: factory,
	}
}

func (e *phase3Env) putUserPrompt(t *testing.T, content string) {
	t.Helper()

	body, err := json.Marshal(phase3UserPromptPayload{Content: content})
	if err != nil {
		t.Fatalf("marshal prompt body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPut, e.baseURL+"/api/settings/user-prompt", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build prompt request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("call prompt api: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected prompt status: %d", resp.StatusCode)
	}
	var payload phase3UserPromptPayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode prompt response: %v", err)
	}
	if payload.Content != content {
		t.Fatalf("unexpected prompt payload: %+v", payload)
	}
}

func (e *phase3Env) postChat(t *testing.T, mode string, content string) phase3ResultPayload {
	t.Helper()

	body, err := json.Marshal(phase3ChatRequest{
		Mode: mode,
		Messages: []phase3ChatMessage{{
			Role:    "user",
			Content: content,
		}},
	})
	if err != nil {
		t.Fatalf("marshal chat body: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, e.baseURL+"/api/llm/chat/stream", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build chat request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("call chat api: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected chat status: %d", resp.StatusCode)
	}

	raw, err := ioReadAll(resp)
	if err != nil {
		t.Fatalf("read sse: %v", err)
	}
	return parsePhase3Result(t, raw)
}

func (e *phase3Env) indicatorValue(t *testing.T, id string) float64 {
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

func parsePhase3Result(t *testing.T, raw string) phase3ResultPayload {
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
			var streamEvent phase3StreamEvent
			if err := json.Unmarshal([]byte(payload), &streamEvent); err != nil {
				t.Fatalf("decode phase3 stream event: %v", err)
			}
			if streamEvent.Error != "" {
				t.Fatalf("unexpected stream error: %s", streamEvent.Error)
			}
			if streamEvent.Type == "result" && streamEvent.Result != nil {
				return *streamEvent.Result
			}
		}
	}
	t.Fatalf("missing result event in %s", raw)
	return phase3ResultPayload{}
}

func ioReadAll(resp *http.Response) (string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
