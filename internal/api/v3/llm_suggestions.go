/**
 * AI 推荐问题动态生成
 *
 * @author Anner
 * Created on 2026/3/28
 */
package v3

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"northstar/internal/dagcalc"
	"northstar/internal/llm"
)

type suggestion struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type suggestionsResponse struct {
	Chat   []suggestion `json:"chat"`
	Adjust []suggestion `json:"adjust"`
}

const suggestionsPromptTpl = `你是 Northstar 经济数据统计系统的 AI 助手。

当前数据上下文：
数据期间：%d年%d月
当前核心指标：
%s

请根据以上真实指标数据，生成推荐问题。要求：
1. 问题必须基于真实数据，引用具体指标名称和数值
2. 问题要有分析价值，不要泛泛而谈
3. 关注异常值、负增长、大幅波动等值得关注的数据点

请严格按以下 JSON 格式输出，不要输出其他内容：
{
  "chat": [
    {"title": "简短标题(6字内)", "content": "完整问题描述"},
    {"title": "简短标题(6字内)", "content": "完整问题描述"}
  ],
  "adjust": [
    {"title": "简短标题(6字内)", "content": "完整调整指令"},
    {"title": "简短标题(6字内)", "content": "完整调整指令"}
  ]
}

chat 类型：分析类问题（解释原因、走势分析、对比分析等）
adjust 类型：调整类指令（调整目标值、添加规则等，使用具体数字）`

// GetSuggestions 生成推荐问题
// GET /api/llm/suggestions
func (h *Handler) GetSuggestions(c *gin.Context) {
	year, month, ok := h.loadCurrentYM(c)
	if !ok {
		return
	}
	groups, err := dagcalc.CalculateIndicators(h.store, year, month)
	if err != nil {
		c.JSON(http.StatusOK, suggestionsResponse{})
		return
	}
	roundIndicatorGroupsInPlace(groups)
	summary := buildIndicatorSummary(groups)
	prompt := fmt.Sprintf(suggestionsPromptTpl, year, month, summary)

	client, err := h.newLLMClient(strings.TrimSpace(prompt))
	if err != nil {
		log.Printf("suggestions: llm client error: %v", err)
		c.JSON(http.StatusOK, suggestionsResponse{})
		return
	}

	result, err := client.Chat(c.Request.Context(), llm.ChatRequest{
		Messages: []llm.ChatMessage{{
			Role:    "user",
			Content: "请根据当前指标数据生成推荐问题。",
		}},
		Year:  year,
		Month: month,
	}, nil)
	if err != nil {
		log.Printf("suggestions: llm chat error: %v", err)
		c.JSON(http.StatusOK, suggestionsResponse{})
		return
	}

	resp := parseSuggestionsResponse(result.Content)
	c.JSON(http.StatusOK, resp)
}

// parseSuggestionsResponse 从 LLM 输出中解析推荐问题
func parseSuggestionsResponse(content string) suggestionsResponse {
	content = strings.TrimSpace(content)
	// 去掉 markdown 代码块包裹
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var resp suggestionsResponse
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		log.Printf("suggestions: parse error: %v, content: %s", err, content)
		return suggestionsResponse{}
	}
	return resp
}
