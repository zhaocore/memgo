// memgo-migrate: 存量 Python (alembic) 部署 → Go server (goose) 版本表迁移工具。
//
// 原理: Python 部署的 mem0_app 已由 alembic 001-006 建表 (alembic_version 表记 head)。
// Go server 用 goose, 版本记于 goose_db_version —— 首次启动会试图重放 001, DDL 已存在即失败。
// 本工具读取 alembic_version, 确认 DDL 面 (幂等探针), 然后把 001-006 补录进 goose_db_version,
// 使 goose.Up 成为无操作。反向 (goose → alembic) 不支持: Python server 与 Go server 不应混跑同一库。
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/zhao-core/memgo/internal/dotenv"
	"github.com/zhao-core/memgo/server/store"
)

// alembicHead 上游 alembic 迁移链 head (server/alembic/versions 006)。
const alembicHead = "006"

// appliedMigrations 是 goose 侧需补录的版本号序列 (server/store/migrations 文件名前缀)。
var appliedMigrations = []int64{1, 2, 3, 4, 5, 6}

func main() {
	if err := dotenv.Load(); err != nil {
		fail("%v", err)
	}

	dsn := os.Getenv("APP_DB_DSN")
	if dsn == "" {
		dsn = store.DSN()
	}
	fmt.Printf("memgo-migrate: 目标库 %s\n", maskDSN(dsn))

	st, err := store.Open(dsn)
	if err != nil {
		fail("连接失败: %v", err)
	}
	defer st.Close()

	ctx := context.Background()

	// 1. 读 alembic_version
	var version string
	err = st.DB.QueryRowContext(ctx, `SELECT version_num FROM alembic_version LIMIT 1`).Scan(&version)
	if err == sql.ErrNoRows {
		fmt.Println("无 alembic_version 表记录 —— 库可能从未被 Python server 迁移, 直接用 goose 即可, 无需本工具。")
		os.Exit(0)
	}
	if err != nil {
		// 表不存在
		if strings.Contains(err.Error(), "does not exist") {
			fmt.Println("无 alembic_version 表 —— 非 Python 部署库, 直接用 goose 即可。")
			os.Exit(0)
		}
		fail("读 alembic_version 失败: %v", err)
	}
	fmt.Printf("alembic head: %s\n", version)
	if version != alembicHead {
		fail("alembic 版本 %s ≠ 预期 %s —— 先在 Python 侧升到 head 再迁移", version, alembicHead)
	}

	// 2. DDL 幂等探针: 五张表与两关键索引必须已存在
	for _, probe := range []string{
		`SELECT 1 FROM information_schema.tables WHERE table_name='users'`,
		`SELECT 1 FROM information_schema.tables WHERE table_name='api_keys'`,
		`SELECT 1 FROM information_schema.tables WHERE table_name='request_logs'`,
		`SELECT 1 FROM information_schema.tables WHERE table_name='settings'`,
		`SELECT 1 FROM information_schema.tables WHERE table_name='refresh_token_jtis'`,
		`SELECT 1 FROM pg_indexes WHERE indexname='ix_users_only_one_admin'`,
		`SELECT 1 FROM pg_indexes WHERE indexname='ix_request_logs_created_at'`,
	} {
		var ok bool
		if err := st.DB.QueryRowContext(ctx, probe).Scan(&ok); err != nil || !ok {
			fail("DDL 探针失败 (%s) —— 库结构与 alembic head 不符, 拒绝标记", probe)
		}
	}
	fmt.Println("DDL 探针: 5 表 + 2 索引齐备 ✓")

	// 3. 补录 goose_db_version (幂等: 已存在则跳过)
	if _, err := st.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS goose_db_version (
		id SERIAL PRIMARY KEY,
		version_id BIGINT NOT NULL,
		is_applied BOOLEAN NOT NULL,
		tstamp TIMESTAMP DEFAULT now() NULL)`); err != nil {
		fail("建 goose 版本表失败: %v", err)
	}
	var existing int
	if err := st.DB.QueryRowContext(ctx, `SELECT count(*) FROM goose_db_version WHERE is_applied = true`).Scan(&existing); err != nil {
		fail("查 goose 版本失败: %v", err)
	}
	if existing >= len(appliedMigrations) {
		fmt.Printf("goose 版本表已记 %d 条 —— 已迁移过, 无需重复。\n", existing)
		os.Exit(0)
	}
	for _, v := range appliedMigrations {
		if _, err := st.DB.ExecContext(ctx,
			`INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, true) ON CONFLICT DO NOTHING`, v); err != nil {
			fail("补录版本 %d 失败: %v", v, err)
		}
		fmt.Printf("补录 goose 版本 %d ✓\n", v)
	}
	fmt.Println("迁移完成 —— Go server 可直接启动 (goose.Up 将为无操作)。")
}

func maskDSN(dsn string) string {
	if i := strings.Index(dsn, "@"); i > 0 && strings.Contains(dsn[:i], ":") {
		return dsn[:strings.Index(dsn, "://")+3] + "***@" + dsn[i+1:]
	}
	return dsn
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "memgo-migrate: "+format+"\n", args...)
	os.Exit(1)
}
