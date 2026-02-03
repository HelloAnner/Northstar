/**
 * 联动预览 API 测试
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
	"northstar/internal/linkage"
	"northstar/internal/store"
)

func TestLinkagePreview_UIAnchor(t *testing.T) {
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

	index, err := linkage.LoadTemplateIndex()
	if err != nil {
		t.Fatalf("load template: %v", err)
	}
	code := index.FirstCode("批发")
	if code == "" {
		t.Fatalf("missing code")
	}

	if err := st.Exec(`
		INSERT INTO wholesale_retail (
			credit_code, name, industry_code, industry_type, company_scale, row_no,
			data_year, data_month,
			sales_current_month, sales_last_year_month,
			retail_current_month, retail_last_year_month,
			sales_current_cumulative, sales_last_year_cumulative,
			retail_current_cumulative, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AAA", "企业A", code, "wholesale", 1, 1, 2025, 12, 120, 100, 80, 70, 1000, 900, 600, 550, "批发", "test.xlsx"); err != nil {
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
		"anchor": map[string]any{
			"ui": map[string]any{
				"rowId":     "wr:" + strconv.FormatInt(id, 10),
				"columnKey": "retailCurrentMonth",
			},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/linkage/preview", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Nodes []linkage.ImpactNode `json:"nodes"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Nodes) == 0 {
		t.Fatalf("empty nodes")
	}
}
