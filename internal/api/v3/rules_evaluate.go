/**
 * 规则校验接口
 *
 * @author Anner
 * Created on 2026/2/6
 */

package v3

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"northstar/internal/ruleengine"
)

type listRuleEvaluationsResponse struct {
	Year  int                         `json:"year"`
	Month int                         `json:"month"`
	Items []ruleengine.RuleEvaluation `json:"items"`
}

// EvaluateRules 计算当前月份规则命中情况
// GET /api/rules/evaluate
func (h *Handler) EvaluateRules(c *gin.Context) {
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取当前年月失败"})
		return
	}

	enabledOnly := c.DefaultQuery("enabledOnly", "true") != "false"
	items, err := ruleengine.EvaluateRules(h.store, year, month, enabledOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "规则校验失败"})
		return
	}

	c.JSON(http.StatusOK, listRuleEvaluationsResponse{
		Year:  year,
		Month: month,
		Items: items,
	})
}
