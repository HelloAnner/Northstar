/**
 * LLM 能力探测：测试连通性、流式输出、工具调用、深度思考
 *
 * @author Anner
 * Created on 2026/4/3
 */
package v3

import (
	"context"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tmc/langchaingo/llms"

	"northstar/internal/llm"
)

type llmProbeRequest struct {
	BaseURL string `json:"baseUrl"`
	Model   string `json:"model"`
	APIKey  string `json:"apiKey"`
}

type llmProbeResult struct {
	Connected bool   `json:"connected"`
	Streaming bool   `json:"streaming"`
	Tools     bool   `json:"tools"`
	Reasoning bool   `json:"reasoning"`
	Error     string `json:"error,omitempty"`
	Latency   int    `json:"latency"` // ms
}

// TestLLMConfig 测试模型能力
// POST /api/v1/config/llm/test
func (h *Handler) TestLLMConfig(c *gin.Context) {
	var req llmProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	cfg := llm.Config{
		BaseURL: strings.TrimSpace(req.BaseURL),
		Model:   strings.TrimSpace(req.Model),
		APIKey:  strings.TrimSpace(req.APIKey),
	}
	if err := cfg.Validate(); err != nil {
		c.JSON(http.StatusOK, llmProbeResult{Error: err.Error()})
		return
	}

	result := probeLLM(cfg)

	// 保存能力检测结果到配置
	_ = h.store.SetConfig("llm_supports_streaming", boolToStr(result.Streaming))
	_ = h.store.SetConfig("llm_supports_tools", boolToStr(result.Tools))
	_ = h.store.SetConfig("llm_supports_reasoning", boolToStr(result.Reasoning))

	c.JSON(http.StatusOK, result)
}

const probeTimeout = 15 * time.Second

func probeLLM(cfg llm.Config) llmProbeResult {
	// 第一步：连通性 + 流式（必须先通过才测后续）
	start := time.Now()
	result := probeConnectAndStream(cfg)
	result.Latency = int(time.Since(start).Milliseconds())
	if !result.Connected {
		return result
	}

	// 第二步：工具调用 + 深度思考 并行测试
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); result.Tools = probeToolCalling(cfg) }()
	go func() { defer wg.Done(); result.Reasoning = probeReasoning(cfg) }()
	wg.Wait()

	return result
}

func probeConnectAndStream(cfg llm.Config) llmProbeResult {
	client, err := llm.NewTextClient(cfg, "你是测试助手。")
	if err != nil {
		return llmProbeResult{Error: "初始化失败: " + err.Error()}
	}

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	streamed := false
	_, err = client.Chat(ctx, llm.ChatRequest{
		Messages: []llm.ChatMessage{{Role: "user", Content: "回复ok"}},
	}, func(chunk string) error {
		streamed = true
		return nil
	})
	if err != nil {
		return llmProbeResult{Error: "调用失败: " + err.Error()}
	}
	return llmProbeResult{Connected: true, Streaming: streamed}
}

func probeToolCalling(cfg llm.Config) bool {
	probeTool := llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "get_current_time",
			Description: "获取当前时间",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
	}

	client, err := llm.NewClientWithTools(cfg, "当用户问时间时，必须调用 get_current_time 工具。", []llms.Tool{probeTool})
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	result, err := client.Chat(ctx, llm.ChatRequest{
		Messages: []llm.ChatMessage{{Role: "user", Content: "几点了"}},
	}, nil)
	if err != nil {
		log.Printf("tool probe: %v", err)
		return false
	}
	return len(result.ToolCalls) > 0
}

func probeReasoning(cfg llm.Config) bool {
	if isKnownReasoningModel(cfg.Model) {
		return true
	}

	client, err := llm.NewTextClient(cfg, "用 <think> 标签包裹推理过程再回答。")
	if err != nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	var buf strings.Builder
	result, err := client.Chat(ctx, llm.ChatRequest{
		Messages: []llm.ChatMessage{{Role: "user", Content: "1+1=?"}},
	}, func(chunk string) error {
		buf.WriteString(chunk)
		// 一旦检测到 <think> 就够了，不用等完整响应
		return nil
	})
	if err != nil {
		return false
	}

	content := buf.String()
	if content == "" {
		content = result.Content
	}
	return strings.Contains(content, "<think>")
}

func isKnownReasoningModel(model string) bool {
	lower := strings.ToLower(model)
	for _, p := range []string{"deepseek-r1", "deepseek-reasoner", "o1", "o3", "o4-mini", "qwq", "qwen3"} {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
