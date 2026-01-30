package v3

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"northstar/internal/importer"
)

// ImportRequest 导入请求
type ImportRequest struct {
	ClearExisting  bool `json:"clearExisting"`  // 是否清空现有数据
	UpdateConfigYM bool `json:"updateConfigYM"` // 是否更新当前年月
}

// Import 导入 Excel 数据 (SSE 流式响应)
// POST /api/import
func (h *Handler) Import(c *gin.Context) {
	// 解析 multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的表单数据"})
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未找到上传文件"})
		return
	}

	uploadedFile := files[0]

	// 保存到临时目录
	tempDir := os.TempDir()
	tempFilePath := filepath.Join(tempDir, fmt.Sprintf("northstar_import_%d_%s", time.Now().Unix(), uploadedFile.Filename))

	if err := c.SaveUploadedFile(uploadedFile, tempFilePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件失败"})
		return
	}

	// 清理临时文件
	defer os.Remove(tempFilePath)

	// 解析导入选项
	clearExisting := c.DefaultPostForm("clearExisting", "true") == "true"
	updateConfigYM := c.DefaultPostForm("updateConfigYM", "true") == "true"

	// 设置 SSE 响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 创建导入协调器
	coordinator := importer.NewCoordinator(h.store)

	// 开始导入
	progressChan := coordinator.Import(importer.ImportOptions{
		FilePath:        tempFilePath,
		ClearExisting:   clearExisting,
		UpdateConfigYM:  updateConfigYM,
		CalculateFields: true,
	})

	// 流式发送进度事件
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "不支持流式响应"})
		return
	}

	for event := range progressChan {
		// 序列化事件为 JSON
		eventData, err := json.Marshal(event)
		if err != nil {
			continue
		}

		// SSE 格式: data: {json}\n\n
		fmt.Fprintf(c.Writer, "data: %s\n\n", eventData)
		flusher.Flush()
	}
}
