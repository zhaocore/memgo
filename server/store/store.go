// Package store: 应用库 (memgo_app) 访问层 + goose 迁移。
// 红线: server 表永不进 pgvector 记忆库 (双库拓扑, doc-01 §5.5)。
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DSN 从环境变量拼装 (对齐 db.py _build_database_url)。
func DSN() string {
	host := envOr("POSTGRES_HOST", "postgres")
	port := envOr("POSTGRES_PORT", "5432")
	user := envOr("POSTGRES_USER", "postgres")
	password := envOr("POSTGRES_PASSWORD", "postgres")
	db := envOr("APP_DB_NAME", "memgo_app")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, password, host, port, db)
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// Store 持有 *sql.DB (pgx stdlib 驱动, 供 goose 与查询共用)。
type Store struct {
	DB *sql.DB
}

// Open 打开连接池 (pool_pre_ping 等价: stdlib 无内建 ping, 依赖池校验)。
func Open(dsn string) (*Store, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("appdb: 解析 DSN 失败: %w", err)
	}
	db := stdlib.OpenDB(*cfg)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("appdb: 连接失败: %w", err)
	}
	return &Store{DB: db}, nil
}

// Migrate 应用 goose 迁移 (alembic 001-006 的翻译)。
func (s *Store) Migrate(ctx context.Context) error {
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("appdb: goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, s.DB, "migrations"); err != nil {
		return fmt.Errorf("appdb: 迁移失败: %w", err)
	}
	return nil
}

// Close 关闭连接池。
func (s *Store) Close() error { return s.DB.Close() }
