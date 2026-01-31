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

func TestUpdateCompany_UpdateSalesLastYearMonth_RecalcRate(t *testing.T) {
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
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AAA", "企业A", "5101", "wholesale", 1, 2, 2025, 12, 120, 100, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	var id int64
	if err := st.QueryRow("SELECT id FROM wholesale_retail WHERE credit_code = ?", "AAA").Scan(&id); err != nil {
		t.Fatalf("query id: %v", err)
	}

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	body, _ := json.Marshal(map[string]any{
		"salesLastYearMonth": 80.0,
	})
	req := httptest.NewRequest(http.MethodPatch, "/api/companies/wr:"+strconv.FormatInt(id, 10), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}

	var rate float64
	if err := st.QueryRow("SELECT sales_month_rate FROM wholesale_retail WHERE id = ?", id).Scan(&rate); err != nil {
		t.Fatalf("query rate: %v", err)
	}
	if diff := rate - 50; diff < -1e-6 || diff > 1e-6 {
		t.Fatalf("unexpected rate: %v", rate)
	}
}

func TestUpdateCompany_UpdateFoodCurrentMonth_RecalcACRetail(t *testing.T) {
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
		INSERT INTO accommodation_catering (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			food_current_month, goods_current_month,
			retail_current_month,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AAA", "企业A", "5101", "accommodation", 1, 2, 2025, 12, 10, 5, 15, "住宿", "test.xlsx"); err != nil {
		t.Fatalf("insert ac: %v", err)
	}

	var id int64
	if err := st.QueryRow("SELECT id FROM accommodation_catering WHERE credit_code = ?", "AAA").Scan(&id); err != nil {
		t.Fatalf("query id: %v", err)
	}

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	body, _ := json.Marshal(map[string]any{
		"foodCurrentMonth": 20.0,
	})
	req := httptest.NewRequest(http.MethodPatch, "/api/companies/ac:"+strconv.FormatInt(id, 10), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}

	var retail float64
	if err := st.QueryRow("SELECT retail_current_month FROM accommodation_catering WHERE id = ?", id).Scan(&retail); err != nil {
		t.Fatalf("query retail: %v", err)
	}
	if diff := retail - 25; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("unexpected retail: %v", retail)
	}
}
