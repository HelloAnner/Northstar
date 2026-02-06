/**
 * 智能调整失败原因提示测试
 *
 * @author Anner
 * Created on 2026/2/4
 */

package v3

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"northstar/internal/store"
)

func TestOptimize_Notices_LastYearZeroRate(t *testing.T) {
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
			retail_current_month, retail_last_year_month,
			retail_current_cumulative, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "AAA", "企业A", "5101", "wholesale", 1, 1, 2025, 12, 120, 0, 120, 0, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	body, _ := json.Marshal(map[string]any{
		"targets": map[string]float64{
			"limitAbove_month_rate": 10,
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/optimize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Notices []OptimizeNotice `json:"notices"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Notices) == 0 {
		t.Fatalf("missing notices")
	}
	found := false
	for _, notice := range resp.Notices {
		if notice.IndicatorID == "limitAbove_month_rate" && notice.Code == "last_year_zero" {
			if !strings.Contains(notice.Message, "根据") || !strings.Contains(notice.Message, "规则无法调整") || !strings.Contains(notice.Message, "建议") {
				t.Fatalf("notice message format invalid: %s", notice.Message)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected last_year_zero notice, got: %+v", resp.Notices)
	}
}

func TestOptimize_Notices_TargetSame_Format(t *testing.T) {
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
			retail_current_month, retail_last_year_month,
			retail_current_cumulative, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "CCC", "企业C", "5101", "wholesale", 1, 1, 2025, 12, 100, 80, 100, 80, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	body, _ := json.Marshal(map[string]any{
		"targets": map[string]float64{
			"limitAbove_month_value": 100,
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/optimize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Notices []OptimizeNotice `json:"notices"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Notices) == 0 {
		t.Fatalf("missing notices")
	}
	found := false
	for _, notice := range resp.Notices {
		if notice.IndicatorID == "limitAbove_month_value" && notice.Code == "target_same" {
			if !strings.Contains(notice.Message, "根据") || !strings.Contains(notice.Message, "规则无法调整") || !strings.Contains(notice.Message, "建议") {
				t.Fatalf("notice message format invalid: %s", notice.Message)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected target_same notice, got: %+v", resp.Notices)
	}
}

func TestOptimize_Notices_BelowMinCumulative(t *testing.T) {
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
			retail_current_month, retail_last_year_month,
			retail_current_cumulative, retail_last_year_cumulative,
			source_sheet, source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "BBB", "企业B", "5101", "wholesale", 1, 1, 2025, 12, 50, 40, 200, 180, "批发", "test.xlsx"); err != nil {
		t.Fatalf("insert wr: %v", err)
	}

	h := NewHandler(st, "")
	r := gin.New()
	api := r.Group("/api")
	h.RegisterRoutes(api)

	body, _ := json.Marshal(map[string]any{
		"targets": map[string]float64{
			"limitAbove_cumulative_value": 100,
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/api/optimize", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", w.Code, w.Body.String())
	}

	var resp struct {
		Notices []OptimizeNotice `json:"notices"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Notices) == 0 {
		t.Fatalf("missing notices")
	}
	found := false
	for _, notice := range resp.Notices {
		if notice.IndicatorID != "limitAbove_cumulative_value" {
			continue
		}
		if notice.Code != "below_min" {
			continue
		}
		if notice.SuggestMin == nil {
			t.Fatalf("missing suggestMin in notice: %+v", notice)
		}
		if math.Round(*notice.SuggestMin) != 150 {
			t.Fatalf("unexpected suggestMin: %.2f", *notice.SuggestMin)
		}
		if !strings.Contains(notice.Message, "根据") || !strings.Contains(notice.Message, "规则无法调整") || !strings.Contains(notice.Message, "建议") {
			t.Fatalf("notice message format invalid: %s", notice.Message)
		}
		found = true
		break
	}
	if !found {
		t.Fatalf("expected below_min notice, got: %+v", resp.Notices)
	}
}
