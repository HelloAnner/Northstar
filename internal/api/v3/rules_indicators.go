/**
 * 指标中心与规则中心接口
 *
 * @author Anner
 * Created on 2026/2/6
 */

package v3

import (
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"northstar/internal/store"
)

type listIndicatorDefinitionsResponse struct {
	Items []store.IndicatorDefinition `json:"items"`
}

// ListIndicatorDefinitions 获取指标定义列表
// GET /api/indicator-definitions
func (h *Handler) ListIndicatorDefinitions(c *gin.Context) {
	enabledOnly := c.DefaultQuery("enabledOnly", "true") != "false"
	items, err := h.store.ListIndicatorDefinitions(enabledOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取指标定义失败"})
		return
	}
	c.JSON(http.StatusOK, listIndicatorDefinitionsResponse{Items: items})
}

type upsertIndicatorDefinitionRequest struct {
	Name         string  `json:"name"`
	GroupCode    string  `json:"groupCode"`
	GroupName    string  `json:"groupName"`
	GroupOrder   int     `json:"groupOrder"`
	Description  string  `json:"description"`
	Formula      string  `json:"formula"`
	Unit         string  `json:"unit"`
	FloatMin     float64 `json:"floatMin"`
	FloatMax     float64 `json:"floatMax"`
	DisplayOrder int     `json:"displayOrder"`
	Enabled      bool    `json:"enabled"`
}

// UpsertIndicatorDefinition 新增或更新指标定义
// PATCH /api/indicator-definitions/:code
func (h *Handler) UpsertIndicatorDefinition(c *gin.Context) {
	marker := strings.TrimSpace(c.Param("code"))

	var req upsertIndicatorDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = marker
	}
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "指标名称不能为空"})
		return
	}

	groupName := strings.TrimSpace(req.GroupName)
	groupCode := strings.TrimSpace(req.GroupCode)
	if groupName == "" {
		groupName = "自定义指标"
	}
	if groupCode == "" {
		groupCode = groupName
	}

	def := store.IndicatorDefinition{
		Code:         name,
		Name:         name,
		GroupCode:    groupCode,
		GroupName:    groupName,
		GroupOrder:   req.GroupOrder,
		Description:  strings.TrimSpace(req.Description),
		Formula:      strings.TrimSpace(req.Formula),
		Unit:         strings.TrimSpace(req.Unit),
		FloatMin:     req.FloatMin,
		FloatMax:     req.FloatMax,
		DisplayOrder: req.DisplayOrder,
		Enabled:      req.Enabled,
	}

	if def.Formula == "" || def.Unit == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "指标定义字段不完整"})
		return
	}

	if err := h.store.UpsertIndicatorDefinition(def); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存指标定义失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

type ruleDetail struct {
	Rule  store.RuleDefinition      `json:"rule"`
	Links []store.RuleIndicatorLink `json:"links"`
}

type listRulesResponse struct {
	Items []ruleDetail `json:"items"`
}

// ListRules 获取规则列表
// GET /api/rules
func (h *Handler) ListRules(c *gin.Context) {
	enabledOnly := c.DefaultQuery("enabledOnly", "true") != "false"
	rules, err := h.store.ListRuleDefinitions(enabledOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取规则失败"})
		return
	}

	links, err := h.store.ListRuleIndicatorLinks("")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取规则关联失败"})
		return
	}
	linkMap := make(map[string][]store.RuleIndicatorLink)
	for _, link := range links {
		linkMap[link.RuleCode] = append(linkMap[link.RuleCode], link)
	}

	items := make([]ruleDetail, 0, len(rules))
	for _, rule := range rules {
		ruleLinks := linkMap[rule.RuleCode]
		if ruleLinks == nil {
			ruleLinks = make([]store.RuleIndicatorLink, 0)
		}
		items = append(items, ruleDetail{
			Rule:  rule,
			Links: ruleLinks,
		})
	}

	c.JSON(http.StatusOK, listRulesResponse{Items: items})
}

type upsertRuleRequest struct {
	Name           string                    `json:"name"`
	Description    string                    `json:"description"`
	Expression     string                    `json:"expression"`
	Severity       string                    `json:"severity"`
	Suggestion     string                    `json:"suggestion"`
	PreferenceJSON string                    `json:"preferenceJson"`
	DisplayOrder   int                       `json:"displayOrder"`
	Enabled        bool                      `json:"enabled"`
	Links          []store.RuleIndicatorLink `json:"links"`
}

// UpsertRule 新增或更新规则定义
// PATCH /api/rules/:code
func (h *Handler) UpsertRule(c *gin.Context) {
	marker := strings.TrimSpace(c.Param("code"))

	var req upsertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = marker
	}
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则名称不能为空"})
		return
	}

	rule := store.RuleDefinition{
		RuleCode:       name,
		Name:           name,
		Description:    strings.TrimSpace(req.Description),
		Expression:     strings.TrimSpace(req.Expression),
		Severity:       strings.TrimSpace(req.Severity),
		Suggestion:     strings.TrimSpace(req.Suggestion),
		PreferenceJSON: strings.TrimSpace(req.PreferenceJSON),
		DisplayOrder:   req.DisplayOrder,
		Enabled:        req.Enabled,
	}

	if rule.Expression == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则定义字段不完整"})
		return
	}
	if rule.Severity == "" {
		rule.Severity = "warn"
	}
	if hasAssignmentOperator(rule.Expression) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "规则表达式必须是约束判断，不能使用赋值公式"})
		return
	}
	if rule.PreferenceJSON == "" {
		rule.PreferenceJSON = "{}"
	}

	if err := h.store.UpsertRuleDefinition(rule); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存规则失败"})
		return
	}

	for i := range req.Links {
		req.Links[i].RuleCode = name
	}
	if err := h.store.ReplaceRuleIndicatorLinks(name, req.Links); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存规则联动失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "保存成功"})
}

func hasAssignmentOperator(expression string) bool {
	for index := 0; index < len(expression); {
		r, size := utf8.DecodeRuneInString(expression[index:])
		if r != '=' {
			index += size
			continue
		}

		prev := rune(0)
		if index > 0 {
			prev, _ = utf8.DecodeLastRuneInString(expression[:index])
		}

		next := rune(0)
		if index+size < len(expression) {
			next, _ = utf8.DecodeRuneInString(expression[index+size:])
		}

		if prev != '<' && prev != '>' && prev != '!' && prev != '=' && next != '=' {
			return true
		}
		index += size
	}
	return false
}
