package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"northstar/internal/config"
	"northstar/internal/server"
	"northstar/internal/util"
)

var (
	port    = flag.Int("port", 0, "服务端口 (config.toml 优先；仅当未显式配置 port 时生效)")
	devMode = flag.Bool("dev", false, "开发模式")
	dataDir = flag.String("dataDir", "", "数据目录 (覆盖配置文件)")
)

func main() {
	flag.Parse()

	fmt.Println("==========================================")
	fmt.Println("  Northstar - 经济数据统计分析工具")
	fmt.Println("==========================================")

	// 加载配置
	cfg, info, err := config.LoadConfigWithInfo()
	if err != nil {
		log.Printf("加载配置失败，使用默认配置: %v", err)
		cfg = config.DefaultConfig()
		info = config.LoadConfigInfo{}
	}

	// 命令行参数覆盖配置
	if *port > 0 && !info.PortSpecified {
		cfg.Server.Port = *port
	}
	if *devMode {
		cfg.Server.DevMode = true
	}
	if *dataDir != "" {
		cfg.Data.DataDir = *dataDir
	}

	// 确保数据目录存在
	dataDir, err := config.EnsureDataDir(cfg)
	if err != nil {
		log.Printf("创建数据目录失败: %v", err)
	} else {
		fmt.Printf("数据目录: %s\n", dataDir)
	}

	// 创建服务器
	srv := server.NewServer(cfg)

	// 构建地址
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	url := fmt.Sprintf("http://localhost:%d", cfg.Server.Port)

	// 启动服务器
	go func() {
		fmt.Printf("服务启动中，监听端口 %d ...\n", cfg.Server.Port)
		if err := srv.Run(addr); err != nil {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 打开浏览器
	if !cfg.Server.DevMode {
		fmt.Printf("正在打开浏览器: %s\n", url)
		if err := util.OpenBrowserWithFallback(url); err != nil {
			fmt.Printf("无法自动打开浏览器，请手动访问: %s\n", url)
		}
	} else {
		fmt.Printf("开发模式: 请访问 %s\n", url)
	}

	fmt.Println("\n按 Ctrl+C 停止服务...")

	// 等待信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n正在关闭服务...")
	if err := srv.SaveNow(); err != nil {
		log.Printf("退出前保存失败: %v", err)
	}
}
