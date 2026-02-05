/**
 * 导入预览接口测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

package v3

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"northstar/internal/model"
	"northstar/internal/store"
)

type previewResp struct {
	Sheets []struct {
		SheetName string `json:"sheetName"`
	} `json:"sheets"`
}

type sheetResp struct {
	SheetName string `json:"sheetName"`
	Cells     []struct {
		RowIdx int `json:"rowIdx"`
		ColIdx int `json:"colIdx"`
	} `json:"cells"`
}

func TestImportPreview_ReturnsRawAndMapping(t *testing.T) {
	gin.SetMode(gin.TestMode)
	st := createPreviewStore(t)
	defer func() { _ = st.Close() }()

	r := gin.New()
	h := NewHandler(st, "")
	api := r.Group("/api")
	h.RegisterRoutes(api)

	req := httptest.NewRequest(http.MethodGet, "/api/import/preview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("preview status: %d body=%s", w.Code, w.Body.String())
	}
	var resp previewResp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("preview decode failed: %v", err)
	}
	if len(resp.Sheets) == 0 {
		t.Fatalf("preview sheets empty")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/import/sheet?name=测试表", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("sheet status: %d", w2.Code)
	}
	var sheet sheetResp
	if err := json.Unmarshal(w2.Body.Bytes(), &sheet); err != nil {
		t.Fatalf("sheet decode failed: %v", err)
	}
	if sheet.SheetName != "测试表" {
		t.Fatalf("sheet name mismatch: %s", sheet.SheetName)
	}
	if len(sheet.Cells) == 0 {
		t.Fatalf("sheet cells empty")
	}
}

func createPreviewStore(t *testing.T) *store.Store {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	id, err := st.CreateImportLog("file.xlsx", "file.xlsx", 10, "hash")
	if err != nil {
		t.Fatalf("create import log failed: %v", err)
	}

	meta := model.SheetMeta{
		SheetName:    "测试表",
		SheetType:    "wholesale",
		Confidence:   0.9,
		TotalRows:    2,
		TotalColumns: 2,
		ImportedRows: 1,
		ColumnsJSON:  "[\"A\",\"B\"]",
		Status:       "imported",
		ImportLogID:  &id,
		SourceFile:   "file.xlsx",
	}
	if err := st.InsertSheetMeta(meta); err != nil {
		t.Fatalf("insert sheet meta failed: %v", err)
	}

	columns := []model.SheetColumn{{
		SheetName:   "测试表",
		ColIdx:      1,
		HeaderText:  "A",
		ColWidth:    10,
		ImportLogID: &id,
		SourceFile:  "file.xlsx",
	}}
	if err := st.InsertSheetColumns(columns); err != nil {
		t.Fatalf("insert sheet columns failed: %v", err)
	}
	rows := []model.SheetRow{{
		SheetName:   "测试表",
		RowIdx:      1,
		RowHeight:   15,
		ImportLogID: &id,
		SourceFile:  "file.xlsx",
	}}
	if err := st.InsertSheetRows(rows); err != nil {
		t.Fatalf("insert sheet rows failed: %v", err)
	}
	cells := []model.SheetCell{{
		SheetName:   "测试表",
		RowIdx:      1,
		ColIdx:      1,
		A1:          "A1",
		CellType:    "string",
		RawValue:    "测试",
		CalcValue:   "测试",
		NumFormat:   "@",
		StyleID:     0,
		ImportLogID: &id,
		SourceFile:  "file.xlsx",
	}}
	if err := st.BatchInsertSheetCells(cells); err != nil {
		t.Fatalf("insert sheet cells failed: %v", err)
	}

	return st
}
