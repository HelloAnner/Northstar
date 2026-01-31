package v3

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"northstar/internal/importer"
	"northstar/internal/store"
)

func TestListCompanies_PRD_HuaNanCompany_IncludesSalesLastYearMonth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	coord := importer.NewCoordinator(st)
	ch := coord.Import(importer.ImportOptions{
		FilePath:         filepath.Join("..", "..", "..", "prd", "12月月报（预估）_补全企业名称社会代码_20260129.xlsx"),
		OriginalFilename: "12月月报（预估）_补全企业名称社会代码_20260129.xlsx",
		ClearExisting:    true,
		UpdateConfigYM:   true,
		CalculateFields:  true,
	})
	for evt := range ch {
		if evt.Type == "error" {
			t.Fatalf("import error: %s", evt.Message)
		}
	}

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	name := "华南烟酒测试企业0355有限责任公司"
	req := httptest.NewRequest(http.MethodGet, "/api/companies?industryType=all&keyword="+name+"&page=1&pageSize=10", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}

	var resp listCompaniesResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v body=%s", err, w.Body.String())
	}
	if len(resp.Items) != 1 {
		t.Fatalf("unexpected items: %d", len(resp.Items))
	}
	it := resp.Items[0]
	if it.Kind != "wr" {
		t.Fatalf("unexpected kind: %s", it.Kind)
	}
	if it.SalesLastYearMonth == nil {
		t.Fatalf("salesLastYearMonth missing")
	}
	if diff := *it.SalesLastYearMonth - 1841; diff < -1e-9 || diff > 1e-9 {
		t.Fatalf("unexpected salesLastYearMonth: %v", *it.SalesLastYearMonth)
	}
}
