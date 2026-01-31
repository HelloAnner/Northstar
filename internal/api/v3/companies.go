package v3

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"northstar/internal/calculator"
	"northstar/internal/model"
	"northstar/internal/store"
)

type companyRow struct {
	ID           string `json:"id"`   // wr:123 / ac:456
	Kind         string `json:"kind"` // wr / ac
	CreditCode   string `json:"creditCode"`
	Name         string `json:"name"`
	IndustryCode string `json:"industryCode"`
	IndustryType string `json:"industryType"`
	CompanyScale int    `json:"companyScale"`
	IsSmallMicro int    `json:"isSmallMicro"`
	IsEatWearUse int    `json:"isEatWearUse"`
	RowNo        int    `json:"rowNo"`
	SourceSheet  string `json:"sourceSheet"`

	// WR
	SalesPrevMonth              *float64 `json:"salesPrevMonth,omitempty"`
	SalesCurrentMonth           *float64 `json:"salesCurrentMonth,omitempty"`
	SalesLastYearMonth          *float64 `json:"salesLastYearMonth,omitempty"`
	SalesCurrentCumulative      *float64 `json:"salesCurrentCumulative,omitempty"`
	SalesLastYearCumulative     *float64 `json:"salesLastYearCumulative,omitempty"`
	SalesMonthRate              *float64 `json:"salesMonthRate,omitempty"`
	SalesCumulativeRate         *float64 `json:"salesCumulativeRate,omitempty"`

	RetailCurrentMonth           *float64 `json:"retailCurrentMonth,omitempty"`
	RetailLastYearMonth          *float64 `json:"retailLastYearMonth,omitempty"`
	RetailPrevMonth              *float64 `json:"retailPrevMonth,omitempty"`
	RetailCurrentCumulative      *float64 `json:"retailCurrentCumulative,omitempty"`
	RetailLastYearCumulative     *float64 `json:"retailLastYearCumulative,omitempty"`
	RetailMonthRate              *float64 `json:"retailMonthRate,omitempty"`
	RetailCumulativeRate         *float64 `json:"retailCumulativeRate,omitempty"`
	RetailRatio                  *float64 `json:"retailRatio,omitempty"`

	// AC
	RevenuePrevMonth          *float64 `json:"revenuePrevMonth,omitempty"`
	RevenueCurrentMonth       *float64 `json:"revenueCurrentMonth,omitempty"`
	RevenueLastYearMonth      *float64 `json:"revenueLastYearMonth,omitempty"`
	RevenueCurrentCumulative  *float64 `json:"revenueCurrentCumulative,omitempty"`
	RevenueLastYearCumulative *float64 `json:"revenueLastYearCumulative,omitempty"`
	RevenueMonthRate          *float64 `json:"revenueMonthRate,omitempty"`
	RevenueCumulativeRate     *float64 `json:"revenueCumulativeRate,omitempty"`

	RoomPrevMonth          *float64 `json:"roomPrevMonth,omitempty"`
	RoomCurrentMonth       *float64 `json:"roomCurrentMonth,omitempty"`
	RoomLastYearMonth      *float64 `json:"roomLastYearMonth,omitempty"`
	RoomCurrentCumulative  *float64 `json:"roomCurrentCumulative,omitempty"`
	RoomLastYearCumulative *float64 `json:"roomLastYearCumulative,omitempty"`

	FoodPrevMonth          *float64 `json:"foodPrevMonth,omitempty"`
	FoodCurrentMonth       *float64 `json:"foodCurrentMonth,omitempty"`
	FoodLastYearMonth      *float64 `json:"foodLastYearMonth,omitempty"`
	FoodCurrentCumulative  *float64 `json:"foodCurrentCumulative,omitempty"`
	FoodLastYearCumulative *float64 `json:"foodLastYearCumulative,omitempty"`

	GoodsPrevMonth          *float64 `json:"goodsPrevMonth,omitempty"`
	GoodsCurrentMonth       *float64 `json:"goodsCurrentMonth,omitempty"`
	GoodsLastYearMonth      *float64 `json:"goodsLastYearMonth,omitempty"`
	GoodsCurrentCumulative  *float64 `json:"goodsCurrentCumulative,omitempty"`
	GoodsLastYearCumulative *float64 `json:"goodsLastYearCumulative,omitempty"`
}

type listCompaniesResponse struct {
	Items    []companyRow `json:"items"`
	Total    int          `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

// ListCompanies 查询企业列表（合并批零/住餐）
// GET /api/companies
func (h *Handler) ListCompanies(c *gin.Context) {
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"items": []companyRow{}, "total": 0, "page": 1, "pageSize": 0})
		return
	}

	industryType := strings.TrimSpace(c.Query("industryType"))
	keyword := strings.TrimSpace(c.Query("keyword"))

	page := parseIntWithDefault(c.Query("page"), 1)
	pageSize := parseIntWithDefault(c.Query("pageSize"), 200)
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 200
	}
	if pageSize > 2000 {
		pageSize = 2000
	}

	items, err := h.loadCompanies(year, month, industryType, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	total := len(items)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	c.JSON(http.StatusOK, listCompaniesResponse{
		Items:    items[start:end],
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetCompany 获取企业详情
// GET /api/companies/:id
func (h *Handler) GetCompany(c *gin.Context) {
	id := c.Param("id")
	kind, numericID, ok := parseCompanyID(id)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	switch kind {
	case "wr":
		rec, err := h.store.GetWRByID(numericID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, toCompanyRowWR(*rec))
	case "ac":
		rec, err := h.store.GetACByID(numericID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, toCompanyRowAC(*rec))
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
	}
}

// UpdateCompany 更新企业数据（微调后联动计算）
// PATCH /api/companies/:id
func (h *Handler) UpdateCompany(c *gin.Context) {
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "system not initialized"})
		return
	}

	id := c.Param("id")
	kind, numericID, ok := parseCompanyID(id)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var patch map[string]interface{}
	if err := c.BindJSON(&patch); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	switch kind {
	case "wr":
		existing, err := h.store.GetWRByID(numericID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		updates := pickWRUpdates(patch)
		if rateUpdates, err := buildWRRateDrivenUpdates(*existing, patch); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else {
			for k, v := range rateUpdates {
				updates[k] = v
			}
		}

		if err := h.store.UpdateWR(numericID, updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := recalcDerivedFields(h.store, year, month); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rec, err := h.store.GetWRByID(numericID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		groups, _ := calculator.NewCalculator(h.store).CalculateAll(year, month)
		roundIndicatorGroupsInPlace(groups)
		c.JSON(http.StatusOK, gin.H{"company": toCompanyRowWR(*rec), "groups": groups})
	case "ac":
		existing, err := h.store.GetACByID(numericID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		updates := pickACUpdates(patch)
		if rateUpdates, err := buildACRateDrivenUpdates(*existing, patch); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		} else {
			for k, v := range rateUpdates {
				updates[k] = v
			}
		}

		if err := h.store.UpdateAC(numericID, updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := recalcDerivedFields(h.store, year, month); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rec, err := h.store.GetACByID(numericID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		groups, _ := calculator.NewCalculator(h.store).CalculateAll(year, month)
		roundIndicatorGroupsInPlace(groups)
		c.JSON(http.StatusOK, gin.H{"company": toCompanyRowAC(*rec), "groups": groups})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
	}
}

func buildWRRateDrivenUpdates(existing model.WholesaleRetail, patch map[string]interface{}) (map[string]interface{}, error) {
	out := map[string]interface{}{}

	if rate, set, ok := parseOptionalFloatFromPatch(patch, "salesMonthRate", "sales_month_rate"); ok && set {
		current := pickFloatOverride(existing.SalesCurrentMonth, patch, "salesCurrentMonth", "sales_current_month")
		lastYear := pickFloatOverride(existing.SalesLastYearMonth, patch, "salesLastYearMonth", "sales_last_year_month")
		currentLocked := patchHasKey(patch, "salesCurrentMonth", "sales_current_month")
		lastYearLocked := patchHasKey(patch, "salesLastYearMonth", "sales_last_year_month")
		if err := applyRateDrivenPair(out, rate, current, lastYear, "sales_current_month", "sales_last_year_month", "sales_month_rate", currentLocked, lastYearLocked); err != nil {
			return nil, err
		}
	}
	if rate, set, ok := parseOptionalFloatFromPatch(patch, "salesCumulativeRate", "sales_cumulative_rate"); ok && set {
		current := pickFloatOverride(existing.SalesCurrentCumulative, patch, "salesCurrentCumulative", "sales_current_cumulative")
		lastYear := pickFloatOverride(existing.SalesLastYearCumulative, patch, "salesLastYearCumulative", "sales_last_year_cumulative")
		currentLocked := patchHasKey(patch, "salesCurrentCumulative", "sales_current_cumulative")
		lastYearLocked := patchHasKey(patch, "salesLastYearCumulative", "sales_last_year_cumulative")
		if err := applyRateDrivenPair(out, rate, current, lastYear, "sales_current_cumulative", "sales_last_year_cumulative", "sales_cumulative_rate", currentLocked, lastYearLocked); err != nil {
			return nil, err
		}
	}
	if rate, set, ok := parseOptionalFloatFromPatch(patch, "retailMonthRate", "retail_month_rate"); ok && set {
		current := pickFloatOverride(existing.RetailCurrentMonth, patch, "retailCurrentMonth", "retail_current_month")
		lastYear := pickFloatOverride(existing.RetailLastYearMonth, patch, "retailLastYearMonth", "retail_last_year_month")
		currentLocked := patchHasKey(patch, "retailCurrentMonth", "retail_current_month")
		lastYearLocked := patchHasKey(patch, "retailLastYearMonth", "retail_last_year_month")
		if err := applyRateDrivenPair(out, rate, current, lastYear, "retail_current_month", "retail_last_year_month", "retail_month_rate", currentLocked, lastYearLocked); err != nil {
			return nil, err
		}
	}
	if rate, set, ok := parseOptionalFloatFromPatch(patch, "retailCumulativeRate", "retail_cumulative_rate"); ok && set {
		current := pickFloatOverride(existing.RetailCurrentCumulative, patch, "retailCurrentCumulative", "retail_current_cumulative")
		lastYear := pickFloatOverride(existing.RetailLastYearCumulative, patch, "retailLastYearCumulative", "retail_last_year_cumulative")
		currentLocked := patchHasKey(patch, "retailCurrentCumulative", "retail_current_cumulative")
		lastYearLocked := patchHasKey(patch, "retailLastYearCumulative", "retail_last_year_cumulative")
		if err := applyRateDrivenPair(out, rate, current, lastYear, "retail_current_cumulative", "retail_last_year_cumulative", "retail_cumulative_rate", currentLocked, lastYearLocked); err != nil {
			return nil, err
		}
	}

	return out, nil
}

func buildACRateDrivenUpdates(existing model.AccommodationCatering, patch map[string]interface{}) (map[string]interface{}, error) {
	out := map[string]interface{}{}

	if rate, set, ok := parseOptionalFloatFromPatch(patch, "revenueMonthRate", "revenue_month_rate"); ok && set {
		current := pickFloatOverride(existing.RevenueCurrentMonth, patch, "revenueCurrentMonth", "revenue_current_month")
		lastYear := pickFloatOverride(existing.RevenueLastYearMonth, patch, "revenueLastYearMonth", "revenue_last_year_month")
		currentLocked := patchHasKey(patch, "revenueCurrentMonth", "revenue_current_month")
		lastYearLocked := patchHasKey(patch, "revenueLastYearMonth", "revenue_last_year_month")
		if err := applyRateDrivenPair(out, rate, current, lastYear, "revenue_current_month", "revenue_last_year_month", "revenue_month_rate", currentLocked, lastYearLocked); err != nil {
			return nil, err
		}
	}
	if rate, set, ok := parseOptionalFloatFromPatch(patch, "revenueCumulativeRate", "revenue_cumulative_rate"); ok && set {
		current := pickFloatOverride(existing.RevenueCurrentCumulative, patch, "revenueCurrentCumulative", "revenue_current_cumulative")
		lastYear := pickFloatOverride(existing.RevenueLastYearCumulative, patch, "revenueLastYearCumulative", "revenue_last_year_cumulative")
		currentLocked := patchHasKey(patch, "revenueCurrentCumulative", "revenue_current_cumulative")
		lastYearLocked := patchHasKey(patch, "revenueLastYearCumulative", "revenue_last_year_cumulative")
		if err := applyRateDrivenPair(out, rate, current, lastYear, "revenue_current_cumulative", "revenue_last_year_cumulative", "revenue_cumulative_rate", currentLocked, lastYearLocked); err != nil {
			return nil, err
		}
	}

	retailUpdates, err := buildACRetailDrivenUpdates(existing, patch)
	if err != nil {
		return nil, err
	}
	for k, v := range retailUpdates {
		out[k] = v
	}

	return out, nil
}

func applyRateDrivenPair(
	updates map[string]interface{},
	rate float64,
	currentValue float64,
	lastYearValue float64,
	currentField string,
	lastYearField string,
	rateField string,
	currentLocked bool,
	lastYearLocked bool,
) error {
	// 按公式：rate = (current - lastYear) / lastYear * 100
	factor := 1 + rate/100.0

	if currentLocked && lastYearLocked {
		return fmt.Errorf("无法根据增速回算：%s 与 %s 同时被锁定", currentField, lastYearField)
	}

	if !currentLocked && lastYearValue != 0 {
		updates[currentField] = math.Round(lastYearValue * factor)
		updates[rateField] = nil
		return nil
	}
	if !lastYearLocked && currentValue != 0 && factor != 0 {
		updates[lastYearField] = math.Round(currentValue / factor)
		updates[rateField] = nil
		return nil
	}

	return fmt.Errorf("无法根据增速回算：缺少 %s/%s 的基础数据", currentField, lastYearField)
}

func buildACRetailDrivenUpdates(existing model.AccommodationCatering, patch map[string]interface{}) (map[string]interface{}, error) {
	out := map[string]interface{}{}

	food := pickFloatOverride(existing.FoodCurrentMonth, patch, "foodCurrentMonth", "food_current_month")
	goods := pickFloatOverride(existing.GoodsCurrentMonth, patch, "goodsCurrentMonth", "goods_current_month")
	desiredRetail, desiredSet := parseFloatFromPatch(patch, "retailCurrentMonth", "retail_current_month")

	foodLocked := patchHasKey(patch, "foodCurrentMonth", "food_current_month")
	goodsLocked := patchHasKey(patch, "goodsCurrentMonth", "goods_current_month")
	retailLocked := patchHasKey(patch, "retailCurrentMonth", "retail_current_month")

	if retailLocked && desiredSet {
		// 默认优先改 goods，保留 food；如果 goods 被锁，则改 food
		if !goodsLocked {
			out["goods_current_month"] = desiredRetail - food
			goods = desiredRetail - food
		} else if !foodLocked {
			out["food_current_month"] = desiredRetail - goods
			food = desiredRetail - goods
		} else {
			return nil, fmt.Errorf("无法根据零售额回算：food_current_month 与 goods_current_month 均被锁定")
		}
	}

	// 只要 food/goods/retail 任一被修改，就重算并写回 retail_current_month
	if foodLocked || goodsLocked || retailLocked {
		out["retail_current_month"] = food + goods
	}

	foodLY := pickFloatOverride(existing.FoodLastYearMonth, patch, "foodLastYearMonth", "food_last_year_month")
	goodsLY := pickFloatOverride(existing.GoodsLastYearMonth, patch, "goodsLastYearMonth", "goods_last_year_month")
	desiredRetailLY, desiredLYSet := parseFloatFromPatch(patch, "retailLastYearMonth", "retail_last_year_month")

	foodLYLocked := patchHasKey(patch, "foodLastYearMonth", "food_last_year_month")
	goodsLYLocked := patchHasKey(patch, "goodsLastYearMonth", "goods_last_year_month")
	retailLYLocked := patchHasKey(patch, "retailLastYearMonth", "retail_last_year_month")

	if retailLYLocked && desiredLYSet {
		if !goodsLYLocked {
			out["goods_last_year_month"] = desiredRetailLY - foodLY
			goodsLY = desiredRetailLY - foodLY
		} else if !foodLYLocked {
			out["food_last_year_month"] = desiredRetailLY - goodsLY
			foodLY = desiredRetailLY - goodsLY
		} else {
			return nil, fmt.Errorf("无法根据上年零售额回算：food_last_year_month 与 goods_last_year_month 均被锁定")
		}
	}

	if foodLYLocked || goodsLYLocked || retailLYLocked {
		out["retail_last_year_month"] = foodLY + goodsLY
	}

	return out, nil
}

func pickFloatOverride(existing float64, patch map[string]interface{}, keys ...string) float64 {
	if v, ok := parseFloatFromPatch(patch, keys...); ok {
		return v
	}
	return existing
}

func patchHasKey(patch map[string]interface{}, keys ...string) bool {
	for _, key := range keys {
		if _, ok := patch[key]; ok {
			return true
		}
	}
	return false
}

// parseOptionalFloatFromPatch 返回 (value, isSet, ok)
// - ok=false: patch 不包含任何 key
// - ok=true & isSet=false: key 存在但为 null（表示清空）
// - ok=true & isSet=true: 正常数值
func parseOptionalFloatFromPatch(patch map[string]interface{}, keys ...string) (value float64, isSet bool, ok bool) {
	for _, key := range keys {
		v, exists := patch[key]
		if !exists {
			continue
		}
		if v == nil {
			return 0, false, true
		}
		switch vv := v.(type) {
		case float64:
			return vv, true, true
		case int:
			return float64(vv), true, true
		}
		return 0, false, true
	}
	return 0, false, false
}

func parseFloatFromPatch(patch map[string]interface{}, keys ...string) (float64, bool) {
	for _, key := range keys {
		v, exists := patch[key]
		if !exists || v == nil {
			continue
		}
		switch vv := v.(type) {
		case float64:
			return vv, true
		case int:
			return float64(vv), true
		}
	}
	return 0, false
}

// ResetCompanies 重置企业数据到导入原始值
// POST /api/companies/reset
func (h *Handler) ResetCompanies(c *gin.Context) {
	year, month, err := h.store.GetCurrentYearMonth()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "system not initialized"})
		return
	}

	var body struct {
		CompanyIDs []string `json:"companyIds"`
	}
	_ = c.BindJSON(&body)

	if len(body.CompanyIDs) == 0 {
		if err := resetAllForMonth(h.store, year, month); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		if err := resetByIDs(h.store, body.CompanyIDs); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	_ = recalcDerivedFields(h.store, year, month)
	groups, _ := calculator.NewCalculator(h.store).CalculateAll(year, month)
	roundIndicatorGroupsInPlace(groups)
	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

func (h *Handler) loadCompanies(year, month int, industryType, keyword string) ([]companyRow, error) {
	var wrRows []*model.WholesaleRetail
	var acRows []*model.AccommodationCatering

	if industryType == "" || industryType == "all" || industryType == "wholesale" || industryType == "retail" {
		var tptr *string
		if industryType == "wholesale" || industryType == "retail" {
			tptr = &industryType
		}
		opts := store.WRQueryOptions{DataYear: &year, DataMonth: &month, IndustryType: tptr}
		rows, err := h.store.GetWRByYearMonth(opts)
		if err != nil {
			return nil, err
		}
		wrRows = rows
	}
	if industryType == "" || industryType == "all" || industryType == "accommodation" || industryType == "catering" {
		var tptr *string
		if industryType == "accommodation" || industryType == "catering" {
			tptr = &industryType
		}
		opts := store.ACQueryOptions{DataYear: &year, DataMonth: &month, IndustryType: tptr}
		rows, err := h.store.GetACByYearMonth(opts)
		if err != nil {
			return nil, err
		}
		acRows = rows
	}

	items := make([]companyRow, 0, len(wrRows)+len(acRows))
	for _, r := range wrRows {
		if keyword != "" && !strings.Contains(r.Name, keyword) && !strings.Contains(r.CreditCode, keyword) {
			continue
		}
		items = append(items, toCompanyRowWR(*r))
	}
	for _, r := range acRows {
		if keyword != "" && !strings.Contains(r.Name, keyword) && !strings.Contains(r.CreditCode, keyword) {
			continue
		}
		items = append(items, toCompanyRowAC(*r))
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].IndustryType != items[j].IndustryType {
			return items[i].IndustryType < items[j].IndustryType
		}
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func toCompanyRowWR(r model.WholesaleRetail) companyRow {
	out := companyRow{
		ID:           fmt.Sprintf("wr:%d", r.ID),
		Kind:         "wr",
		CreditCode:   r.CreditCode,
		Name:         r.Name,
		IndustryCode: r.IndustryCode,
		IndustryType: r.IndustryType,
		CompanyScale: r.CompanyScale,
		IsSmallMicro: r.IsSmallMicro,
		IsEatWearUse: r.IsEatWearUse,
		RowNo:        r.RowNo,
		SourceSheet:  r.SourceSheet,
	}

	out.SalesPrevMonth = floatPtr(r.SalesPrevMonth)
	out.SalesCurrentMonth = floatPtr(r.SalesCurrentMonth)
	out.SalesLastYearMonth = floatPtr(r.SalesLastYearMonth)
	out.SalesCurrentCumulative = floatPtr(r.SalesCurrentCumulative)
	out.SalesLastYearCumulative = floatPtr(r.SalesLastYearCumulative)
	out.SalesMonthRate = floatPtrNullable(r.SalesMonthRate)
	out.SalesCumulativeRate = floatPtrNullable(r.SalesCumulativeRate)

	out.RetailCurrentMonth = floatPtr(r.RetailCurrentMonth)
	out.RetailLastYearMonth = floatPtr(r.RetailLastYearMonth)
	out.RetailPrevMonth = floatPtr(r.RetailPrevMonth)
	out.RetailCurrentCumulative = floatPtr(r.RetailCurrentCumulative)
	out.RetailLastYearCumulative = floatPtr(r.RetailLastYearCumulative)
	out.RetailMonthRate = floatPtrNullable(r.RetailMonthRate)
	out.RetailCumulativeRate = floatPtrNullable(r.RetailCumulativeRate)
	out.RetailRatio = floatPtrNullable(r.RetailRatio)
	return out
}

func toCompanyRowAC(r model.AccommodationCatering) companyRow {
	out := companyRow{
		ID:           fmt.Sprintf("ac:%d", r.ID),
		Kind:         "ac",
		CreditCode:   r.CreditCode,
		Name:         r.Name,
		IndustryCode: r.IndustryCode,
		IndustryType: r.IndustryType,
		CompanyScale: r.CompanyScale,
		IsSmallMicro: r.IsSmallMicro,
		IsEatWearUse: r.IsEatWearUse,
		RowNo:        r.RowNo,
		SourceSheet:  r.SourceSheet,
	}

	out.RevenuePrevMonth = floatPtr(r.RevenuePrevMonth)
	out.RevenueCurrentMonth = floatPtr(r.RevenueCurrentMonth)
	out.RevenueLastYearMonth = floatPtr(r.RevenueLastYearMonth)
	out.RevenueCurrentCumulative = floatPtr(r.RevenueCurrentCumulative)
	out.RevenueLastYearCumulative = floatPtr(r.RevenueLastYearCumulative)
	out.RevenueMonthRate = floatPtrNullable(r.RevenueMonthRate)
	out.RevenueCumulativeRate = floatPtrNullable(r.RevenueCumulativeRate)

	out.RoomPrevMonth = floatPtr(r.RoomPrevMonth)
	out.RoomCurrentMonth = floatPtr(r.RoomCurrentMonth)
	out.RoomLastYearMonth = floatPtr(r.RoomLastYearMonth)
	out.RoomCurrentCumulative = floatPtr(r.RoomCurrentCumulative)
	out.RoomLastYearCumulative = floatPtr(r.RoomLastYearCumulative)

	out.FoodPrevMonth = floatPtr(r.FoodPrevMonth)
	out.FoodCurrentMonth = floatPtr(r.FoodCurrentMonth)
	out.FoodLastYearMonth = floatPtr(r.FoodLastYearMonth)
	out.FoodCurrentCumulative = floatPtr(r.FoodCurrentCumulative)
	out.FoodLastYearCumulative = floatPtr(r.FoodLastYearCumulative)

	out.GoodsPrevMonth = floatPtr(r.GoodsPrevMonth)
	out.GoodsCurrentMonth = floatPtr(r.GoodsCurrentMonth)
	out.GoodsLastYearMonth = floatPtr(r.GoodsLastYearMonth)
	out.GoodsCurrentCumulative = floatPtr(r.GoodsCurrentCumulative)
	out.GoodsLastYearCumulative = floatPtr(r.GoodsLastYearCumulative)

	out.RetailCurrentMonth = floatPtr(r.RetailCurrentMonth)
	out.RetailLastYearMonth = floatPtr(r.RetailLastYearMonth)
	return out
}

func floatPtr(v float64) *float64 {
	val := math.Round(v)
	return &val
}

func floatPtrNullable(v *float64) *float64 {
	if v == nil {
		return nil
	}
	val := math.Round(*v)
	return &val
}

func parseCompanyID(id string) (kind string, numericID int64, ok bool) {
	parts := strings.Split(id, ":")
	if len(parts) != 2 {
		return "", 0, false
	}
	kind = parts[0]
	n, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || n <= 0 {
		return "", 0, false
	}
	if kind != "wr" && kind != "ac" {
		return "", 0, false
	}
	return kind, n, true
}

func parseIntWithDefault(v string, d int) int {
	if v == "" {
		return d
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return d
	}
	return i
}

func pickWRUpdates(patch map[string]interface{}) map[string]interface{} {
	allowed := map[string]bool{
		"sales_current_month":        true,
		"sales_last_year_month":      true,
		"sales_current_cumulative":   true,
		"sales_last_year_cumulative": true,

		"retail_current_month":        true,
		"retail_last_year_month":      true,
		"retail_current_cumulative":   true,
		"retail_last_year_cumulative": true,

		"is_eat_wear_use": true,
		"is_small_micro":  true,
	}
	aliases := map[string]string{
		"salesCurrentMonth":       "sales_current_month",
		"salesLastYearMonth":      "sales_last_year_month",
		"salesCurrentCumulative":  "sales_current_cumulative",
		"salesLastYearCumulative": "sales_last_year_cumulative",

		"retailCurrentMonth":       "retail_current_month",
		"retailLastYearMonth":      "retail_last_year_month",
		"retailCurrentCumulative":  "retail_current_cumulative",
		"retailLastYearCumulative": "retail_last_year_cumulative",

		"isEatWearUse": "is_eat_wear_use",
		"isSmallMicro": "is_small_micro",
	}
	return pickUpdates(patch, allowed, aliases)
}

func pickACUpdates(patch map[string]interface{}) map[string]interface{} {
	allowed := map[string]bool{
		"revenue_current_month":        true,
		"revenue_last_year_month":      true,
		"revenue_current_cumulative":   true,
		"revenue_last_year_cumulative": true,

		"room_current_month":  true,
		"food_current_month":  true,
		"goods_current_month": true,

		"retail_current_month":   true,
		"retail_last_year_month": true,

		"is_eat_wear_use": true,
		"is_small_micro":  true,
	}
	aliases := map[string]string{
		"revenueCurrentMonth":       "revenue_current_month",
		"revenueLastYearMonth":      "revenue_last_year_month",
		"revenueCurrentCumulative":  "revenue_current_cumulative",
		"revenueLastYearCumulative": "revenue_last_year_cumulative",

		"roomCurrentMonth":  "room_current_month",
		"foodCurrentMonth":  "food_current_month",
		"goodsCurrentMonth": "goods_current_month",

		"retailCurrentMonth":  "retail_current_month",
		"retailLastYearMonth": "retail_last_year_month",

		"isEatWearUse": "is_eat_wear_use",
		"isSmallMicro": "is_small_micro",
	}
	return pickUpdates(patch, allowed, aliases)
}

func pickUpdates(patch map[string]interface{}, allowed map[string]bool, aliases map[string]string) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range patch {
		key := strings.TrimSpace(k)
		if !allowed[key] {
			if mapped, ok := aliases[key]; ok {
				key = mapped
			}
		}
		if !allowed[key] {
			continue
		}
		switch vv := v.(type) {
		case float64:
			out[key] = vv
		case int:
			out[key] = vv
		case bool:
			if vv {
				out[key] = 1
			} else {
				out[key] = 0
			}
		}
	}
	return out
}

func recalcDerivedFields(st *store.Store, year, month int) error {
	// 复用导入后的 SQL（一次更新，保证指标一致）
	if err := st.Exec(`
		UPDATE wholesale_retail SET
			sales_month_rate = CASE
				WHEN sales_last_year_month = 0 THEN -100
				ELSE (sales_current_month - sales_last_year_month) / sales_last_year_month * 100
			END,
			sales_cumulative_rate = CASE
				WHEN sales_last_year_cumulative = 0 THEN -100
				ELSE (sales_current_cumulative - sales_last_year_cumulative) / sales_last_year_cumulative * 100
			END,
			retail_month_rate = CASE
				WHEN retail_last_year_month = 0 THEN -100
				ELSE (retail_current_month - retail_last_year_month) / retail_last_year_month * 100
			END,
			retail_cumulative_rate = CASE
				WHEN retail_last_year_cumulative = 0 THEN -100
				ELSE (retail_current_cumulative - retail_last_year_cumulative) / retail_last_year_cumulative * 100
			END,
			retail_ratio = CASE
				WHEN sales_current_month = 0 THEN NULL
				ELSE retail_current_month / sales_current_month * 100
			END
		WHERE data_year = ? AND data_month = ?
	`, year, month); err != nil {
		return err
	}

	if err := st.Exec(`
		UPDATE accommodation_catering SET
			revenue_month_rate = CASE
				WHEN revenue_last_year_month = 0 THEN -100
				ELSE (revenue_current_month - revenue_last_year_month) / revenue_last_year_month * 100
			END,
			revenue_cumulative_rate = CASE
				WHEN revenue_last_year_cumulative = 0 THEN -100
				ELSE (revenue_current_cumulative - revenue_last_year_cumulative) / revenue_last_year_cumulative * 100
			END
		WHERE data_year = ? AND data_month = ?
	`, year, month); err != nil {
		return err
	}
	return nil
}

func resetAllForMonth(st *store.Store, year, month int) error {
	if err := st.Exec(`
		UPDATE wholesale_retail SET
			sales_current_month = COALESCE(original_sales_current_month, sales_current_month),
			retail_current_month = COALESCE(original_retail_current_month, retail_current_month)
		WHERE data_year = ? AND data_month = ?
	`, year, month); err != nil {
		return err
	}
	if err := st.Exec(`
		UPDATE accommodation_catering SET
			revenue_current_month = COALESCE(original_revenue_current_month, revenue_current_month),
			room_current_month = COALESCE(original_room_current_month, room_current_month),
			food_current_month = COALESCE(original_food_current_month, food_current_month),
			goods_current_month = COALESCE(original_goods_current_month, goods_current_month)
		WHERE data_year = ? AND data_month = ?
	`, year, month); err != nil {
		return err
	}
	return nil
}

func resetByIDs(st *store.Store, ids []string) error {
	var wrIDs []int64
	var acIDs []int64
	for _, id := range ids {
		kind, numericID, ok := parseCompanyID(id)
		if !ok {
			continue
		}
		if kind == "wr" {
			wrIDs = append(wrIDs, numericID)
		}
		if kind == "ac" {
			acIDs = append(acIDs, numericID)
		}
	}

	if len(wrIDs) > 0 {
		if err := execInClause(st, `
			UPDATE wholesale_retail SET
				sales_current_month = COALESCE(original_sales_current_month, sales_current_month),
				retail_current_month = COALESCE(original_retail_current_month, retail_current_month)
			WHERE id IN (%s)
		`, wrIDs); err != nil {
			return err
		}
	}
	if len(acIDs) > 0 {
		if err := execInClause(st, `
			UPDATE accommodation_catering SET
				revenue_current_month = COALESCE(original_revenue_current_month, revenue_current_month),
				room_current_month = COALESCE(original_room_current_month, room_current_month),
				food_current_month = COALESCE(original_food_current_month, food_current_month),
				goods_current_month = COALESCE(original_goods_current_month, goods_current_month)
			WHERE id IN (%s)
		`, acIDs); err != nil {
			return err
		}
	}
	return nil
}

func execInClause(st *store.Store, tmpl string, ids []int64) error {
	placeholders := make([]string, 0, len(ids))
	args := make([]interface{}, 0, len(ids))
	for range ids {
		placeholders = append(placeholders, "?")
	}
	for _, id := range ids {
		args = append(args, id)
	}
	q := fmt.Sprintf(tmpl, strings.Join(placeholders, ","))
	return st.Exec(q, args...)
}
