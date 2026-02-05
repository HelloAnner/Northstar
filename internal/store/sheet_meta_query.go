/**
 * Sheet 元信息查询
 *
 * @author Anner
 * Created on 2026/2/5
 */

package store

import (
	"fmt"

	"northstar/internal/model"
)

// ListSheetMetaByImportLog 查询指定导入批次的 Sheet 元信息
func (s *Store) ListSheetMetaByImportLog(importLogID int64) ([]model.SheetMeta, error) {
	rows, err := s.db.Query(`
		SELECT id, sheet_name, sheet_type, confidence,
		       total_rows, total_columns, imported_rows,
		       columns_json, column_mapping_json,
		       status, error_message, import_log_id, source_file, created_at
		FROM sheets_meta
		WHERE import_log_id = ?
		ORDER BY id ASC
	`, importLogID)
	if err != nil {
		return nil, fmt.Errorf("failed to query sheets_meta: %w", err)
	}
	defer rows.Close()

	result := make([]model.SheetMeta, 0)
	for rows.Next() {
		var meta model.SheetMeta
		if err := rows.Scan(
			&meta.ID,
			&meta.SheetName,
			&meta.SheetType,
			&meta.Confidence,
			&meta.TotalRows,
			&meta.TotalColumns,
			&meta.ImportedRows,
			&meta.ColumnsJSON,
			&meta.ColumnMappingJSON,
			&meta.Status,
			&meta.ErrorMessage,
			&meta.ImportLogID,
			&meta.SourceFile,
			&meta.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sheets_meta: %w", err)
		}
		result = append(result, meta)
	}
	return result, nil
}
