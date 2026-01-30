package store

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

// Store SQLite 数据库存储层
type Store struct {
	db *sql.DB
}

// New 创建新的 Store 实例
func New(dbPath string) (*Store, error) {
	// 确保 data 目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// 打开数据库连接
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// 设置连接池参数
	db.SetMaxOpenConns(1) // SQLite 建议单连接
	db.SetMaxIdleConns(1)

	store := &Store{db: db}

	// 初始化数据库结构
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema 初始化数据库结构
func (s *Store) initSchema() error {
	schemaSQL, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("failed to read schema.sql: %w", err)
	}

	// 执行建表语句
	if _, err := s.db.Exec(string(schemaSQL)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// Close 关闭数据库连接
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// DB 获取原始数据库连接（用于事务等高级操作）
func (s *Store) DB() *sql.DB {
	return s.db
}

// BeginTx 开始事务
func (s *Store) BeginTx() (*sql.Tx, error) {
	return s.db.Begin()
}

// Exec 执行 SQL 语句
func (s *Store) Exec(query string, args ...interface{}) error {
	_, err := s.db.Exec(query, args...)
	return err
}

// QueryRow 查询单行
func (s *Store) QueryRow(query string, args ...interface{}) *sql.Row {
	return s.db.QueryRow(query, args...)
}

// Query 查询多行
func (s *Store) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return s.db.Query(query, args...)
}
