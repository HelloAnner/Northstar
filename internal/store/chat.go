/**
 * 对话历史存储
 *
 * @author Anner
 * Created on 2026/3/28
 */
package store

import (
	"database/sql"
	"encoding/json"
	"time"
)

// ChatSession 对话会话
type ChatSession struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Mode      string    `json:"mode"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ChatMessage 对话消息
type ChatMessage struct {
	ID           string          `json:"id"`
	SessionID    string          `json:"sessionId"`
	Role         string          `json:"role"`
	Content      string          `json:"content"`
	AppliedRules json.RawMessage `json:"appliedRules,omitempty"`
	RuleAdded    json.RawMessage `json:"ruleAdded,omitempty"`
	Seq          int             `json:"seq"`
	CreatedAt    time.Time       `json:"createdAt"`
}

// ListChatSessions 获取会话列表，按更新时间倒序
func (s *Store) ListChatSessions(limit int) ([]ChatSession, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT id, title, mode, created_at, updated_at
		 FROM chat_sessions ORDER BY updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []ChatSession
	for rows.Next() {
		var s ChatSession
		if err := rows.Scan(&s.ID, &s.Title, &s.Mode, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// CreateChatSession 创建会话
func (s *Store) CreateChatSession(id, title, mode string) error {
	_, err := s.db.Exec(
		`INSERT INTO chat_sessions (id, title, mode) VALUES (?, ?, ?)`,
		id, title, mode)
	return err
}

// UpdateChatSessionTitle 更新会话标题
func (s *Store) UpdateChatSessionTitle(id, title string) error {
	_, err := s.db.Exec(
		`UPDATE chat_sessions SET title = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		title, id)
	return err
}

// TouchChatSession 更新会话时间戳
func (s *Store) TouchChatSession(id string) error {
	_, err := s.db.Exec(
		`UPDATE chat_sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// DeleteChatSession 删除会话及其消息
func (s *Store) DeleteChatSession(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM chat_messages WHERE session_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM chat_sessions WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// GetChatMessages 获取会话的所有消息
func (s *Store) GetChatMessages(sessionID string) ([]ChatMessage, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, role, content, applied_rules, rule_added, seq, created_at
		 FROM chat_messages WHERE session_id = ? ORDER BY seq`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []ChatMessage
	for rows.Next() {
		var m ChatMessage
		var appliedRules, ruleAdded sql.NullString
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content,
			&appliedRules, &ruleAdded, &m.Seq, &m.CreatedAt); err != nil {
			return nil, err
		}
		if appliedRules.Valid {
			m.AppliedRules = json.RawMessage(appliedRules.String)
		}
		if ruleAdded.Valid {
			m.RuleAdded = json.RawMessage(ruleAdded.String)
		}
		messages = append(messages, m)
	}
	return messages, nil
}

// SaveChatMessage 保存单条消息
func (s *Store) SaveChatMessage(msg ChatMessage) error {
	var appliedRules, ruleAdded *string
	if len(msg.AppliedRules) > 0 {
		s := string(msg.AppliedRules)
		appliedRules = &s
	}
	if len(msg.RuleAdded) > 0 {
		s := string(msg.RuleAdded)
		ruleAdded = &s
	}
	_, err := s.db.Exec(
		`INSERT INTO chat_messages (id, session_id, role, content, applied_rules, rule_added, seq)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET content = ?, applied_rules = ?, rule_added = ?`,
		msg.ID, msg.SessionID, msg.Role, msg.Content, appliedRules, ruleAdded, msg.Seq,
		msg.Content, appliedRules, ruleAdded)
	return err
}

// GetChatSessionExists 检查会话是否存在
func (s *Store) GetChatSessionExists(id string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM chat_sessions WHERE id = ?`, id).Scan(&count)
	return count > 0, err
}
