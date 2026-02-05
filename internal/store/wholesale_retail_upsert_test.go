/**
 * 批零覆盖导入测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

package store

import (
	"path/filepath"
	"testing"

	"northstar/internal/model"
)

func TestBatchInsertWR_UpsertByKey(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	year, month := 2025, 12
	credit := "913000000000000001"
	industry := "wholesale"

	first := &model.WholesaleRetail{
		CreditCode:        credit,
		Name:              "企业A",
		IndustryType:      industry,
		DataYear:          year,
		DataMonth:         month,
		SalesCurrentMonth: 100,
	}
	if err := st.BatchInsertWR([]*model.WholesaleRetail{first}); err != nil {
		t.Fatalf("insert wr failed: %v", err)
	}

	second := &model.WholesaleRetail{
		CreditCode:        credit,
		Name:              "企业A-更新",
		IndustryType:      industry,
		DataYear:          year,
		DataMonth:         month,
		SalesCurrentMonth: 220,
	}
	if err := st.BatchInsertWR([]*model.WholesaleRetail{second}); err != nil {
		t.Fatalf("upsert wr failed: %v", err)
	}

	var count int
	if err := st.QueryRow(
		`SELECT COUNT(*) FROM wholesale_retail WHERE data_year = ? AND data_month = ? AND credit_code = ? AND industry_type = ?`,
		year, month, credit, industry,
	).Scan(&count); err != nil {
		t.Fatalf("count wr failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("unexpected count: %d", count)
	}

	var name string
	var sales float64
	if err := st.QueryRow(
		`SELECT name, sales_current_month FROM wholesale_retail WHERE data_year = ? AND data_month = ? AND credit_code = ? AND industry_type = ?`,
		year, month, credit, industry,
	).Scan(&name, &sales); err != nil {
		t.Fatalf("query wr failed: %v", err)
	}
	if name != "企业A-更新" {
		t.Fatalf("name not updated: %s", name)
	}
	if sales != 220 {
		t.Fatalf("sales not updated: %v", sales)
	}
}
