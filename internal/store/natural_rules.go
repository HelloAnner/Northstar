package store

import (
	"database/sql"
	"fmt"
)

// NaturalRule 自然语言规则记录
type NaturalRule struct {
	ID      int64  `json:"id"`
	Text    string `json:"text"`
	Enabled bool   `json:"enabled"`
}

// ListNaturalRules 列出自然语言规则
func (s *Store) ListNaturalRules(enabledOnly bool) ([]NaturalRule, error) {
	query := `SELECT id, text, enabled FROM natural_rules`
	if enabledOnly {
		query += " WHERE enabled = 1"
	}
	query += " ORDER BY id ASC"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query natural rules: %w", err)
	}
	defer rows.Close()

	result := make([]NaturalRule, 0)
	for rows.Next() {
		r, err := scanNaturalRule(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func scanNaturalRule(rows *sql.Rows) (NaturalRule, error) {
	var r NaturalRule
	var enabled int
	if err := rows.Scan(&r.ID, &r.Text, &enabled); err != nil {
		return NaturalRule{}, fmt.Errorf("scan natural rule: %w", err)
	}
	r.Enabled = enabled == 1
	return r, nil
}

// CreateNaturalRule 新增自然语言规则
func (s *Store) CreateNaturalRule(text string) (int64, error) {
	result, err := s.db.Exec(
		`INSERT INTO natural_rules (text, enabled) VALUES (?, 1)`, text,
	)
	if err != nil {
		return 0, fmt.Errorf("insert natural rule: %w", err)
	}
	return result.LastInsertId()
}

// UpdateNaturalRule 更新自然语言规则
func (s *Store) UpdateNaturalRule(id int64, text string, enabled bool) error {
	_, err := s.db.Exec(
		`UPDATE natural_rules SET text = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		text, boolToInt(enabled), id,
	)
	if err != nil {
		return fmt.Errorf("update natural rule: %w", err)
	}
	return nil
}

// DeleteNaturalRule 删除自然语言规则
func (s *Store) DeleteNaturalRule(id int64) error {
	_, err := s.db.Exec("DELETE FROM natural_rules WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete natural rule: %w", err)
	}
	return nil
}
