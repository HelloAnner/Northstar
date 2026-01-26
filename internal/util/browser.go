package util

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser 打开默认浏览器
// 支持 Windows 7/10/11, macOS, Linux
func OpenBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Windows 7+ 兼容方式：使用 rundll32 调用 url.dll
		// 这比 cmd /c start 更稳定，特别是在 Windows 7 上
		// 同时也支持 Windows 10/11
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		// Linux: 尝试多种方式打开浏览器
		// 优先使用 xdg-open，如果失败则尝试其他方式
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}

// OpenBrowserWithFallback 带降级方案的浏览器打开
// 如果主要方式失败，会尝试备选方式
func OpenBrowserWithFallback(url string) error {
	err := OpenBrowser(url)
	if err == nil {
		return nil
	}

	// 降级方案
	switch runtime.GOOS {
	case "windows":
		// 备选方案：使用 explorer
		return exec.Command("explorer", url).Start()
	case "linux":
		// 尝试常见浏览器
		browsers := []string{"google-chrome", "firefox", "chromium-browser", "sensible-browser"}
		for _, browser := range browsers {
			if err := exec.Command(browser, url).Start(); err == nil {
				return nil
			}
		}
	}

	return err
}

// FindAvailablePort 查找可用端口
func FindAvailablePort(startPort int) int {
	// 简单实现：返回起始端口
	// 实际应用中应该检测端口是否被占用
	return startPort
}

// FormatPercent 格式化百分比
func FormatPercent(value float64) string {
	sign := ""
	if value > 0 {
		sign = "+"
	}
	return fmt.Sprintf("%s%.2f%%", sign, value*100)
}

// FormatCurrency 格式化货币（千分位）
func FormatCurrency(value float64) string {
	// 简单实现
	return fmt.Sprintf("%.2f", value)
}
