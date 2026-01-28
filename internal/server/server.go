package server

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"

	"northstar/internal/config"
	"northstar/internal/server/handlers"
	"northstar/internal/service/calculator"
	"northstar/internal/service/project"
	"northstar/internal/service/store"
)

//go:embed all:dist
var staticFiles embed.FS

// Server HTTP服务器
type Server struct {
	router    *gin.Engine
	store     *store.MemoryStore
	engine    *calculator.Engine
	optimizer *calculator.Optimizer
	projects  *project.Manager
	handlers  *handlers.Handlers
}

// NewServer 创建服务器
func NewServer(cfg *config.AppConfig) *Server {
	devMode := cfg.Server.DevMode
	if !devMode {
		gin.SetMode(gin.ReleaseMode)
	}

	memStore := store.NewMemoryStore()
	calcEngine := calculator.NewEngine(memStore)
	opt := calculator.NewOptimizer(memStore, calcEngine)

	dataDir, err := config.EnsureDataDir(cfg)
	if err != nil {
		dataDir = cfg.Data.DataDir
	}
	projectManager, err := project.NewManager(dataDir, memStore, calcEngine)
	if err != nil {
		projectManager, _ = project.NewManager(cfg.Data.DataDir, memStore, calcEngine)
	}

	h := handlers.NewHandlers(memStore, calcEngine, opt, projectManager)

	s := &Server{
		router:    gin.Default(),
		store:     memStore,
		engine:    calcEngine,
		optimizer: opt,
		projects:  projectManager,
		handlers:  h,
	}

	s.setupRoutes(devMode)

	return s
}

// setupRoutes 设置路由
func (s *Server) setupRoutes(devMode bool) {
	// CORS
	s.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API路由
	api := s.router.Group("/api/v1")
	{
		// 项目
		api.GET("/projects", s.handlers.ListProjects)
		api.POST("/projects", s.handlers.CreateProject)
		api.GET("/projects/current", s.handlers.GetCurrentProject)
		api.POST("/projects/current/save", s.handlers.SaveCurrentProject)
		api.POST("/projects/current/undo", s.handlers.UndoCurrentProject)
		api.POST("/projects/:projectId/select", s.handlers.SelectProject)
		api.GET("/projects/:projectId", s.handlers.GetProjectDetail)
		api.DELETE("/projects/:projectId", s.handlers.DeleteProject)

		// 导入相关
		api.POST("/import/upload", s.handlers.UploadFile)
		api.GET("/import/:fileId/columns", s.handlers.GetColumns)
		api.POST("/import/:fileId/mapping", s.handlers.SetMapping)
		api.POST("/import/:fileId/execute", s.handlers.ExecuteImport)

		// 企业数据
		api.GET("/companies", s.handlers.ListCompanies)
		api.GET("/companies/:id", s.handlers.GetCompany)
		api.PATCH("/companies/:id", s.handlers.UpdateCompany)
		api.PATCH("/companies/batch", s.handlers.BatchUpdateCompanies)
		api.POST("/companies/reset", s.handlers.ResetCompanies)

		// 指标
		api.GET("/indicators", s.handlers.GetIndicators)
		api.POST("/indicators/adjust", s.handlers.AdjustIndicator)

		// 智能调整
		api.POST("/optimize", s.handlers.Optimize)
		api.POST("/optimize/preview", s.handlers.PreviewOptimize)

		// 配置
		api.GET("/config", s.handlers.GetConfig)
		api.PATCH("/config", s.handlers.UpdateConfig)

		// 导出
		api.POST("/export", s.handlers.Export)
		api.GET("/export/download/:exportId", s.handlers.Download)
	}

	// 静态资源
	if devMode {
		// 开发模式：代理到前端开发服务器
		s.router.NoRoute(func(c *gin.Context) {
			c.Redirect(http.StatusTemporaryRedirect, "http://localhost:5173"+c.Request.URL.Path)
		})
	} else {
		// 生产模式：使用embed的静态资源
		sub, _ := fs.Sub(staticFiles, "dist")

		// 静态资源 - assets 目录
		assetsSub, _ := fs.Sub(sub, "assets")
		s.router.StaticFS("/assets", http.FS(assetsSub))

		// favicon
		s.router.GET("/favicon.svg", func(c *gin.Context) {
			data, err := fs.ReadFile(sub, "favicon.svg")
			if err != nil {
				c.Status(http.StatusNotFound)
				return
			}
			c.Data(http.StatusOK, "image/svg+xml", data)
		})

		// 首页
		s.router.GET("/", func(c *gin.Context) {
			data, _ := fs.ReadFile(sub, "index.html")
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		})

		// SPA 路由 fallback
		s.router.NoRoute(func(c *gin.Context) {
			data, _ := fs.ReadFile(sub, "index.html")
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		})
	}
}

// Run 启动服务器
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

// SaveNow 立即持久化当前项目（用于退出前防止数据丢失）
func (s *Server) SaveNow() error {
	if s.projects == nil {
		return nil
	}
	return s.projects.SaveNow()
}

// GetStore 获取存储（用于测试）
func (s *Server) GetStore() *store.MemoryStore {
	return s.store
}

// GetEngine 获取计算引擎（用于测试）
func (s *Server) GetEngine() *calculator.Engine {
	return s.engine
}
