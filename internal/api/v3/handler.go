package v3

import (
	"github.com/gin-gonic/gin"
	"northstar/internal/dagcalc"
	"northstar/internal/store"
)

// Handler V3 API 处理器
type Handler struct {
	store                *store.Store
	templatePath         string
	downloads            *exportDownloadStore
	engine               *dagcalc.Engine
	rulesRepo            *rulesFileRepo
	ruleConverterFactory func() (RuleConverter, error)
	llmClientFactory     func(prompt string) (LLMChatClient, error)
}

// NewHandler 创建 V3 API 处理器
func NewHandler(store *store.Store, templatePath string) *Handler {
	return NewHandlerWithEngine(store, templatePath, nil)
}

// NewHandlerWithEngine 创建带规则引擎的 V3 API 处理器。
func NewHandlerWithEngine(store *store.Store, templatePath string, engine *dagcalc.Engine) *Handler {
	if engine == nil {
		engine = dagcalc.NewEngine(dagcalc.NewGraph(), store, 0, 0, "")
	}
	return &Handler{
		store:        store,
		templatePath: templatePath,
		downloads:    newExportDownloadStore(),
		engine:       engine,
	}
}

// RegisterRoutes 注册 V3 API 路由
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	// 系统状态
	router.GET("/status", h.GetStatus)
	// 可用月份
	router.GET("/months", h.ListMonths)
	router.POST("/months/select", h.SelectMonth)

	// 配置管理
	router.GET("/config", h.GetConfig)
	router.PATCH("/config", h.UpdateConfig)
	router.GET("/settings/user-prompt", h.GetUserPrompt)
	router.PUT("/settings/user-prompt", h.UpdateUserPrompt)
	router.GET("/rules", h.GetRules)
	router.POST("/rules", h.CreateRule)
	router.PUT("/rules/:index", h.UpdateRule)
	router.DELETE("/rules/:index", h.DeleteRule)
	router.POST("/rules/convert", h.ConvertRules)
	router.GET("/rules/status", h.GetRuleStatus)

	// 数据导入
	router.POST("/import", h.Import)
	router.GET("/import/preview", h.GetImportPreview)
	router.GET("/import/sheet", h.GetImportSheet)
	router.GET("/import/verify", h.VerifyImport)

	// 企业数据查询
	router.GET("/companies", h.ListCompanies)
	router.PATCH("/companies/batch", h.BatchUpdateCompanies)
	router.GET("/companies/:id", h.GetCompany)
	router.PATCH("/companies/:id", h.UpdateCompany)
	router.POST("/companies/reset", h.ResetCompanies)

	// 指标查询
	router.GET("/indicators", h.GetIndicators)
	router.GET("/indicator-definitions", h.ListIndicatorDefinitions)
	router.PATCH("/indicator-definitions/:code", h.UpsertIndicatorDefinition)
	router.GET("/rules", h.ListRules)
	router.GET("/rules/evaluate", h.EvaluateRules)
	router.PATCH("/rules/:code", h.UpsertRule)
	// 联动预览
	router.POST("/linkage/preview", h.PreviewLinkage)
	// 智能调整
	router.POST("/optimize", h.Optimize)
	// LLM 对话
	router.POST("/llm/chat/stream", h.StreamLLMChat)
	router.GET("/llm/suggestions", h.GetSuggestions)
	// 对话历史
	router.GET("/chat/sessions", h.ListChatSessions)
	router.POST("/chat/sessions", h.SaveChatSession)
	router.GET("/chat/sessions/:id", h.GetChatSession)
	router.DELETE("/chat/sessions/:id", h.DeleteChatSession)

	// 数据导出
	router.POST("/export", h.Export)
	router.POST("/export/stream", h.ExportStream)
	router.GET("/export/download/:token", h.DownloadExport)
}

// SetLLMClientFactory 设置 AI 对话客户端工厂，主要用于测试。
func (h *Handler) SetLLMClientFactory(factory func(prompt string) (LLMChatClient, error)) {
	h.llmClientFactory = factory
}
