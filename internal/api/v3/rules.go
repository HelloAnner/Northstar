/**
 * 规则管理接口
 *
 * @author Anner
 * Created on 2026/3/14
 */

package v3

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	rulepkg "northstar/internal/rules"
)

var numberedRulePattern = regexp.MustCompile(`^\d+\.\s+(.+)$`)

// RuleItem 表示 rules.md 中的一条规则。
type RuleItem struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

type RuleConverter interface {
	ConvertAsync(mdContent string)
}

type rulesStatusResponse struct {
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
	Error     string `json:"error"`
	Step      string `json:"step,omitempty"`
	Attempt   string `json:"attempt,omitempty"`
}

type ruleTextRequest struct {
	Text string `json:"text"`
}

type rulesFileRepo struct {
	path string
}

func (r *rulesFileRepo) ReadRules() ([]RuleItem, error) {
	if strings.TrimSpace(r.path) == "" {
		return []RuleItem{}, nil
	}
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []RuleItem{}, nil
		}
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	items := make([]RuleItem, 0)
	for _, line := range lines {
		text := parseRuleLine(line)
		if text == "" {
			continue
		}
		items = append(items, RuleItem{Index: len(items) + 1, Text: text})
	}
	return items, nil
}

func parseRuleLine(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return ""
	}
	match := numberedRulePattern.FindStringSubmatch(trimmed)
	if len(match) != 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func (r *rulesFileRepo) WriteRules(items []RuleItem) error {
	if strings.TrimSpace(r.path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(r.path), 0755); err != nil {
		return err
	}
	builder := strings.Builder{}
	builder.WriteString("# 调整规则\n\n")
	for idx, item := range items {
		builder.WriteString(strconv.Itoa(idx + 1))
		builder.WriteString(". ")
		builder.WriteString(strings.TrimSpace(item.Text))
		builder.WriteString("\n")
	}
	return os.WriteFile(r.path, []byte(builder.String()), 0644)
}

func (r *rulesFileRepo) ReadContent() (string, error) {
	if strings.TrimSpace(r.path) == "" {
		return "", nil
	}
	data, err := os.ReadFile(r.path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ConfigureRuleManagement 配置 rules.md 与 role.json 路径。
func (h *Handler) ConfigureRuleManagement(mdPath string, rolePath string) {
	h.rulesRepo = &rulesFileRepo{path: mdPath}
	h.ruleConverterFactory = func() (RuleConverter, error) {
		return rulepkg.NewConverter(h.store, h.engine, rolePath, mdPath)
	}
}

// SetRuleConverterFactory 设置规则转换器工厂，主要用于测试。
func (h *Handler) SetRuleConverterFactory(factory func() (RuleConverter, error)) {
	h.ruleConverterFactory = factory
}

// GetRules 获取规则列表。
func (h *Handler) GetRules(c *gin.Context) {
	repo, ok := h.ensureRulesRepo(c)
	if !ok {
		return
	}
	items, err := repo.ReadRules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取规则失败"})
		return
	}
	c.JSON(http.StatusOK, items)
}

// CreateRule 新增规则，仅保存不自动转换。
func (h *Handler) CreateRule(c *gin.Context) {
	req, ok := bindRuleTextRequest(c)
	if !ok {
		return
	}
	items, _, ok := h.mutateRules(c, func(items []RuleItem) ([]RuleItem, bool) {
		return append(items, RuleItem{Index: len(items) + 1, Text: req.Text}), true
	})
	if !ok {
		return
	}
	h.markRulesPending()
	c.JSON(http.StatusCreated, items)
}

// UpdateRule 更新指定规则，仅保存不自动转换。
func (h *Handler) UpdateRule(c *gin.Context) {
	req, ok := bindRuleTextRequest(c)
	if !ok {
		return
	}
	index, ok := parseRuleIndex(c)
	if !ok {
		return
	}
	items, _, ok := h.mutateRules(c, func(items []RuleItem) ([]RuleItem, bool) {
		if index < 1 || index > len(items) {
			c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
			return nil, false
		}
		items[index-1].Text = req.Text
		return items, true
	})
	if !ok {
		return
	}
	h.markRulesPending()
	c.JSON(http.StatusOK, items)
}

// DeleteRule 删除指定规则，仅保存不自动转换。
func (h *Handler) DeleteRule(c *gin.Context) {
	index, ok := parseRuleIndex(c)
	if !ok {
		return
	}
	items, _, ok := h.mutateRules(c, func(items []RuleItem) ([]RuleItem, bool) {
		if index < 1 || index > len(items) {
			c.JSON(http.StatusNotFound, gin.H{"error": "规则不存在"})
			return nil, false
		}
		next := append([]RuleItem{}, items[:index-1]...)
		next = append(next, items[index:]...)
		return next, true
	})
	if !ok {
		return
	}
	h.markRulesPending()
	c.JSON(http.StatusOK, items)
}

// ConvertRules 手动触发规则转换。
func (h *Handler) ConvertRules(c *gin.Context) {
	repo, ok := h.ensureRulesRepo(c)
	if !ok {
		return
	}
	content, err := repo.ReadContent()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取规则失败"})
		return
	}
	h.triggerRuleConvert(content)
	c.JSON(http.StatusOK, gin.H{"status": "running"})
}

// GetRuleStatus 获取转换状态，单次查询返回全部状态字段。
func (h *Handler) GetRuleStatus(c *gin.Context) {
	keys := []string{
		"rules_convert_status", "rules_convert_at",
		"rules_convert_error", "rules_convert_step", "rules_convert_attempt",
	}
	values, err := h.store.GetConfigBatch(keys)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取状态失败"})
		return
	}
	fallback := func(key string) string {
		if v, ok := values[key]; ok {
			return v
		}
		return ""
	}
	status := fallback("rules_convert_status")
	if status == "" {
		status = "idle"
	}
	c.JSON(http.StatusOK, rulesStatusResponse{
		Status:    status,
		UpdatedAt: fallback("rules_convert_at"),
		Error:     fallback("rules_convert_error"),
		Step:      fallback("rules_convert_step"),
		Attempt:   fallback("rules_convert_attempt"),
	})
}

func bindRuleTextRequest(c *gin.Context) (ruleTextRequest, bool) {
	var req ruleTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return ruleTextRequest{}, false
	}
	req.Text = strings.TrimSpace(req.Text)
	if req.Text == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text 不能为空"})
		return ruleTextRequest{}, false
	}
	return req, true
}

func parseRuleIndex(c *gin.Context) (int, bool) {
	index, err := strconv.Atoi(c.Param("index"))
	if err != nil || index <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "非法规则序号"})
		return 0, false
	}
	return index, true
}

func (h *Handler) mutateRules(c *gin.Context, mutate func([]RuleItem) ([]RuleItem, bool)) ([]RuleItem, string, bool) {
	repo, ok := h.ensureRulesRepo(c)
	if !ok {
		return nil, "", false
	}
	items, err := repo.ReadRules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取规则失败"})
		return nil, "", false
	}
	next, ok := mutate(items)
	if !ok {
		return nil, "", false
	}
	reindexRules(next)
	if err := repo.WriteRules(next); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入规则失败"})
		return nil, "", false
	}
	content, err := repo.ReadContent()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取规则失败"})
		return nil, "", false
	}
	return next, content, true
}

func reindexRules(items []RuleItem) {
	for idx := range items {
		items[idx].Index = idx + 1
	}
}

func (h *Handler) ensureRulesRepo(c *gin.Context) (*rulesFileRepo, bool) {
	if h.rulesRepo == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "规则管理未初始化"})
		return nil, false
	}
	return h.rulesRepo, true
}

func (h *Handler) triggerRuleConvert(content string) {
	converter, err := h.newRuleConverter()
	if err != nil || converter == nil {
		if err == nil {
			err = os.ErrInvalid
		}
		_ = h.store.SetConfig("rules_convert_status", "error")
		_ = h.store.SetConfig("rules_convert_error", err.Error())
		return
	}
	converter.ConvertAsync(content)
}

func (h *Handler) newRuleConverter() (RuleConverter, error) {
	if h.ruleConverterFactory == nil {
		return nil, nil
	}
	return h.ruleConverterFactory()
}

// markRulesPending 标记规则已修改、待转换。
func (h *Handler) markRulesPending() {
	_ = h.store.SetConfig("rules_convert_status", "pending")
}

func (h *Handler) getRuleConfig(key string, fallback string) string {
	value, err := h.store.GetConfig(key)
	if err != nil {
		return fallback
	}
	return value
}
