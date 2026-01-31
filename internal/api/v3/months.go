package v3

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"northstar/internal/calculator"
	"northstar/internal/store"
)

type monthsResponse struct {
	CurrentYear  int                   `json:"currentYear"`
	CurrentMonth int                   `json:"currentMonth"`
	Items        []store.YearMonthStat `json:"items"`
}

// ListMonths 获取可用年月列表
// GET /api/months
func (h *Handler) ListMonths(c *gin.Context) {
	items, err := h.store.ListAvailableYearMonths()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		year = 0
		month = 0
	}

	c.JSON(http.StatusOK, monthsResponse{
		CurrentYear:  year,
		CurrentMonth: month,
		Items:        items,
	})
}

type selectMonthRequest struct {
	Year  int `json:"year"`
	Month int `json:"month"`
}

// SelectMonth 切换当前操作年月（影响：指标/明细/导出）
// POST /api/months/select
func (h *Handler) SelectMonth(c *gin.Context) {
	var req selectMonthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}
	if req.Month < 1 || req.Month > 12 || req.Year <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "非法年月"})
		return
	}

	// 校验该年月是否存在数据（避免切到空月份导致“很多数据都是空的”）
	items, err := h.store.ListAvailableYearMonths()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ok := false
	for _, it := range items {
		if it.Year == req.Year && it.Month == req.Month && it.Total > 0 {
			ok = true
			break
		}
	}
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该月份无可用数据"})
		return
	}

	if err := h.store.SetCurrentYearMonth(req.Year, req.Month); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	wrCount, _ := h.store.CountWR(store.WRQueryOptions{
		DataYear:  &req.Year,
		DataMonth: &req.Month,
	})
	acCount, _ := h.store.CountAC(store.ACQueryOptions{
		DataYear:  &req.Year,
		DataMonth: &req.Month,
	})

	groups, err := calculator.NewCalculator(h.store).CalculateAll(req.Year, req.Month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "计算指标失败"})
		return
	}
	roundIndicatorGroupsInPlace(groups)
	c.JSON(http.StatusOK, gin.H{
		"status": StatusResponse{
			Initialized:    wrCount+acCount > 0,
			CurrentYear:    req.Year,
			CurrentMonth:   req.Month,
			TotalCompanies: wrCount + acCount,
			WRCount:        wrCount,
			ACCount:        acCount,
			LastImportTime: "",
		},
		"groups": groups,
	})
}
