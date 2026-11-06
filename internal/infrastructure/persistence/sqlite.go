package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"eventdispatcher/pkg/errcode"

	_ "modernc.org/sqlite"
)

// DB 封装 SQLite 数据库实例与仓储实现。
type DB struct {
	raw *sql.DB
}

// Open 初始化并连接 SQLite 数据库，强制配置 WAL 模式与单连接防止并发死锁。
func Open(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("创建数据库存储目录失败: %w", err)
	}

	dsn := dbPath + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
	rawDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite 失败: %w", err)
	}

	// 核心：单连接保证写事务严格串行化，彻底消除 "database is locked"
	rawDB.SetMaxOpenConns(1)
	rawDB.SetMaxIdleConns(1)
	rawDB.SetConnMaxLifetime(0)

	for _, stmt := range SchemaDDL {
		if _, err := rawDB.Exec(stmt); err != nil {
			rawDB.Close()
			return nil, fmt.Errorf("执行数据库初始化 DDL 失败: %w", err)
		}
	}

	return &DB{raw: rawDB}, nil
}

// Close 安全关闭数据库连接池。
func (db *DB) Close() error {
	return db.raw.Close()
}

// Ping 检测数据库连通性。
func (db *DB) Ping(ctx context.Context) error {
	return db.raw.PingContext(ctx)
}

// Raw 返回底层 *sql.DB。
func (db *DB) Raw() *sql.DB {
	return db.raw
}

func formatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return errcode.ErrNotFound
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "constraint") || strings.Contains(msg, "unique") {
		return errcode.ErrConflict
	}
	return err
}
