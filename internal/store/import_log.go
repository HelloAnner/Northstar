package store

import (
	"database/sql"
	"fmt"

	"northstar/internal/model"
)

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

// GetLatestImportLog 获取最新导入日志
func (s *Store) GetLatestImportLog() (*model.ImportLog, error) {
	row := s.db.QueryRow(`
		SELECT id, filename, file_path, file_size, file_hash,
		       total_sheets, imported_sheets, skipped_sheets,
		       total_rows, imported_rows, error_rows,
		       status, error_message, started_at, completed_at
		FROM import_logs
		ORDER BY id DESC
		LIMIT 1
	`)

	var log model.ImportLog
	var completedAt sql.NullTime
	var errorMessage sql.NullString
	if err := row.Scan(
		&log.ID,
		&log.Filename,
		&log.FilePath,
		&log.FileSize,
		&log.FileHash,
		&log.TotalSheets,
		&log.ImportedSheets,
		&log.SkippedSheets,
		&log.TotalRows,
		&log.ImportedRows,
		&log.ErrorRows,
		&log.Status,
		&errorMessage,
		&log.StartedAt,
		&completedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get latest import log: %w", err)
	}
	if completedAt.Valid {
		log.CompletedAt = &completedAt.Time
	}
	if errorMessage.Valid {
		log.ErrorMessage = errorMessage.String
	}
	return &log, nil
}
