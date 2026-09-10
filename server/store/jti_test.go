package store

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testDSN 复用契约栈 postgres; 不可达则跳过。
func testStore(t *testing.T) *Store {
	t.Helper()
	st, err := Open("postgres://postgres:contract-pw@localhost:8432/postgres")
	if err != nil {
		t.Skipf("应用库不可达 (契约栈未起?), 跳过: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(context.Background()); err != nil {
		t.Skipf("迁移不可用: %v", err)
	}
	return st
}

// TestConsumeRefreshJTIConcurrentReplay: T11 — 并发重放同一 jti 恰一成功。
// CAS 语义 (条件 UPDATE + rowcount) 是防重放的核心契约; Dashboard 会话安全依赖它。
func TestConsumeRefreshJTIConcurrentReplay(t *testing.T) {
	st := testStore(t)
	ctx := context.Background()
	jti := newTestUUID()
	userID := newTestUUID()
	if _, err := st.DB.ExecContext(ctx,
		`INSERT INTO users (id, name, email, password_hash, role) VALUES ($1,'t','t@t.dev','x','admin') ON CONFLICT (id) DO NOTHING`, userID); err != nil {
		t.Fatalf("建用户失败: %v", err)
	}
	t.Cleanup(func() {
		_, _ = st.DB.ExecContext(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})
	if err := st.InsertRefreshJTI(ctx, jti, userID, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("插 jti 失败: %v", err)
	}
	const workers = 16
	var success atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := st.ConsumeRefreshJTI(ctx, jti)
			if err == nil && ok {
				success.Add(1)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 {
		t.Fatalf("并发重放必须恰一成功: %d", success.Load())
	}
}
