package server

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"northstar/internal/api/v3"
	"northstar/internal/config"
	"northstar/internal/dagcalc"
	"northstar/internal/store"
)

//go:embed all:dist
var staticFiles embed.FS

//go:embed defaults/rules.md defaults/role.json
var defaultAssets embed.FS

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

	configDir := filepath.Join(dataDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatalf("Failed to initialize config dir: %v", err)
	}
	rulesPath := filepath.Join(configDir, "rules.md")
	rulePath := filepath.Join(configDir, "role.json")
	if err := ensureRulesMarkdown(rulesPath); err != nil {
		log.Fatalf("Failed to initialize rules.md: %v", err)
	}
	if err := ensureRoleJSON(rulePath); err != nil {
		log.Fatalf("Failed to initialize role.json: %v", err)
	}
	if err := ensureRuleManagementConfig(sqliteStore); err != nil {
		log.Fatalf("Failed to initialize rule management config: %v", err)
	}
	engine := dagcalc.NewEngine(dagcalc.NewGraph(), sqliteStore, 0, 0, rulePath)
	if err := engine.ReloadRules(); err != nil {
		log.Printf("reload rules failed: %v", err)
	}

	// 创建 V3 API 处理器
	v3Handler := v3.NewHandlerWithEngine(sqliteStore, cfg.Excel.TemplatePath, engine)
	v3Handler.ConfigureRuleManagement(rulesPath, rulePath)

	s := &Server{
		router: gin.Default(),
		store:  sqliteStore,
		v3:     v3Handler,
	}

	s.setupRoutes(devMode)

	return s
}

func ensureRulesMarkdown(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := defaultAssets.ReadFile("defaults/rules.md")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func ensureRoleJSON(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	data, err := defaultAssets.ReadFile("defaults/role.json")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func ensureRuleManagementConfig(st *store.Store) error {
	defaults := map[string]string{
		"llm_user_prompt":      "",
		"rules_convert_status": "idle",
		"rules_convert_at":     "",
		"rules_convert_error":  "",
	}
	for key, value := range defaults {
		if err := st.EnsureConfig(key, value); err != nil {
			return err
		}
	}
	return nil
}

// setupRoutes 设置路由
func (s *Server) setupRoutes(devMode bool) {
	// CORS
	s.router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// API 路由（前端与测试默认使用 /api/v1；同时保留 /api 兼容旧路径）
	apiV1 := s.router.Group("/api/v1")
	{
		s.v3.RegisterRoutes(apiV1)
	}
	apiCompat := s.router.Group("/api")
	{
		s.v3.RegisterRoutes(apiCompat)
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

// RouterForTest 暴露路由，仅用于测试。
func (s *Server) RouterForTest() http.Handler {
	return s.router
}
