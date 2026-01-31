package v3

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"northstar/internal/calculator"
	"northstar/internal/exporter"
)

// GetIndicators 获取16项指标
// GET /api/indicators
func (h *Handler) GetIndicators(c *gin.Context) {
	// 获取当前年月
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取当前年月失败"})
		return
	}

	// 创建计算器
	calc := calculator.NewCalculator(h.store)

	// 计算所有指标
	groups, err := calc.CalculateAll(year, month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "计算指标失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"year":   year,
		"month":  month,
		"groups": groups,
	})
}

// Export 导出 Excel
// POST /api/export
func (h *Handler) Export(c *gin.Context) {
	// 获取当前年月
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取当前年月失败"})
		return
	}

	// 创建导出器
	exp := exporter.NewExporter(h.store, h.templatePath)

	// 导出 Excel
	file, err := exp.Export(exporter.ExportOptions{
		Year:  year,
		Month: month,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "导出失败: " + err.Error()})
		return
	}

	// 设置响应头
	filename := fmt.Sprintf("月报-%d-%02d.xlsx", year, month)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")

	// 写入文件
	if err := file.Write(c.Writer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件失败"})
		return
	}
}
