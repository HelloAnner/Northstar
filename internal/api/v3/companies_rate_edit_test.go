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

func TestUpdateCompany_EditRate_RecomputeFromExistingData(t *testing.T) {
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

	// 准备一个批零企业：有上年同期，本月为空；编辑增速后，应回算本月金额并按公式重算增速
	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			sales_current_month, sales_last_year_month,
			first_report_ip, fill_ip,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AAA", "企业A", "5101", "wholesale", 1, 2, 2025, 12, 0, 100, "", "", "批发", "test.xlsx"); err != nil {
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
		"salesMonthRate": 30.0,
	})
	req := httptest.NewRequest(http.MethodPatch, "/api/companies/wr:"+strconv.FormatInt(id, 10), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}

	var salesMonth float64
	var salesRate float64
	if err := st.QueryRow(
		"SELECT sales_current_month, sales_month_rate FROM wholesale_retail WHERE id = ?",
		id,
	).Scan(&salesMonth, &salesRate); err != nil {
		t.Fatalf("query row: %v", err)
	}

	if diff := salesMonth - 130; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("unexpected sales_current_month: %v", salesMonth)
	}
	if diff := salesRate - 30; diff < -1e-6 || diff > 1e-6 {
		t.Fatalf("unexpected sales_month_rate: %v", salesRate)
	}
}
