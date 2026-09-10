package history

import (
	"testing"
)

// TestHistoryRoundtrip: 历史读写顺序 (created_at ASC, updated_at ASC) + is_deleted 布尔化。
func TestHistoryRoundtrip(t *testing.T) {
	m, err := NewManager(":memory:")
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}
	defer m.Close()
	ca1, ca2 := "2026-01-01T00:00:00.000000+00:00", "2026-01-02T00:00:00.000000+00:00"
	old := "old text"
	if err := m.AddHistory(AddHistoryRecord{MemoryID: "m1", NewMemory: &old, Event: "ADD", CreatedAt: &ca1}); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	if err := m.AddHistory(AddHistoryRecord{MemoryID: "m1", OldMemory: &old, Event: "DELETE", CreatedAt: &ca2, IsDeleted: 1}); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	rows, err := m.GetHistory("m1")
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(rows) != 2 || rows[0].Event != "ADD" || rows[1].Event != "DELETE" {
		t.Fatalf("顺序不符: %+v", rows)
	}
	if rows[1].IsDeleted != true || rows[0].IsDeleted != false {
		t.Errorf("is_deleted 必须布尔化 (Python bool(r[7]))")
	}
}

// TestMessagesEviction: 每个 scope 仅保留最近 10 条 (storage.py 语义)。
func TestMessagesEviction(t *testing.T) {
	m, _ := NewManager(":memory:")
	defer m.Close()
	for i := 0; i < 15; i++ {
		msgs := []map[string]any{{"role": "user", "content": string(rune('a' + i))}}
		if err := m.SaveMessages(msgs, "scope-1"); err != nil {
			t.Fatalf("写入失败: %v", err)
		}
	}
	rows, err := m.GetLastMessages("scope-1", 10)
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	if len(rows) != 10 {
		t.Fatalf("应恰好保留 10 条: %d", len(rows))
	}
	if rows[9].Content == nil || *rows[9].Content != "o" {
		t.Errorf("最新一条应为第 15 次写入: %+v", rows[9])
	}
}

// TestBatchAddHistory: 批量写 + Reset 清表。
func TestBatchAddHistory(t *testing.T) {
	m, _ := NewManager(":memory:")
	defer m.Close()
	recs := make([]AddHistoryRecord, 0, 3)
	for i := 0; i < 3; i++ {
		nt := "n"
		recs = append(recs, AddHistoryRecord{MemoryID: "m", NewMemory: &nt, Event: "ADD"})
	}
	if err := m.BatchAddHistory(recs); err != nil {
		t.Fatalf("批量失败: %v", err)
	}
	rows, _ := m.GetHistory("m")
	if len(rows) != 3 {
		t.Fatalf("批量应有 3 行: %d", len(rows))
	}
	if err := m.Reset(); err != nil {
		t.Fatalf("重置失败: %v", err)
	}
	rows, _ = m.GetHistory("m")
	if len(rows) != 0 {
		t.Errorf("重置后应清空")
	}
}
