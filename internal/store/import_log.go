package store

import "fmt"

// CreateImportLog 创建导入日志，返回 import_log_id
func (s *Store) CreateImportLog(filename, filePath string, fileSize int64, fileHash string) (int64, error) {
	res, err := s.db.Exec(`
		INSERT INTO import_logs (filename, file_path, file_size, file_hash, status)
		VALUES (?, ?, ?, ?, 'processing')
	`, filename, filePath, fileSize, fileHash)
	if err != nil {
		return 0, fmt.Errorf("failed to create import log: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get import log id: %w", err)
	}
	return id, nil
}

// UpdateImportLog 完成导入日志更新
func (s *Store) UpdateImportLog(id int64, totalSheets, importedSheets, skippedSheets, totalRows, importedRows, errorRows int, status, errorMessage string) error {
	_, err := s.db.Exec(`
		UPDATE import_logs SET
			total_sheets = ?,
			imported_sheets = ?,
			skipped_sheets = ?,
			total_rows = ?,
			imported_rows = ?,
			error_rows = ?,
			status = ?,
			error_message = ?,
			completed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, totalSheets, importedSheets, skippedSheets, totalRows, importedRows, errorRows, status, errorMessage, id)
	if err != nil {
		return fmt.Errorf("failed to update import log: %w", err)
	}
	return nil
}

