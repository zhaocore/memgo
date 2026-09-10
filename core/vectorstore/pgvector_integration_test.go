package vectorstore

import (
	"os"
	"testing"
)

// TestPGVectorIntegration: 真实 pgvector 往返 (契约栈 postgres: 宿主 8432)。
// 未运行栈或 TEST_PGVECTOR_DSN 未指 → 跳过 (CI 由 compose 提供依赖)。
func TestPGVectorIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_PGVECTOR_DSN")
	if dsn == "" {
		dsn = "postgres://postgres:contract-pw@localhost:8432/postgres"
	}
	pg, err := NewPGVector(PGVectorConfig{
		DBName: "postgres", CollectionName: "memgo_it_test", EmbeddingModelDims: 8,
		User: "postgres", Password: "contract-pw", Host: "localhost", Port: 8432,
		HNSW: false,
	})
	if err != nil {
		t.Skipf("pgvector 不可达 (契约栈未起?), 跳过: %v", err)
	}
	defer pg.Close()
	if err := pg.Reset(); err != nil {
		t.Skipf("pgvector 不可用, 跳过: %v", err)
	}
	defer func() { _ = pg.DeleteCol() }()

	vec := []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}
	if err := pg.Insert([][]float64{vec}, []string{"11111111-1111-4111-8111-111111111111"},
		[]map[string]any{{"data": "hello", "user_id": "u1"}}); err != nil {
		t.Fatalf("插入失败: %v", err)
	}
	rows, err := pg.List(nil, 100)
	if err != nil || len(rows) != 1 || len(rows[0]) != 1 {
		t.Fatalf("list 往返失败: %v %v", rows, err)
	}
	if rows[0][0].Payload["data"] != "hello" {
		t.Errorf("payload 往返失真: %v", rows[0][0].Payload)
	}
	// 过滤语义
	miss, _ := pg.List(map[string]any{"user_id": "other"}, 100)
	if len(miss[0]) != 0 {
		t.Errorf("user_id 过滤应排除")
	}
	got, err := pg.Search("q", vec, 5, map[string]any{"user_id": "u1"})
	if err != nil || len(got) != 1 {
		t.Fatalf("search 失败: %v %v", got, err)
	}
	if got[0].Score == nil || *got[0].Score < 0.999 {
		t.Errorf("自检索分数应为 1.0: %v", got[0].Score)
	}
	if err := pg.Delete("11111111-1111-4111-8111-111111111111"); err != nil {
		t.Fatalf("删除失败: %v", err)
	}
	row, _ := pg.Get("11111111-1111-4111-8111-111111111111")
	if row != nil {
		t.Errorf("删除后 Get 必须为 nil")
	}
}
