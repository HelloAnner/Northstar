package v3

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"northstar/internal/store"
)

func TestBatchUpdateCompanies_UpdatesWRAndAC(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := st.SetCurrentYearMonth(2025, 12); err != nil {
		t.Fatalf("set ym: %v", err)
	}

	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			sales_current_month, sales_last_year_month,
			sales_current_cumulative, sales_last_year_cumulative,
			retail_current_month, retail_last_year_month,
			retail_current_cumulative, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "WR001", "批零企业", "5101", "wholesale", 1, 1, 2025, 12, 100, 80, 100, 80, 90, 70, 90, 70, "批零", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	if err := st.Exec(`
		INSERT INTO accommodation_catering (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			revenue_current_month, revenue_last_year_month,
			revenue_current_cumulative, revenue_last_year_cumulative,
			room_current_month, room_current_cumulative,
			food_current_month, food_current_cumulative,
			goods_current_month, goods_current_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AC001", "住餐企业", "6101", "accommodation", 2, 2, 2025, 12, 200, 150, 200, 150, 60, 160, 70, 170, 80, 180, "住餐", "test.xlsx"); err != nil {
		t.Fatalf("insert ac: %v", err)
	}

	var wrID int64
	if err := st.QueryRow("SELECT id FROM wholesale_retail WHERE credit_code = ?", "WR001").Scan(&wrID); err != nil {
		t.Fatalf("query wr id: %v", err)
	}
	var acID int64
	if err := st.QueryRow("SELECT id FROM accommodation_catering WHERE credit_code = ?", "AC001").Scan(&acID); err != nil {
		t.Fatalf("query ac id: %v", err)
	}

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	body, _ := json.Marshal(map[string]any{
		"updates": []map[string]any{
			{
				"id": "wr:" + strconv.FormatInt(wrID, 10),
				"patch": map[string]any{
					"salesCurrentMonth":  120,
					"retailCurrentMonth": 95,
				},
			},
			{
				"id": "ac:" + strconv.FormatInt(acID, 10),
				"patch": map[string]any{
					"roomCurrentMonth":       40,
					"roomCurrentCumulative":  140,
					"foodCurrentMonth":       50,
					"foodCurrentCumulative":  150,
					"goodsCurrentMonth":      60,
					"goodsCurrentCumulative": 160,
				},
			},
		},
	})

	req := httptest.NewRequest(http.MethodPatch, "/api/companies/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}

	var salesCurrent float64
	if err := st.QueryRow("SELECT sales_current_month FROM wholesale_retail WHERE id = ?", wrID).Scan(&salesCurrent); err != nil {
		t.Fatalf("query wr: %v", err)
	}
	if salesCurrent != 120 {
		t.Fatalf("unexpected sales_current_month: %v", salesCurrent)
	}

	var roomCur, roomCum, foodCur, foodCum, goodsCur, goodsCum, retailCur float64
	if err := st.QueryRow(`
		SELECT room_current_month, room_current_cumulative,
			food_current_month, food_current_cumulative,
			goods_current_month, goods_current_cumulative,
			retail_current_month
		FROM accommodation_catering WHERE id = ?
	`, acID).Scan(&roomCur, &roomCum, &foodCur, &foodCum, &goodsCur, &goodsCum, &retailCur); err != nil {
		t.Fatalf("query ac: %v", err)
	}

	if roomCur != 40 || roomCum != 140 {
		t.Fatalf("unexpected room values: %v %v", roomCur, roomCum)
	}
	if foodCur != 50 || foodCum != 150 {
		t.Fatalf("unexpected food values: %v %v", foodCur, foodCum)
	}
	if goodsCur != 60 || goodsCum != 160 {
		t.Fatalf("unexpected goods values: %v %v", goodsCur, goodsCum)
	}
	if retailCur != 110 {
		t.Fatalf("unexpected retail_current_month: %v", retailCur)
	}
}
