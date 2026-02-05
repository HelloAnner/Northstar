/**
 * 导出三方对比汇总测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

package v3

import (
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
	"northstar/internal/model"
	"northstar/internal/store"
)

func TestBuildExportCompareSummary(t *testing.T) {
	st := createCompareStore(t)
	defer func() { _ = st.Close() }()

	f := excelize.NewFile()
	for _, name := range expectedExportSheets() {
		if name == f.GetSheetName(0) {
			f.SetSheetName(name, name)
			continue
		}
		f.NewSheet(name)
	}

	summary, err := buildExportCompareSummary(st, f, 2025, 12)
	if err != nil {
		t.Fatalf("build compare summary failed: %v", err)
	}
	if len(summary.Items) != 3 {
		t.Fatalf("unexpected compare items: %d", len(summary.Items))
	}
	for _, item := range summary.Items {
		if item.Status != "pass" {
			t.Fatalf("status not pass: %s", item.Status)
		}
	}
}

func createCompareStore(t *testing.T) *store.Store {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}

	logID, err := st.CreateImportLog("file.xlsx", "file.xlsx", 10, "hash")
	if err != nil {
		t.Fatalf("create import log failed: %v", err)
	}

	err = st.Exec(`
		INSERT INTO sheet_cells (sheet_name, row_idx, col_idx, import_log_id)
		VALUES (?, ?, ?, ?)
	`, "测试表", 1, 1, logID)
	if err != nil {
		t.Fatalf("insert sheet_cells failed: %v", err)
	}

	wr := &model.WholesaleRetail{
		Name:      "企业A",
		DataYear:  2025,
		DataMonth: 12,
	}
	if err := st.BatchInsertWR([]*model.WholesaleRetail{wr}); err != nil {
		t.Fatalf("insert wr failed: %v", err)
	}

	ac := &model.AccommodationCatering{
		Name:      "企业B",
		DataYear:  2025,
		DataMonth: 12,
	}
	if err := st.BatchInsertAC([]*model.AccommodationCatering{ac}); err != nil {
		t.Fatalf("insert ac failed: %v", err)
	}

	return st
}
