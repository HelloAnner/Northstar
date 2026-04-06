/**
 * 规则管理接口
 *
 * @author Anner
 * Created on 2026/3/14
 */

package v3

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"northstar/internal/store"
)

// --- 硬约束 API ---

// ListConstraints 列出所有硬约束。
func (h *Handler) ListConstraints(c *gin.Context) {
	constraints, err := h.store.ListAdjustmentConstraints(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取约束失败"})
		return
	}
	c.JSON(http.StatusOK, constraints)
}

// CreateConstraint 新增硬约束。
func (h *Handler) CreateConstraint(c *gin.Context) {
	var req store.AdjustmentConstraint
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if err := validateConstraint(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Enabled = true
	id, err := h.store.CreateAdjustmentConstraint(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建约束失败"})
		return
	}
	req.ID = id
	_ = h.engine.ReloadRules()
	c.JSON(http.StatusCreated, req)
}

// UpdateConstraint 更新硬约束。
func (h *Handler) UpdateConstraint(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req store.AdjustmentConstraint
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if err := validateConstraint(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = id
	if err := h.store.UpdateAdjustmentConstraint(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新约束失败"})
		return
	}
	_ = h.engine.ReloadRules()
	c.JSON(http.StatusOK, req)
}

// DeleteConstraint 删除硬约束。
func (h *Handler) DeleteConstraint(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.store.DeleteAdjustmentConstraint(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除约束失败"})
		return
	}
	_ = h.engine.ReloadRules()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- 自然语言规则 API ---

// ListNaturalRules 列出所有自然语言规则。
func (h *Handler) ListNaturalRules(c *gin.Context) {
	rules, err := h.store.ListNaturalRules(false)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取规则失败"})
		return
	}
	c.JSON(http.StatusOK, rules)
}

type naturalRuleRequest struct {
	Text string `json:"text"`
}

// CreateNaturalRule 新增自然语言规则。
func (h *Handler) CreateNaturalRule(c *gin.Context) {
	var req naturalRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则内容不能为空"})
		return
	}
	id, err := h.store.CreateNaturalRule(text)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建规则失败"})
		return
	}
	c.JSON(http.StatusCreated, store.NaturalRule{ID: id, Text: text, Enabled: true})
}

// UpdateNaturalRule 更新自然语言规则。
func (h *Handler) UpdateNaturalRule(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req naturalRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则内容不能为空"})
		return
	}
	if err := h.store.UpdateNaturalRule(id, text, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新规则失败"})
		return
	}
	c.JSON(http.StatusOK, store.NaturalRule{ID: id, Text: text, Enabled: true})
}

// DeleteNaturalRule 删除自然语言规则。
func (h *Handler) DeleteNaturalRule(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.store.DeleteNaturalRule(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除规则失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- helpers ---

func parseIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "非法 ID"})
		return 0, false
	}
	return id, true
}

func validateConstraint(c store.AdjustmentConstraint) error {
	switch c.Type {
	case "clamp_target":
		if c.IndicatorID == "" {
			return errMsg("indicatorId 不能为空")
		}
		if c.MinValue == nil && c.MaxValue == nil {
			return errMsg("min 和 max 不能同时为空")
		}
	case "filter_allocation":
		if c.IndicatorID == "" {
			return errMsg("indicatorId 不能为空")
		}
		if c.FilterMode == "" {
			return errMsg("filterMode 不能为空")
		}
	case "compensate":
		if c.TriggerID == "" {
			return errMsg("triggerId 不能为空")
		}
		if c.EnsureID == "" {
			return errMsg("ensureId 不能为空")
		}
		if c.Relation != "gte" && c.Relation != "lte" {
			return errMsg("relation 必须为 gte 或 lte")
		}
	default:
		return errMsg("未知约束类型: " + c.Type)
	}
	return nil
}

type constraintError string

func (e constraintError) Error() string { return string(e) }
func errMsg(msg string) error           { return constraintError(msg) }
