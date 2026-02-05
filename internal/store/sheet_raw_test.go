/**
 * raw 表写入测试
 *
 * @author Anner
 * Created on 2026/2/5
 */

package store

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestInsertSheetRaw_SavesColumnsRowsCells(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	insertByName(t, st, "InsertSheetColumns", buildColumnsValue(t, st))
	insertByName(t, st, "InsertSheetRows", buildRowsValue(t, st))
	insertByName(t, st, "BatchInsertSheetCells", buildCellsValue(t, st))
	insertByName(t, st, "InsertSheetMerges", buildMergesValue(t, st))

	assertCount(t, st, "sheet_columns", 1)
	assertCount(t, st, "sheet_rows", 1)
	assertCount(t, st, "sheet_cells", 1)
	assertCount(t, st, "sheet_merges", 1)
}

func TestCountSheetCellsByImportLog(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "northstar.db")
	st, err := New(dbPath)
	if err != nil {
		t.Fatalf("create store failed: %v", err)
	}
	defer func() { _ = st.Close() }()

	_, err = st.db.Exec(`
		INSERT INTO sheet_cells (sheet_name, row_idx, col_idx, import_log_id)
		VALUES (?, ?, ?, ?), (?, ?, ?, ?)
	`, "测试表", 1, 1, 1, "测试表", 2, 1, 2)
	if err != nil {
		t.Fatalf("insert sheet_cells failed: %v", err)
	}

	count, err := st.CountSheetCellsByImportLog(1)
	if err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("count mismatch: got=%d want=1", count)
	}
}

func insertByName(t *testing.T, st *Store, name string, arg reflect.Value) {
	m := reflect.ValueOf(st).MethodByName(name)
	if !m.IsValid() {
		t.Fatalf("missing method: %s", name)
	}
	if m.Type().NumIn() != 1 {
		t.Fatalf("unexpected method signature: %s", name)
	}
	res := m.Call([]reflect.Value{arg})
	if len(res) != 1 {
		t.Fatalf("unexpected return for %s", name)
	}
	if !res[0].IsNil() {
		err := res[0].Interface().(error)
		t.Fatalf("%s failed: %v", name, err)
	}
}

func buildColumnsValue(t *testing.T, st *Store) reflect.Value {
	return buildSliceArg(t, st, "InsertSheetColumns", func(v reflect.Value) {
		setFieldString(v, "SheetName", "测试表")
		setFieldInt(v, "ColIdx", 1)
		setFieldString(v, "HeaderText", "字段A")
		setFieldString(v, "NormalizedHeader", "字段A")
		setFieldFloat(v, "ColWidth", 12.5)
	})
}

func buildRowsValue(t *testing.T, st *Store) reflect.Value {
	return buildSliceArg(t, st, "InsertSheetRows", func(v reflect.Value) {
		setFieldString(v, "SheetName", "测试表")
		setFieldInt(v, "RowIdx", 1)
		setFieldFloat(v, "RowHeight", 18)
	})
}

func buildCellsValue(t *testing.T, st *Store) reflect.Value {
	return buildSliceArg(t, st, "BatchInsertSheetCells", func(v reflect.Value) {
		setFieldString(v, "SheetName", "测试表")
		setFieldInt(v, "RowIdx", 1)
		setFieldInt(v, "ColIdx", 1)
		setFieldString(v, "A1", "A1")
		setFieldString(v, "CellType", "string")
		setFieldString(v, "RawValue", "示例")
		setFieldString(v, "CalcValue", "示例")
		setFieldString(v, "NumFormat", "@")
		setFieldInt(v, "StyleID", 0)
	})
}

func buildMergesValue(t *testing.T, st *Store) reflect.Value {
	return buildSliceArg(t, st, "InsertSheetMerges", func(v reflect.Value) {
		setFieldString(v, "SheetName", "测试表")
		setFieldString(v, "MergeRange", "A1:B2")
		setFieldInt(v, "StartRow", 1)
		setFieldInt(v, "StartCol", 1)
		setFieldInt(v, "EndRow", 2)
		setFieldInt(v, "EndCol", 2)
	})
}

func buildSliceArg(t *testing.T, st *Store, method string, fill func(v reflect.Value)) reflect.Value {
	m := reflect.ValueOf(st).MethodByName(method)
	if !m.IsValid() {
		t.Fatalf("missing method: %s", method)
	}
	if m.Type().NumIn() != 1 {
		t.Fatalf("unexpected method signature: %s", method)
	}
	param := m.Type().In(0)
	if param.Kind() != reflect.Slice {
		t.Fatalf("unexpected param type: %s", param.Kind())
	}
	itemType := param.Elem()
	item := reflect.New(itemType).Elem()
	fill(item)
	slice := reflect.MakeSlice(param, 0, 1)
	slice = reflect.Append(slice, item)
	return slice
}

func setFieldString(v reflect.Value, name, val string) {
	f := v.FieldByName(name)
	if f.IsValid() && f.CanSet() && f.Kind() == reflect.String {
		f.SetString(val)
	}
}

func setFieldInt(v reflect.Value, name string, val int) {
	f := v.FieldByName(name)
	if f.IsValid() && f.CanSet() {
		switch f.Kind() {
		case reflect.Int, reflect.Int32, reflect.Int64:
			f.SetInt(int64(val))
		}
	}
}

func setFieldFloat(v reflect.Value, name string, val float64) {
	f := v.FieldByName(name)
	if f.IsValid() && f.CanSet() && (f.Kind() == reflect.Float32 || f.Kind() == reflect.Float64) {
		f.SetFloat(val)
	}
}

func assertCount(t *testing.T, st *Store, table string, want int) {
	row := st.QueryRow("SELECT COUNT(*) FROM " + table)
	var got int
	if err := row.Scan(&got); err != nil {
		t.Fatalf("count %s failed: %v", table, err)
	}
	if got != want {
		t.Fatalf("%s count mismatch: got=%d want=%d", table, got, want)
	}
}
