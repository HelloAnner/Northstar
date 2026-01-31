package v3

import (
	"github.com/gin-gonic/gin"
	"northstar/internal/store"
)

// Handler V3 API 处理器
type Handler struct {
	store        *store.Store
	templatePath string
	downloads    *exportDownloadStore
}

// NewHandler 创建 V3 API 处理器
func NewHandler(store *store.Store, templatePath string) *Handler {
	return &Handler{
		store:        store,
		templatePath: templatePath,
		downloads:    newExportDownloadStore(),
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

	// 数据导入
	router.POST("/import", h.Import)

	// 企业数据查询
	router.GET("/companies", h.ListCompanies)
	router.GET("/companies/:id", h.GetCompany)
	router.PATCH("/companies/:id", h.UpdateCompany)
	router.POST("/companies/reset", h.ResetCompanies)

	// 指标查询
	router.GET("/indicators", h.GetIndicators)
	// 智能调整
	router.POST("/optimize", h.Optimize)

	// 数据导出
	router.POST("/export", h.Export)
	router.POST("/export/stream", h.ExportStream)
	router.GET("/export/download/:token", h.DownloadExport)
}
