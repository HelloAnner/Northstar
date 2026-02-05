/**
 * LLM 对话流式接口
 *
 * @author Anner
 * @since 12.0
 * Created on 2026/2/1
 */
package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"northstar/internal/linkage"
	"northstar/internal/llm"
)

type llmChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmChatRequest struct {
	SessionID string           `json:"sessionId"`
	Messages  []llmChatMessage `json:"messages"`
}

type llmToolSummary struct {
	UpdatedCompanies     int               `json:"updatedCompanies"`
	TargetIndicators     int               `json:"targetIndicators"`
	Optimized            bool              `json:"optimized"`
	ToolPositionCount    int               `json:"toolPositionCount,omitempty"`
	ImpactCellCount      int               `json:"impactCellCount,omitempty"`
	ImpactIndicatorCount int               `json:"impactIndicatorCount,omitempty"`
	ImpactCells          []linkage.UICoord `json:"impactCells,omitempty"`
	ImpactIndicators     []string          `json:"impactIndicators,omitempty"`
	Warnings             []string          `json:"warnings,omitempty"`
}

type llmStreamEvent struct {
	Type    string          `json:"type"`
	Content string          `json:"content,omitempty"`
	Summary *llmToolSummary `json:"summary,omitempty"`
	Error   string          `json:"error,omitempty"`
}

type llmStreamWriter struct {
	writer  gin.ResponseWriter
	flusher http.Flusher
}

type llmChatContext struct {
	ctx     context.Context
	request llmChatRequest
	year    int
	month   int
	client  *llm.Client
}

// StreamLLMChat LLM 聊天接口（SSE）
// POST /api/llm/chat/stream
func (h *Handler) StreamLLMChat(c *gin.Context) {
	req, err := decodeLLMChatRequest(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	log.Printf("llm chat start: session=%s messages=%d", req.SessionID, len(req.Messages))
	chatCtx, err := h.buildLLMChatContext(c.Request.Context(), req)
	if err != nil {
		log.Printf("llm chat context error: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	stream := newLLMStreamWriter(c)
	if err := h.runLLMChat(stream, chatCtx); err != nil {
		log.Printf("llm chat run error: %v", err)
		stream.SendError(err)
		return
	}
	_ = stream.Send(llmStreamEvent{Type: "final"})
}

func decodeLLMChatRequest(c *gin.Context) (llmChatRequest, error) {
	var req llmChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return req, err
	}
	return req, nil
}

func (h *Handler) buildLLMChatContext(ctx context.Context, req llmChatRequest) (llmChatContext, error) {
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		return llmChatContext{}, fmt.Errorf("系统未初始化")
	}
	cfg, err := llm.LoadConfig(h.store)
	if err != nil {
		return llmChatContext{}, err
	}
	prompt := llm.BuildSystemPrompt(llm.PromptContext{Year: year, Month: month})
	client, err := llm.NewClient(cfg, prompt)
	if err != nil {
		return llmChatContext{}, fmt.Errorf("初始化模型失败: %v", err)
	}
	return llmChatContext{ctx: ctx, request: req, year: year, month: month, client: client}, nil
}

func toLLMChatRequest(req llmChatRequest, year, month int) llm.ChatRequest {
	messages := make([]llm.ChatMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, llm.ChatMessage{Role: msg.Role, Content: msg.Content})
	}
	return llm.ChatRequest{Messages: messages, Year: year, Month: month}
}

func newLLMStreamWriter(c *gin.Context) *llmStreamWriter {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	flusher, _ := c.Writer.(http.Flusher)
	return &llmStreamWriter{writer: c.Writer, flusher: flusher}
}

func (w *llmStreamWriter) Send(event llmStreamEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := w.writer.Write([]byte("data: " + string(payload) + "\n\n")); err != nil {
		return err
	}
	if w.flusher != nil {
		w.flusher.Flush()
	}
	return nil
}

func (w *llmStreamWriter) SendDelta(content string) error {
	return w.Send(llmStreamEvent{Type: "message_delta", Content: content})
}

func (w *llmStreamWriter) SendError(err error) {
	_ = w.Send(llmStreamEvent{Type: "error", Error: err.Error()})
}

func (h *Handler) runLLMChat(stream *llmStreamWriter, ctx llmChatContext) error {
	result, err := ctx.client.Chat(
		ctx.ctx,
		toLLMChatRequest(ctx.request, ctx.year, ctx.month),
		stream.SendDelta,
	)
	if err != nil {
		return err
	}
	parsed, err := llm.ParseToolCalls(result.ToolCalls)
	if err != nil {
		return err
	}
	summary, err := h.executeLLMTools(parsed, ctx.year, ctx.month)
	if err != nil {
		return err
	}
	return stream.Send(llmStreamEvent{Type: "tool_result", Summary: &summary})
}

func (h *Handler) executeLLMTools(parsed llm.ParsedToolCalls, year, month int) (llmToolSummary, error) {
	excludes, updated, applied, warnings, err := h.applyLLMCompanyUpdates(parsed.CompanyUpdates, year, month)
	if err != nil {
		return llmToolSummary{}, err
	}
	optimized := false
	if len(parsed.IndicatorTargets) > 0 {
		if err := applyIndicatorTargetsWithExcludes(h.store, year, month, parsed.IndicatorTargets, excludes); err != nil {
			return llmToolSummary{}, err
		}
		optimized = true
	}
	impact, err := buildLLMToolImpact(h.store, year, month, applied, parsed.IndicatorTargets)
	if err != nil {
		return llmToolSummary{}, err
	}
	return llmToolSummary{
		UpdatedCompanies:     updated,
		TargetIndicators:     len(parsed.IndicatorTargets),
		Optimized:            optimized,
		ToolPositionCount:    impact.ToolPositionCount,
		ImpactCellCount:      impact.ImpactCellCount,
		ImpactIndicatorCount: impact.ImpactIndicatorCount,
		ImpactCells:          impact.ImpactCells,
		ImpactIndicators:     impact.ImpactIndicators,
		Warnings:             warnings,
	}, nil
}
