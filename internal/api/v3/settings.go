/**
 * AI 偏好设置接口
 *
 * @author Anner
 * Created on 2026/3/14
 */
package v3

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

const maxUserPromptLength = 500
const maxSystemPromptLength = 5000

type userPromptRequest struct {
	Content string `json:"content"`
}

type userPromptResponse struct {
	Content string `json:"content"`
}

// GetUserPrompt 获取用户 AI 偏好提示词。
func (h *Handler) GetUserPrompt(c *gin.Context) {
	c.JSON(http.StatusOK, userPromptResponse{
		Content: h.getConfigValue("llm_user_prompt", ""),
	})
}

// UpdateUserPrompt 更新用户 AI 偏好提示词。
func (h *Handler) UpdateUserPrompt(c *gin.Context) {
	var req userPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	content := strings.TrimSpace(req.Content)
	if utf8.RuneCountInString(content) > maxUserPromptLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content 长度不能超过 500"})
		return
	}
	if err := h.store.SetConfig("llm_user_prompt", content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存 AI 偏好失败"})
		return
	}

	c.JSON(http.StatusOK, userPromptResponse{Content: content})
}

// GetSystemPrompt 获取系统提示词正文。
func (h *Handler) GetSystemPrompt(c *gin.Context) {
	content := h.getConfigValue("llm_system_prompt", "")
	c.JSON(http.StatusOK, userPromptResponse{Content: content})
}

// UpdateSystemPrompt 更新系统提示词正文。
func (h *Handler) UpdateSystemPrompt(c *gin.Context) {
	var req userPromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	content := strings.TrimSpace(req.Content)
	if utf8.RuneCountInString(content) > maxSystemPromptLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content 长度不能超过 5000"})
		return
	}
	if err := h.store.SetConfig("llm_system_prompt", content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存系统提示词失败"})
		return
	}
	_ = h.store.SetConfig("llm_system_prompt_modified", "true")

	c.JSON(http.StatusOK, userPromptResponse{Content: content})
}
