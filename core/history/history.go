// Package history: SQLite 历史库, 逐列对齐 mem0/memory/storage.py (history + messages 表)。
package history

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// Manager 对齐 Python SQLiteManager (单连接 + 锁语义: MaxOpenConns=1)。
type Manager struct {
	db *sql.DB
	mu sync.Mutex
}

// NewManager 打开数据库并建表 (history/messages), 等价 __init__。
// Python 默认 ":memory:"; 调用方显式传路径。
func NewManager(dbPath string) (*Manager, error) {
	dsn := dbPath
	if strings.Contains(dbPath, ":memory:") {
		dsn = "file::memory:?cache=shared"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("history: 打开失败 %s: %w", dbPath, err)
	}
	db.SetMaxOpenConns(1)
	m := &Manager{db: db}
	if err := m.createTables(); err != nil {
		db.Close()
		return nil, err
	}
	return m, nil
}

const historyDDL = `CREATE TABLE IF NOT EXISTS history (
	id           TEXT PRIMARY KEY,
	memory_id    TEXT,
	old_memory   TEXT,
	new_memory   TEXT,
	event        TEXT,
	created_at   DATETIME,
	updated_at   DATETIME,
	is_deleted   INTEGER,
	actor_id     TEXT,
	role         TEXT
)`

const messagesDDL = `CREATE TABLE IF NOT EXISTS messages (
	id TEXT PRIMARY KEY,
	session_scope TEXT,
	role TEXT,
	content TEXT,
	name TEXT,
	created_at DATETIME
)`

func (m *Manager) createTables() error {
	for _, ddl := range []string{historyDDL, messagesDDL} {
		if _, err := m.db.Exec(ddl); err != nil {
			return fmt.Errorf("history: 建表失败: %w", err)
		}
	}
	return nil
}

// AddHistoryRecord 对齐 add_history 参数面。
type AddHistoryRecord struct {
	MemoryID  string
	OldMemory *string
	NewMemory *string
	Event     string
	CreatedAt *string
	UpdatedAt *string
	IsDeleted int
	ActorID   *string
	Role      *string
}

// AddHistory 插入一条历史 (id 为随机 UUID)。
func (m *Manager) AddHistory(r AddHistoryRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, err := m.db.Exec(`INSERT INTO history (
		id, memory_id, old_memory, new_memory, event, created_at, updated_at, is_deleted, actor_id, role
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		newUUID(), r.MemoryID, r.OldMemory, r.NewMemory, r.Event,
		r.CreatedAt, r.UpdatedAt, r.IsDeleted, r.ActorID, r.Role)
	if err != nil {
		return fmt.Errorf("history: 写入失败: %w", err)
	}
	return nil
}

// BatchAddHistory 对齐 batch_add_history (records 字段与 AddHistoryRecord 一致)。
func (m *Manager) BatchAddHistory(records []AddHistoryRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("history: 开事务失败: %w", err)
	}
	stmt, err := tx.Prepare(`INSERT INTO history (
		id, memory_id, old_memory, new_memory, event, created_at, updated_at, is_deleted, actor_id, role
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("history: 预编译失败: %w", err)
	}
	for _, r := range records {
		if _, err := stmt.Exec(newUUID(), r.MemoryID, r.OldMemory, r.NewMemory, r.Event,
			r.CreatedAt, r.UpdatedAt, r.IsDeleted, r.ActorID, r.Role); err != nil {
			_ = stmt.Close()
			_ = tx.Rollback()
			return fmt.Errorf("history: 批量写入失败: %w", err)
		}
	}
	if err := stmt.Close(); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("history: 提交失败: %w", err)
	}
	return nil
}

// HistoryRow 对齐 get_history 返回行 (is_deleted 转 bool)。
type HistoryRow struct {
	ID        string
	MemoryID  string
	OldMemory *string
	NewMemory *string
	Event     string
	CreatedAt *string
	UpdatedAt *string
	IsDeleted bool
	ActorID   *string
	Role      *string
}

// GetHistory 对齐 get_history: ORDER BY created_at ASC, DATETIME(updated_at) ASC。
func (m *Manager) GetHistory(memoryID string) ([]HistoryRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows, err := m.db.Query(`SELECT id, memory_id, old_memory, new_memory, event,
		created_at, updated_at, is_deleted, actor_id, role
		FROM history WHERE memory_id = ?
		ORDER BY created_at ASC, DATETIME(updated_at) ASC`, memoryID)
	if err != nil {
		return nil, fmt.Errorf("history: 查询失败: %w", err)
	}
	defer rows.Close()
	var out []HistoryRow
	for rows.Next() {
		var r HistoryRow
		var deleted int
		if err := rows.Scan(&r.ID, &r.MemoryID, &r.OldMemory, &r.NewMemory, &r.Event,
			&r.CreatedAt, &r.UpdatedAt, &deleted, &r.ActorID, &r.Role); err != nil {
			return nil, fmt.Errorf("history: 扫描失败: %w", err)
		}
		r.IsDeleted = deleted != 0
		out = append(out, r)
	}
	return out, rows.Err()
}

// SaveMessages 对齐 save_messages: 写入后驱逐超出最近 10 条的旧行。
func (m *Manager) SaveMessages(messages []map[string]any, sessionScope string) error {
	if len(messages) == 0 {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000000+00:00")
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("history: 开事务失败: %w", err)
	}
	for _, msg := range messages {
		role, _ := msg["role"].(string)
		content, _ := msg["content"].(string)
		var name *string
		if n, ok := msg["name"].(string); ok {
			name = &n
		}
		if _, err := tx.Exec(`INSERT INTO messages (id, session_scope, role, content, name, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`, newUUID(), sessionScope, role, content, name, now); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("history: 消息写入失败: %w", err)
		}
	}
	// 驱逐超出最近 10 条的旧行 (派生表强制先排序, 对齐 Python 注释)
	if _, err := tx.Exec(`DELETE FROM messages WHERE session_scope = ? AND id NOT IN (
		SELECT id FROM (
			SELECT id FROM messages WHERE session_scope = ? ORDER BY created_at DESC LIMIT 10
		)
	)`, sessionScope, sessionScope); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("history: 消息驱逐失败: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("history: 提交失败: %w", err)
	}
	return nil
}

// MessageRow 对齐 get_last_messages 返回行。
type MessageRow struct {
	Role      string
	Content   *string
	Name      *string
	CreatedAt *string
}

// GetLastMessages 对齐 get_last_messages: 取最新 N 条后按时间正序返回。
func (m *Manager) GetLastMessages(sessionScope string, limit int) ([]MessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rows, err := m.db.Query(`SELECT role, content, name, created_at FROM (
		SELECT role, content, name, created_at FROM messages
		WHERE session_scope = ? ORDER BY created_at DESC LIMIT ?
	) ORDER BY created_at ASC`, sessionScope, limit)
	if err != nil {
		return nil, fmt.Errorf("history: 查询失败: %w", err)
	}
	defer rows.Close()
	var out []MessageRow
	for rows.Next() {
		var r MessageRow
		if err := rows.Scan(&r.Role, &r.Content, &r.Name, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("history: 扫描失败: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Reset 对齐 reset: 删两张表。
func (m *Manager) Reset() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := m.db.Exec("DROP TABLE IF EXISTS history"); err != nil {
		return fmt.Errorf("history: 重置失败: %w", err)
	}
	if _, err := m.db.Exec("DROP TABLE IF EXISTS messages"); err != nil {
		return fmt.Errorf("history: 重置失败: %w", err)
	}
	return nil
}

// Close 关闭连接。
func (m *Manager) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}
