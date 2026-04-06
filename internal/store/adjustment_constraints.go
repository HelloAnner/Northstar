package store

import (
	"database/sql"
	"fmt"
)

// AdjustmentConstraint 硬约束记录
type AdjustmentConstraint struct {
	ID          int64    `json:"id"`
	Type        string   `json:"type"`
	IndicatorID string   `json:"indicatorId,omitempty"`
	MinValue    *float64 `json:"minValue,omitempty"`
	MaxValue    *float64 `json:"maxValue,omitempty"`
	FilterMode  string   `json:"filterMode,omitempty"`
	TriggerID   string   `json:"triggerId,omitempty"`
	EnsureID    string   `json:"ensureId,omitempty"`
	Relation    string   `json:"relation,omitempty"`
	Tolerance   float64  `json:"tolerance"`
	Enabled     bool     `json:"enabled"`
}

// ListAdjustmentConstraints 列出约束，enabledOnly=true 时只返回启用的
func (s *Store) ListAdjustmentConstraints(enabledOnly bool) ([]AdjustmentConstraint, error) {
	query := `SELECT id, type, indicator_id, min_value, max_value, filter_mode,
		trigger_id, ensure_id, relation, tolerance, enabled
		FROM adjustment_constraints`
	if enabledOnly {
		query += " WHERE enabled = 1"
	}
	query += " ORDER BY id ASC"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query adjustment constraints: %w", err)
	}
	defer rows.Close()

	result := make([]AdjustmentConstraint, 0)
	for rows.Next() {
		c, err := scanAdjustmentConstraint(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func scanAdjustmentConstraint(rows *sql.Rows) (AdjustmentConstraint, error) {
	var c AdjustmentConstraint
	var indicatorID, filterMode, triggerID, ensureID, relation sql.NullString
	var minVal, maxVal sql.NullFloat64
	var enabled int

	err := rows.Scan(
		&c.ID, &c.Type, &indicatorID, &minVal, &maxVal, &filterMode,
		&triggerID, &ensureID, &relation, &c.Tolerance, &enabled,
	)
	if err != nil {
		return AdjustmentConstraint{}, fmt.Errorf("scan adjustment constraint: %w", err)
	}

	c.IndicatorID = indicatorID.String
	c.FilterMode = filterMode.String
	c.TriggerID = triggerID.String
	c.EnsureID = ensureID.String
	c.Relation = relation.String
	if minVal.Valid {
		c.MinValue = &minVal.Float64
	}
	if maxVal.Valid {
		c.MaxValue = &maxVal.Float64
	}
	c.Enabled = enabled == 1
	return c, nil
}

// CreateAdjustmentConstraint 新增约束
func (s *Store) CreateAdjustmentConstraint(c AdjustmentConstraint) (int64, error) {
	result, err := s.db.Exec(`
		INSERT INTO adjustment_constraints
			(type, indicator_id, min_value, max_value, filter_mode,
			 trigger_id, ensure_id, relation, tolerance, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Type,
		nullString(c.IndicatorID),
		nullFloat(c.MinValue),
		nullFloat(c.MaxValue),
		nullString(c.FilterMode),
		nullString(c.TriggerID),
		nullString(c.EnsureID),
		nullString(c.Relation),
		c.Tolerance,
		boolToInt(c.Enabled),
	)
	if err != nil {
		return 0, fmt.Errorf("insert adjustment constraint: %w", err)
	}
	return result.LastInsertId()
}

// UpdateAdjustmentConstraint 更新约束
func (s *Store) UpdateAdjustmentConstraint(c AdjustmentConstraint) error {
	_, err := s.db.Exec(`
		UPDATE adjustment_constraints SET
			type = ?, indicator_id = ?, min_value = ?, max_value = ?,
			filter_mode = ?, trigger_id = ?, ensure_id = ?, relation = ?,
			tolerance = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		c.Type,
		nullString(c.IndicatorID),
		nullFloat(c.MinValue),
		nullFloat(c.MaxValue),
		nullString(c.FilterMode),
		nullString(c.TriggerID),
		nullString(c.EnsureID),
		nullString(c.Relation),
		c.Tolerance,
		boolToInt(c.Enabled),
		c.ID,
	)
	if err != nil {
		return fmt.Errorf("update adjustment constraint: %w", err)
	}
	return nil
}

// DeleteAdjustmentConstraint 删除约束
func (s *Store) DeleteAdjustmentConstraint(id int64) error {
	_, err := s.db.Exec("DELETE FROM adjustment_constraints WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete adjustment constraint: %w", err)
	}
	return nil
}

func nullString(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nullFloat(f *float64) any {
	if f == nil {
		return nil
	}
	return *f
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
