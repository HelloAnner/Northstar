/**
 * raw 与汇总表结构测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

package store

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestSchema_IncludesRawAndSummaryTables(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	expect := []string{
		"sheet_cells",
		"sheet_columns",
		"sheet_rows",
		"sheet_merges",
		"summary_limit_above_retail",
		"summary_micro_small",
		"summary_eat_wear_use",
	}

	for _, name := range expect {
		if !tableExists(st.db, name) {
			t.Fatalf("missing table: %s", name)
		}
	}
}

func tableExists(db *sql.DB, name string) bool {
	row := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name = ?`, name)
	var got string
	if err := row.Scan(&got); err != nil {
		return false
	}
	return got == name
}
