/**
 * 累计联动测试
 *
 * @author Anner
 * Created on 2026/2/3
 */

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

func TestUpdateCompany_SalesCurrentMonth_RecalcCumulative(t *testing.T) {
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
			sales_prev_cumulative, sales_current_month, sales_current_cumulative,
			retail_prev_cumulative, retail_current_month, retail_current_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "WR", "企业A", "5101", "wholesale", 1, 1, 2025, 12, 1000, 100, 1100, 800, 80, 880, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	var id int64
	if err := st.QueryRow("SELECT id FROM wholesale_retail WHERE credit_code = ?", "WR").Scan(&id); err != nil {
		t.Fatalf("query id: %v", err)
	}

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	body, _ := json.Marshal(map[string]any{
		"salesCurrentMonth": 200.0,
	})
	req := httptest.NewRequest(http.MethodPatch, "/api/companies/wr:"+strconv.FormatInt(id, 10), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}

	var cumulative float64
	if err := st.QueryRow("SELECT sales_current_cumulative FROM wholesale_retail WHERE id = ?", id).Scan(&cumulative); err != nil {
		t.Fatalf("query cumulative: %v", err)
	}
	if cumulative != 1200 {
		t.Fatalf("unexpected cumulative: %v", cumulative)
	}
}

func TestUpdateCompany_RevenueCurrentMonth_RecalcCumulative(t *testing.T) {
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
			revenue_prev_cumulative, revenue_current_month, revenue_current_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AC", "企业B", "6101", "accommodation", 1, 1, 2025, 12, 500, 50, 550, "住宿", "test.xlsx"); err != nil {
		t.Fatalf("insert ac: %v", err)
	}

	var id int64
	if err := st.QueryRow("SELECT id FROM accommodation_catering WHERE credit_code = ?", "AC").Scan(&id); err != nil {
		t.Fatalf("query id: %v", err)
	}

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	body, _ := json.Marshal(map[string]any{
		"revenueCurrentMonth": 90.0,
	})
	req := httptest.NewRequest(http.MethodPatch, "/api/companies/ac:"+strconv.FormatInt(id, 10), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}

	var cumulative float64
	if err := st.QueryRow("SELECT revenue_current_cumulative FROM accommodation_catering WHERE id = ?", id).Scan(&cumulative); err != nil {
		t.Fatalf("query cumulative: %v", err)
	}
	if cumulative != 590 {
		t.Fatalf("unexpected cumulative: %v", cumulative)
	}
}
