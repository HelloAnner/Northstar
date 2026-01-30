package v3

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ConfigResponse 配置响应
type ConfigResponse struct {
	// 时间配置
	CurrentYear  int `json:"currentYear"`
	CurrentMonth int `json:"currentMonth"`

	// 社零额(定) 手工输入项
	SmallMicroRateMonth  float64 `json:"smallMicroRateMonth"`  // 本月小微增速
	EatWearUseRateMonth  float64 `json:"eatWearUseRateMonth"`  // 本月吃穿用增速
	SampleRateMonth      float64 `json:"sampleRateMonth"`      // 本月抽样单位增速
	SmallMicroRatePrev   float64 `json:"smallMicroRatePrev"`   // 上月小微增速
	EatWearUseRatePrev   float64 `json:"eatWearUseRatePrev"`   // 上月吃穿用增速
	SampleRatePrev       float64 `json:"sampleRatePrev"`       // 上月抽样单位增速
	WeightSmallMicro     float64 `json:"weightSmallMicro"`     // 小微权重
	WeightEatWearUse     float64 `json:"weightEatWearUse"`     // 吃穿用权重
	WeightSample         float64 `json:"weightSample"`         // 抽样权重
	ProvinceLimitBelow   float64 `json:"provinceLimitBelow"`   // 全省限下增速变动量

	// 历史累计社零额
	HistorySocialE18 float64 `json:"historySocialE18"`
	HistorySocialE19 float64 `json:"historySocialE19"`
	HistorySocialE20 float64 `json:"historySocialE20"`
	HistorySocialE21 float64 `json:"historySocialE21"`
	HistorySocialE22 float64 `json:"historySocialE22"`
	HistorySocialE23 float64 `json:"historySocialE23"`

	// 汇总表(定) 输入项
	TotalCompanyCount    int `json:"totalCompanyCount"`    // 单位总数
	ReportedCompanyCount int `json:"reportedCompanyCount"` // 已上报单位数
	NegativeGrowthCount  int `json:"negativeGrowthCount"`  // 负增长企业数

	// 限下社零额
	LastYearLimitBelowCumulative float64 `json:"lastYearLimitBelowCumulative"` // 上年累计限下社零额
}

// UpdateConfigRequest 更新配置请求
type UpdateConfigRequest struct {
	// 使用 map 允许部分更新
	Updates map[string]interface{} `json:"updates"`
}

// GetConfig 获取所有配置
// GET /api/config
func (h *Handler) GetConfig(c *gin.Context) {
	// 获取所有配置
	allConfig, err := h.store.GetAllConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取配置失败"})
		return
	}

	// 辅助函数：安全转换为整数
	getInt := func(key string) int {
		if val, ok := allConfig[key]; ok {
			if i, err := strconv.Atoi(val); err == nil {
				return i
			}
		}
		return 0
	}

	// 辅助函数：安全转换为浮点数
	getFloat := func(key string) float64 {
		if val, ok := allConfig[key]; ok {
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				return f
			}
		}
		return 0
	}

	response := ConfigResponse{
		// 时间配置
		CurrentYear:  getInt("current_year"),
		CurrentMonth: getInt("current_month"),

		// 社零额(定) 手工输入项
		SmallMicroRateMonth: getFloat("small_micro_rate_month"),
		EatWearUseRateMonth: getFloat("eat_wear_use_rate_month"),
		SampleRateMonth:     getFloat("sample_rate_month"),
		SmallMicroRatePrev:  getFloat("small_micro_rate_prev"),
		EatWearUseRatePrev:  getFloat("eat_wear_use_rate_prev"),
		SampleRatePrev:      getFloat("sample_rate_prev"),
		WeightSmallMicro:    getFloat("weight_small_micro"),
		WeightEatWearUse:    getFloat("weight_eat_wear_use"),
		WeightSample:        getFloat("weight_sample"),
		ProvinceLimitBelow:  getFloat("province_limit_below_rate_change"),

		// 历史累计社零额
		HistorySocialE18: getFloat("history_social_e18"),
		HistorySocialE19: getFloat("history_social_e19"),
		HistorySocialE20: getFloat("history_social_e20"),
		HistorySocialE21: getFloat("history_social_e21"),
		HistorySocialE22: getFloat("history_social_e22"),
		HistorySocialE23: getFloat("history_social_e23"),

		// 汇总表(定) 输入项
		TotalCompanyCount:    getInt("total_company_count"),
		ReportedCompanyCount: getInt("reported_company_count"),
		NegativeGrowthCount:  getInt("negative_growth_count"),

		// 限下社零额
		LastYearLimitBelowCumulative: getFloat("last_year_limit_below_cumulative"),
	}

	c.JSON(http.StatusOK, response)
}

// UpdateConfig 更新配置
// PATCH /api/config
func (h *Handler) UpdateConfig(c *gin.Context) {
	var req UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式错误"})
		return
	}

	// 遍历更新项
	for key, value := range req.Updates {
		var strValue string

		switch v := value.(type) {
		case string:
			strValue = v
		case float64:
			strValue = strconv.FormatFloat(v, 'f', -1, 64)
		case int:
			strValue = strconv.Itoa(v)
		case bool:
			if v {
				strValue = "1"
			} else {
				strValue = "0"
			}
		default:
			continue // 跳过不支持的类型
		}

		// 更新配置
		if err := h.store.SetConfig(key, strValue); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "更新配置失败: " + key,
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置更新成功"})
}
