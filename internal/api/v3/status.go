package v3

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"northstar/internal/store"
)

// StatusResponse 系统状态响应
type StatusResponse struct {
	Initialized    bool   `json:"initialized"`     // 是否已初始化（有数据）
	CurrentYear    int    `json:"currentYear"`     // 当前操作年份
	CurrentMonth   int    `json:"currentMonth"`    // 当前操作月份
	TotalCompanies int    `json:"totalCompanies"`  // 企业总数
	WRCount        int    `json:"wrCount"`         // 批零企业数
	ACCount        int    `json:"acCount"`         // 住餐企业数
	LastImportTime string `json:"lastImportTime"`  // 最后导入时间
}

// GetStatus 获取系统状态
// GET /api/status
func (h *Handler) GetStatus(c *gin.Context) {
	// 获取当前年月配置
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		c.JSON(http.StatusOK, StatusResponse{
			Initialized: false,
		})
		return
	}

	// 统计企业数量
	wrOpts := store.WRQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	}
	wrCount, err := h.store.CountWR(wrOpts)
	if err != nil {
		wrCount = 0
	}

	acOpts := store.ACQueryOptions{
		DataYear:  &year,
		DataMonth: &month,
	}
	acCount, err := h.store.CountAC(acOpts)
	if err != nil {
		acCount = 0
	}

	totalCompanies := wrCount + acCount

	// 判断是否已初始化
	initialized := totalCompanies > 0

	c.JSON(http.StatusOK, StatusResponse{
		Initialized:    initialized,
		CurrentYear:    year,
		CurrentMonth:   month,
		TotalCompanies: totalCompanies,
		WRCount:        wrCount,
		ACCount:        acCount,
		LastImportTime: "", // TODO: 从 import_logs 获取最后导入时间
	})
}
