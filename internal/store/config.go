package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// GetConfig 获取配置项
func (s *Store) GetConfig(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("config key not found: %s", key)
		}
		return "", err
	}
	return value, nil
}

// GetConfigInt 获取整数配置项
func (s *Store) GetConfigInt(key string) (int, error) {
	value, err := s.GetConfig(key)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(value)
}

// GetConfigFloat 获取浮点数配置项
func (s *Store) GetConfigFloat(key string) (float64, error) {
	value, err := s.GetConfig(key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(value, 64)
}

// SetConfig 设置配置项
func (s *Store) SetConfig(key, value string) error {
	_, err := s.db.Exec(`
		INSERT INTO config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = CURRENT_TIMESTAMP
	`, key, value, value)
	return err
}

// SetConfigInt 设置整数配置项
func (s *Store) SetConfigInt(key string, value int) error {
	return s.SetConfig(key, strconv.Itoa(value))
}

// SetConfigFloat 设置浮点数配置项
func (s *Store) SetConfigFloat(key string, value float64) error {
	return s.SetConfig(key, strconv.FormatFloat(value, 'f', -1, 64))
}

// EnsureConfig 在配置不存在时写入默认值。
func (s *Store) EnsureConfig(key string, value string) error {
	var existing string
	err := s.db.QueryRow("SELECT value FROM config WHERE key = ?", key).Scan(&existing)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return err
	}
	return s.SetConfig(key, value)
}

// GetConfigBatch 批量获取配置项，单次查询返回多个 key 的值。
func (s *Store) GetConfigBatch(keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return map[string]string{}, nil
	}
	placeholders := make([]string, len(keys))
	args := make([]interface{}, len(keys))
	for i, k := range keys {
		placeholders[i] = "?"
		args[i] = k
	}
	query := "SELECT key, value FROM config WHERE key IN (" + strings.Join(placeholders, ",") + ")"
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]string, len(keys))
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		result[k] = v
	}
	return result, rows.Err()
}

// GetAllConfig 获取所有配置项
func (s *Store) GetAllConfig() (map[string]string, error) {
	rows, err := s.db.Query("SELECT key, value FROM config")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	config := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		config[key] = value
	}

	return config, rows.Err()
}

// GetCurrentYearMonth 获取当前操作的年月
func (s *Store) GetCurrentYearMonth() (year, month int, err error) {
	year, err = s.GetConfigInt("current_year")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get current_year: %w", err)
	}

	month, err = s.GetConfigInt("current_month")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get current_month: %w", err)
	}

	return year, month, nil
}

// SetCurrentYearMonth 设置当前操作的年月
func (s *Store) SetCurrentYearMonth(year, month int) error {
	if err := s.SetConfigInt("current_year", year); err != nil {
		return err
	}
	if err := s.SetConfigInt("current_month", month); err != nil {
		return err
	}
	return nil
}
