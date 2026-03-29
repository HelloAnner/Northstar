/**
 * 对话历史 API
 *
 * @author Anner
 * Created on 2026/3/28
 */
package v3

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"northstar/internal/store"
)

// ListChatSessions 获取对话历史列表
// GET /api/chat/sessions
func (h *Handler) ListChatSessions(c *gin.Context) {
	sessions, err := h.store.ListChatSessions(50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取对话历史失败"})
		return
	}
	if sessions == nil {
		sessions = []store.ChatSession{}
	}
	c.JSON(http.StatusOK, gin.H{"items": sessions})
}

// GetChatSession 获取单个会话的消息
// GET /api/chat/sessions/:id
func (h *Handler) GetChatSession(c *gin.Context) {
	id := c.Param("id")
	messages, err := h.store.GetChatMessages(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取对话消息失败"})
		return
	}
	if messages == nil {
		messages = []store.ChatMessage{}
	}
	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

type saveChatRequest struct {
	SessionID string            `json:"sessionId"`
	Title     string            `json:"title"`
	Mode      string            `json:"mode"`
	Messages  []saveChatMessage `json:"messages"`
}

type saveChatMessage struct {
	ID           string          `json:"id"`
	Role         string          `json:"role"`
	Content      string          `json:"content"`
	AppliedRules json.RawMessage `json:"appliedRules,omitempty"`
	RuleAdded    json.RawMessage `json:"ruleAdded,omitempty"`
}

// SaveChatSession 保存对话（创建或更新）
// POST /api/chat/sessions
func (h *Handler) SaveChatSession(c *gin.Context) {
	var req saveChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.SessionID == "" || len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId 和 messages 不能为空"})
		return
	}

	exists, err := h.store.GetChatSessionExists(req.SessionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查会话失败"})
		return
	}

	if !exists {
		title := req.Title
		if title == "" {
			title = extractTitle(req.Messages)
		}
		mode := req.Mode
		if mode == "" {
			mode = "chat"
		}
		if err := h.store.CreateChatSession(req.SessionID, title, mode); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败"})
			return
		}
	} else {
		if err := h.store.TouchChatSession(req.SessionID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新会话失败"})
			return
		}
	}

	for i, msg := range req.Messages {
		if err := h.store.SaveChatMessage(store.ChatMessage{
			ID:           msg.ID,
			SessionID:    req.SessionID,
			Role:         msg.Role,
			Content:      msg.Content,
			AppliedRules: msg.AppliedRules,
			RuleAdded:    msg.RuleAdded,
			Seq:          i,
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存消息失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DeleteChatSession 删除对话
// DELETE /api/chat/sessions/:id
func (h *Handler) DeleteChatSession(c *gin.Context) {
	id := c.Param("id")
	if err := h.store.DeleteChatSession(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除会话失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// extractTitle 从消息列表中提取标题（首条用户消息前 30 字）
func extractTitle(messages []saveChatMessage) string {
	for _, msg := range messages {
		if msg.Role == "user" && msg.Content != "" {
			title := []rune(msg.Content)
			if len(title) > 30 {
				return string(title[:30]) + "…"
			}
			return string(title)
		}
	}
	return "新对话"
}
