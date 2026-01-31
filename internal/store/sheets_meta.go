package store

import (
	"encoding/json"
	"fmt"

	"northstar/internal/model"
)

// InsertSheetMeta 写入 Sheet 元信息（用于追溯与容错）
func (s *Store) InsertSheetMeta(meta model.SheetMeta) error {
	_, err := s.db.Exec(`
		INSERT INTO sheets_meta (
			sheet_name, sheet_type, confidence,
			total_rows, total_columns,
			imported_rows,
			columns_json, column_mapping_json,
			status, error_message,
			import_log_id,
			source_file
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		meta.SheetName, meta.SheetType, meta.Confidence,
		meta.TotalRows, meta.TotalColumns,
		meta.ImportedRows,
		meta.ColumnsJSON, meta.ColumnMappingJSON,
		meta.Status, meta.ErrorMessage,
		meta.ImportLogID,
		meta.SourceFile,
	)
	if err != nil {
		return fmt.Errorf("failed to insert sheets_meta: %w", err)
	}
	return nil
}

// BuildColumnsJSON 将列名序列化为 JSON（避免上层重复处理）
func BuildColumnsJSON(columns []string) string {
	b, err := json.Marshal(columns)
	if err != nil {
		return "[]"
	}
	return string(b)
}

