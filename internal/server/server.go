package server

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"northstar/internal/api/v3"
	"northstar/internal/config"
	"northstar/internal/store"
)

//go:embed all:dist
var staticFiles embed.FS

// Server HTTP服务器
type Server struct {
	router *gin.Engine
	store  *store.Store
	v3     *v3.Handler
}

// NewServer 创建服务器
func NewServer(cfg *config.AppConfig) *Server {
	devMode := cfg.Server.DevMode
	if !devMode {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化 SQLite Store
	dataDir, err := config.EnsureDataDir(cfg)
	if err != nil {
		dataDir = cfg.Data.DataDir
	}
	dbPath := filepath.Join(dataDir, "northstar.db")

	sqliteStore, err := store.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 创建 V3 API 处理器
	v3Handler := v3.NewHandler(sqliteStore)

	s := &Server{
		router: gin.Default(),
		store:  sqliteStore,
		v3:     v3Handler,
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

	// V3 API 路由
	api := s.router.Group("/api")
	{
		s.v3.RegisterRoutes(api)
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

// SaveNow 立即持久化（V3 使用 SQLite，自动持久化）
func (s *Server) SaveNow() error {
	return nil
}

// GetStore 获取存储（用于测试）
func (s *Server) GetStore() *store.Store {
	return s.store
}
