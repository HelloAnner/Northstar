package v3

import (
	"fmt"
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
	SalesCurrentMonth      *float64 `json:"salesCurrentMonth,omitempty"`
	SalesLastYearMonth     *float64 `json:"salesLastYearMonth,omitempty"`
	SalesCurrentCumulative *float64 `json:"salesCurrentCumulative,omitempty"`
	SalesLastYearCumulative *float64 `json:"salesLastYearCumulative,omitempty"`
	SalesMonthRate         *float64 `json:"salesMonthRate,omitempty"`
	SalesCumulativeRate    *float64 `json:"salesCumulativeRate,omitempty"`

	RetailCurrentMonth      *float64 `json:"retailCurrentMonth,omitempty"`
	RetailLastYearMonth     *float64 `json:"retailLastYearMonth,omitempty"`
	RetailCurrentCumulative *float64 `json:"retailCurrentCumulative,omitempty"`
	RetailLastYearCumulative *float64 `json:"retailLastYearCumulative,omitempty"`
	RetailMonthRate         *float64 `json:"retailMonthRate,omitempty"`
	RetailCumulativeRate    *float64 `json:"retailCumulativeRate,omitempty"`
	RetailRatio             *float64 `json:"retailRatio,omitempty"`

	// AC
	RevenueCurrentMonth      *float64 `json:"revenueCurrentMonth,omitempty"`
	RevenueLastYearMonth     *float64 `json:"revenueLastYearMonth,omitempty"`
	RevenueCurrentCumulative *float64 `json:"revenueCurrentCumulative,omitempty"`
	RevenueLastYearCumulative *float64 `json:"revenueLastYearCumulative,omitempty"`
	RevenueMonthRate         *float64 `json:"revenueMonthRate,omitempty"`
	RevenueCumulativeRate    *float64 `json:"revenueCumulativeRate,omitempty"`

	RoomCurrentMonth      *float64 `json:"roomCurrentMonth,omitempty"`
	RoomLastYearMonth     *float64 `json:"roomLastYearMonth,omitempty"`
	RoomCurrentCumulative *float64 `json:"roomCurrentCumulative,omitempty"`
	RoomLastYearCumulative *float64 `json:"roomLastYearCumulative,omitempty"`

	FoodCurrentMonth      *float64 `json:"foodCurrentMonth,omitempty"`
	FoodLastYearMonth     *float64 `json:"foodLastYearMonth,omitempty"`
	FoodCurrentCumulative *float64 `json:"foodCurrentCumulative,omitempty"`
	FoodLastYearCumulative *float64 `json:"foodLastYearCumulative,omitempty"`

	GoodsCurrentMonth      *float64 `json:"goodsCurrentMonth,omitempty"`
	GoodsLastYearMonth     *float64 `json:"goodsLastYearMonth,omitempty"`
	GoodsCurrentCumulative *float64 `json:"goodsCurrentCumulative,omitempty"`
	GoodsLastYearCumulative *float64 `json:"goodsLastYearCumulative,omitempty"`
}

type listCompaniesResponse struct {
	Items []companyRow `json:"items"`
	Total int          `json:"total"`
	Page  int          `json:"page"`
	PageSize int       `json:"pageSize"`
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
		Items: items[start:end],
		Total: total,
		Page: page,
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
		updates := pickWRUpdates(patch)
		if err := h.store.UpdateWR(numericID, updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := recalcDerivedFields(h.store, year, month); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rec, _ := h.store.GetWRByID(numericID)
		groups, _ := calculator.NewCalculator(h.store).CalculateAll(year, month)
		c.JSON(http.StatusOK, gin.H{"company": toCompanyRowWR(*rec), "groups": groups})
	case "ac":
		updates := pickACUpdates(patch)
		if err := h.store.UpdateAC(numericID, updates); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := recalcDerivedFields(h.store, year, month); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		rec, _ := h.store.GetACByID(numericID)
		groups, _ := calculator.NewCalculator(h.store).CalculateAll(year, month)
		c.JSON(http.StatusOK, gin.H{"company": toCompanyRowAC(*rec), "groups": groups})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
	}
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

	out.SalesCurrentMonth = floatPtr(r.SalesCurrentMonth)
	out.SalesLastYearMonth = floatPtr(r.SalesLastYearMonth)
	out.SalesCurrentCumulative = floatPtr(r.SalesCurrentCumulative)
	out.SalesLastYearCumulative = floatPtr(r.SalesLastYearCumulative)
	out.SalesMonthRate = r.SalesMonthRate
	out.SalesCumulativeRate = r.SalesCumulativeRate

	out.RetailCurrentMonth = floatPtr(r.RetailCurrentMonth)
	out.RetailLastYearMonth = floatPtr(r.RetailLastYearMonth)
	out.RetailCurrentCumulative = floatPtr(r.RetailCurrentCumulative)
	out.RetailLastYearCumulative = floatPtr(r.RetailLastYearCumulative)
	out.RetailMonthRate = r.RetailMonthRate
	out.RetailCumulativeRate = r.RetailCumulativeRate
	out.RetailRatio = r.RetailRatio
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

	out.RevenueCurrentMonth = floatPtr(r.RevenueCurrentMonth)
	out.RevenueLastYearMonth = floatPtr(r.RevenueLastYearMonth)
	out.RevenueCurrentCumulative = floatPtr(r.RevenueCurrentCumulative)
	out.RevenueLastYearCumulative = floatPtr(r.RevenueLastYearCumulative)
	out.RevenueMonthRate = r.RevenueMonthRate
	out.RevenueCumulativeRate = r.RevenueCumulativeRate

	out.RoomCurrentMonth = floatPtr(r.RoomCurrentMonth)
	out.RoomLastYearMonth = floatPtr(r.RoomLastYearMonth)
	out.RoomCurrentCumulative = floatPtr(r.RoomCurrentCumulative)
	out.RoomLastYearCumulative = floatPtr(r.RoomLastYearCumulative)

	out.FoodCurrentMonth = floatPtr(r.FoodCurrentMonth)
	out.FoodLastYearMonth = floatPtr(r.FoodLastYearMonth)
	out.FoodCurrentCumulative = floatPtr(r.FoodCurrentCumulative)
	out.FoodLastYearCumulative = floatPtr(r.FoodLastYearCumulative)

	out.GoodsCurrentMonth = floatPtr(r.GoodsCurrentMonth)
	out.GoodsLastYearMonth = floatPtr(r.GoodsLastYearMonth)
	out.GoodsCurrentCumulative = floatPtr(r.GoodsCurrentCumulative)
	out.GoodsLastYearCumulative = floatPtr(r.GoodsLastYearCumulative)

	out.RetailCurrentMonth = floatPtr(r.RetailCurrentMonth)
	out.RetailLastYearMonth = floatPtr(r.RetailLastYearMonth)
	return out
}

func floatPtr(v float64) *float64 {
	val := v
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
		"sales_current_month":  true,
		"retail_current_month": true,
		"is_eat_wear_use":      true,
		"is_small_micro":       true,
	}
	aliases := map[string]string{
		"salesCurrentMonth":  "sales_current_month",
		"retailCurrentMonth": "retail_current_month",
		"isEatWearUse":       "is_eat_wear_use",
		"isSmallMicro":       "is_small_micro",
	}
	return pickUpdates(patch, allowed, aliases)
}

func pickACUpdates(patch map[string]interface{}) map[string]interface{} {
	allowed := map[string]bool{
		"revenue_current_month": true,
		"room_current_month":    true,
		"food_current_month":    true,
		"goods_current_month":   true,
		"retail_current_month":  true,
		"is_eat_wear_use":       true,
		"is_small_micro":        true,
	}
	aliases := map[string]string{
		"revenueCurrentMonth": "revenue_current_month",
		"roomCurrentMonth":    "room_current_month",
		"foodCurrentMonth":    "food_current_month",
		"goodsCurrentMonth":   "goods_current_month",
		"retailCurrentMonth":  "retail_current_month",
		"isEatWearUse":        "is_eat_wear_use",
		"isSmallMicro":        "is_small_micro",
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
				WHEN sales_last_year_month = 0 THEN NULL
				ELSE (sales_current_month - sales_last_year_month) / sales_last_year_month * 100
			END,
			sales_cumulative_rate = CASE
				WHEN sales_last_year_cumulative = 0 THEN NULL
				ELSE (sales_current_cumulative - sales_last_year_cumulative) / sales_last_year_cumulative * 100
			END,
			retail_month_rate = CASE
				WHEN retail_last_year_month = 0 THEN NULL
				ELSE (retail_current_month - retail_last_year_month) / retail_last_year_month * 100
			END,
			retail_cumulative_rate = CASE
				WHEN retail_last_year_cumulative = 0 THEN NULL
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
				WHEN revenue_last_year_month = 0 THEN NULL
				ELSE (revenue_current_month - revenue_last_year_month) / revenue_last_year_month * 100
			END,
			revenue_cumulative_rate = CASE
				WHEN revenue_last_year_cumulative = 0 THEN NULL
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
