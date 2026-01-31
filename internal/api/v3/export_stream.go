package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"northstar/internal/exporter"
)

type exportProgressEvent struct {
	Type      string      `json:"type"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// ExportStream 导出 Excel（SSE 进度 + 完成后提供下载地址）
// POST /api/export/stream
func (h *Handler) ExportStream(c *gin.Context) {
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取当前年月失败"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "不支持流式响应"})
		return
	}

	send := func(event exportProgressEvent) {
		b, err := json.Marshal(event)
		if err != nil {
			return
		}
		fmt.Fprintf(c.Writer, "data: %s\n\n", b)
		flusher.Flush()
	}

	send(exportProgressEvent{
		Type:    "start",
		Message: "开始导出",
		Data: map[string]any{
			"year":  year,
			"month": month,
		},
		Timestamp: time.Now(),
	})

	exp := exporter.NewExporter(h.store, h.templatePath)

	lastPercent := -1
	progressFn := func(p exporter.ProgressEvent) {
		if p.Percent == lastPercent {
			return
		}
		lastPercent = p.Percent
		send(exportProgressEvent{
			Type:      "progress",
			Message:   p.Stage,
			Data:      map[string]any{"percent": p.Percent},
			Timestamp: time.Now(),
		})
	}

	file, err := exp.Export(exporter.ExportOptions{
		Year:     year,
		Month:    month,
		Progress: progressFn,
	})
	if err != nil {
		send(exportProgressEvent{
			Type:      "error",
			Message:   "导出失败: " + err.Error(),
			Data:      map[string]any{},
			Timestamp: time.Now(),
		})
		return
	}
	defer file.Close()

	tempPath := filepath.Join(os.TempDir(), fmt.Sprintf("northstar_export_%d_%d.xlsx", time.Now().UnixNano(), os.Getpid()))
	if err := file.SaveAs(tempPath); err != nil {
		send(exportProgressEvent{
			Type:      "error",
			Message:   "写入导出文件失败: " + err.Error(),
			Data:      map[string]any{},
			Timestamp: time.Now(),
		})
		_ = os.Remove(tempPath)
		return
	}

	token := h.downloads.put(tempPath, year, month, 10*time.Minute)
	prefix := "/api"
	if strings.HasPrefix(c.Request.URL.Path, "/api/v1/") {
		prefix = "/api/v1"
	}
	downloadURL := fmt.Sprintf("%s/export/download/%s", prefix, token)

	send(exportProgressEvent{
		Type:    "done",
		Message: "导出完成",
		Data: map[string]any{
			"percent":     100,
			"downloadUrl": downloadURL,
		},
		Timestamp: time.Now(),
	})
}

// DownloadExport 下载导出的 Excel 文件（一次性）
// GET /api/export/download/:token
func (h *Handler) DownloadExport(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 token"})
		return
	}

	item, ok := h.downloads.get(token)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "下载链接已失效"})
		return
	}

	if _, err := os.Stat(item.filePath); err != nil {
		h.downloads.delete(token)
		c.JSON(http.StatusNotFound, gin.H{"error": "导出文件不存在"})
		return
	}

	c.Header("Content-Disposition", buildExportContentDisposition(item.year, item.month))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.File(item.filePath)

	h.downloads.delete(token)
	_ = os.Remove(item.filePath)
}
