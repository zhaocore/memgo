package memory

import (
	"crypto/md5"
	"encoding/hex"
	"regexp"
	"testing"

	"github.com/zhao-core/memgo/core/config"
	"github.com/zhao-core/memgo/core/history"
	"github.com/zhao-core/memgo/core/jsonx"
)

// newTestMemory 组装桩注入的 Memory (P1 验收: LLM/embedder 接口桩注入)。
func newTestMemory(t *testing.T, llmResponse string) *Memory {
	t.Helper()
	db, err := history.NewManager(":memory:")
	if err != nil {
		t.Fatalf("history 初始化失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	cfg := &config.MemoryConfig{
		Version: "v1.1",
		VectorStore: config.ProviderConfig{Provider: "pgvector", Config: map[string]any{
			"collection_name": "memories",
		}},
		LLM:      config.ProviderConfig{Provider: "openai"},
		Embedder: config.ProviderConfig{Provider: "openai"},
	}
	return New(cfg, newFakeVectorStore(), &fakeEmbedder{dims: 8}, &fakeLLM{response: llmResponse}, db, nil)
}

// extractionResponse 模拟契约 stub 的固定抽取输出。
const extractionResponse = `{"memory":[{"text":"Lives in Berlin"},{"text":"Enjoys hiking"}]}`

var uuidRe = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// isoTS: Python isoformat 形状 (微秒 6 位 + "+00:00") — 合同红线 T3。
var isoTSRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{6}\+00:00$`)

// TestAddInferFalseDeterministic: infer=false 与 Python 对拍基线 (P1 验收跨语言对拍的桩级等价)。
// 断言: event=ADD、UUID v4、hash=md5(原文)、时间戳 isoformat、role/actor_id 提升。
func TestAddInferFalseDeterministic(t *testing.T) {
	m := newTestMemory(t, extractionResponse)
	results, err := m.Add([]map[string]any{
		{"role": "user", "content": "I live in Berlin"},
		{"role": "system", "content": "must be skipped"},
		{"role": "assistant", "content": "Hello!", "name": "bot-1"},
	}, AddParams{UserID: "alice", Infer: false})
	if err != nil {
		t.Fatalf("add 失败: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("system 消息必须跳过: 期望 2 条, 得 %d", len(results))
	}
	r0 := results[0]
	if r0["event"] != "ADD" {
		t.Errorf("infer=false 恒 ADD, 得 %v", r0["event"])
	}
	if !uuidRe.MatchString(r0["id"].(string)) {
		t.Errorf("id 须为 UUID v4: %v", r0["id"])
	}
	wantHash := md5.Sum([]byte("I live in Berlin"))
	if r0["memory"].(string) != "I live in Berlin" {
		t.Errorf("memory 应为原文")
	}
	row, err := m.VectorStore.Get(r0["id"].(string))
	if err != nil || row == nil {
		t.Fatalf("向量行不存在: %v", err)
	}
	if row.Payload["hash"] != hex.EncodeToString(wantHash[:]) {
		t.Errorf("hash 必须等于 md5(原文): %v", row.Payload["hash"])
	}
	if !isoTSRe.MatchString(row.Payload["created_at"].(string)) {
		t.Errorf("created_at 须为 Python isoformat 形状: %v", row.Payload["created_at"])
	}
	if row.Payload["updated_at"] != row.Payload["created_at"] {
		t.Errorf("新建记忆 updated_at=created_at")
	}
	if row.Payload["user_id"] != "alice" {
		t.Errorf("scope user_id 必须落 payload")
	}
	if row.Payload["role"] != "user" {
		t.Errorf("role 必须提升进 payload")
	}
	if _, has := row.Payload["actor_id"]; has {
		t.Errorf("无 name 的消息不应有 actor_id")
	}
	r1 := results[1]
	if r1["actor_id"] != "bot-1" {
		t.Errorf("name 应提升为 actor_id: %v", r1["actor_id"])
	}
	// 历史 ADD 两行
	h, err := m.History(r0["id"].(string))
	if err != nil || len(h) != 1 || h[0].Event != "ADD" {
		t.Fatalf("历史应恰一条 ADD: %v %v", h, err)
	}
}

// TestAddInferredHashDedup: infer=true 阶段化流水线 — 同事实重复加入被 hash 去重。
func TestAddInferredHashDedup(t *testing.T) {
	m := newTestMemory(t, extractionResponse)
	first, err := m.Add([]map[string]any{{"role": "user", "content": "I live in Berlin"}}, AddParams{UserID: "u", Infer: true})
	if err != nil {
		t.Fatalf("首次 add 失败: %v", err)
	}
	if len(first) != 2 {
		t.Fatalf("stub 返回 2 事实: 得 %d", len(first))
	}
	second, err := m.Add([]map[string]any{{"role": "user", "content": "I live in Berlin"}}, AddParams{UserID: "u", Infer: true})
	if err != nil {
		t.Fatalf("重复 add 失败: %v", err)
	}
	if len(second) != 0 {
		t.Fatalf("相同事实必须被 hash 去重: 得 %d 条", len(second))
	}
}

func strPtr(s string) *string { return &s }

// TestUpdatePartialSemantics: 红线 T1 — fields_set 语义。
// 只传 metadata 不动内容; expiration null 清除; {} 与三空 → 精确错误文案。
func TestUpdatePartialSemantics(t *testing.T) {
	m := newTestMemory(t, extractionResponse)
	added, err := m.Add([]map[string]any{{"role": "user", "content": "I like hiking"}}, AddParams{
		UserID: "alice", Infer: false, Metadata: map[string]any{"pref": "hiking"},
		ExpirationDate: "2099-01-01",
	})
	if err != nil {
		t.Fatalf("add 失败: %v", err)
	}
	id := added[0]["id"].(string)

	// 只传 metadata: 内容/hash 不变, metadata 替换
	if _, err := m.Update(UpdateParams{MemoryID: id, Metadata: map[string]any{"pref": "trail"}}); err != nil {
		t.Fatalf("metadata-only update 失败: %v", err)
	}
	row, _ := m.VectorStore.Get(id)
	if row.Payload["data"] != "I like hiking" {
		t.Errorf("只传 metadata 不得改内容: %v", row.Payload["data"])
	}
	if row.Payload["pref"] != "trail" {
		t.Errorf("metadata 应已替换: %v", row.Payload["pref"])
	}
	if row.Payload["expiration_date"] != "2099-01-01" {
		t.Errorf("未传 expiration 不得清除: %v", row.Payload["expiration_date"])
	}

	// identity 键经 metadata 传入被剥离
	if _, err := m.Update(UpdateParams{MemoryID: id, Metadata: map[string]any{"user_id": "mallory"}}); err != nil {
		t.Fatalf("identity update 失败: %v", err)
	}
	row, _ = m.VectorStore.Get(id)
	if row.Payload["user_id"] != "alice" {
		t.Errorf("identity 键不可经 metadata 改写: %v", row.Payload["user_id"])
	}

	// expiration 显式 null → 清除
	if _, err := m.Update(UpdateParams{MemoryID: id, ExpirationSet: true, ExpirationDate: nil}); err != nil {
		t.Fatalf("expiration 清除失败: %v", err)
	}
	row, _ = m.VectorStore.Get(id)
	if v, ok := row.Payload["expiration_date"]; ok && v != nil {
		t.Errorf("expiration 应为 null, 得 %v", v)
	}

	// 空参数 → Python 文案逐字
	_, err = m.Update(UpdateParams{MemoryID: id})
	if err == nil || err.Error() != "At least one of text, metadata, or expiration_date must be provided." {
		t.Errorf("空参数错误文案不符: %v", err)
	}

	// 不存在 id → Python 文案逐字 (server 映射 404)
	_, err = m.Update(UpdateParams{MemoryID: "00000000-0000-0000-0000-000000000000", Text: strPtr("x")})
	if err == nil || err.Error() != "Memory with id 00000000-0000-0000-0000-000000000000 not found. Please provide a valid 'memory_id'" {
		t.Errorf("缺失 id 错误文案不符: %v", err)
	}
}

// TestDeleteContract: 删除文案与 not-found 文案 (server 404 映射依据)。
func TestDeleteContract(t *testing.T) {
	m := newTestMemory(t, extractionResponse)
	added, _ := m.Add([]map[string]any{{"role": "user", "content": "temp"}}, AddParams{UserID: "u", Infer: false})
	id := added[0]["id"].(string)
	res, err := m.Delete(id)
	if err != nil || res["message"] != "Memory deleted successfully!" {
		t.Fatalf("删除文案不符: %v %v", res, err)
	}
	h, _ := m.History(id)
	if len(h) != 2 || h[1].Event != "DELETE" || h[1].IsDeleted != true {
		t.Fatalf("删除后历史应含 DELETE 行 (is_deleted=1): %v", h)
	}
	_, err = m.Delete(id)
	want := "Memory with id " + id + " not found"
	if err == nil || err.Error() != want {
		t.Errorf("not-found 文案不符: %v", err)
	}
}

// TestDeleteAllRequiresFilter: 无过滤 → Python 文案逐字。
func TestDeleteAllRequiresFilter(t *testing.T) {
	m := newTestMemory(t, extractionResponse)
	_, err := m.DeleteAll(DeleteAllParams{})
	want := "At least one filter is required to delete all memories. If you want to delete all memories, use the `reset()` method."
	if err == nil || err.Error() != want {
		t.Errorf("文案不符: %v", err)
	}
}

// TestSearchScoringAndShape: 混合评分 + 序列化形状 (score/metadata/promoted 键)。
// 恒定桩向量 → 语义分 1.0; BM25 缺席 (fake store 无 keyword) → combined = 1.0/1.0。
func TestSearchScoringAndShape(t *testing.T) {
	m := newTestMemory(t, extractionResponse)
	if _, err := m.Add([]map[string]any{{"role": "user", "content": "Alice enjoys swimming"}}, AddParams{
		UserID: "alice", Infer: false, Metadata: map[string]any{"pref": "swim"},
	}); err != nil {
		t.Fatalf("add 失败: %v", err)
	}
	threshold := 0.5
	res, err := m.Search(SearchParams{Query: "swimming", Filters: map[string]any{"user_id": "alice"}, TopK: 5, Threshold: &threshold})
	if err != nil {
		t.Fatalf("search 失败: %v", err)
	}
	results := res["results"].([]map[string]any)
	if len(results) != 1 {
		t.Fatalf("应命中 1 条: %d", len(results))
	}
	r := results[0]
	if float64(r["score"].(jsonx.PyFloat)) < 0.99 {
		t.Errorf("恒定嵌入下 score 应为 1.0 封顶: %v", r["score"])
	}
	if _, has := r["metadata"]; !has {
		t.Errorf("search 序列化必须含 metadata 键 (null 或填充)")
	}
	if r["pref"] != nil {
		t.Errorf("pref 属 metadata, 不得提升")
	}
	if r["metadata"].(map[string]any)["pref"] != "swim" {
		t.Errorf("metadata 应含自定义键: %v", r["metadata"])
	}
	// threshold 过滤: 语义分 1.0 > 1 → 无结果不可能; 用上界外值验证 400 语义
	th := 1.5
	if _, err := m.Search(SearchParams{Query: "x", Filters: map[string]any{"user_id": "alice"}, Threshold: &th}); err == nil {
		t.Errorf("threshold>1 应报错 (SDK ValueError → server 400)")
	}
	// 空 query → 报错
	if _, err := m.Search(SearchParams{Query: "   ", Filters: map[string]any{"user_id": "alice"}}); err == nil {
		t.Errorf("空白 query 应报错")
	}
	// 无实体过滤 → 报错 (server 400 依据)
	if _, err := m.Search(SearchParams{Query: "x"}); err == nil {
		t.Errorf("缺实体过滤应报错")
	}
}

// TestScoreAndRankThresholdGating: 阈值先滤语义分再加成 (scoring 契约)。
func TestScoreAndRankThresholdGating(t *testing.T) {
	cands := []rankCandidate{
		{id: "a", score: 0.9, payload: map[string]any{}},
		{id: "b", score: 0.05, payload: map[string]any{}},
	}
	out := scoreAndRank(cands, map[string]float64{"b": 1.0}, nil, 0.1, 10, false)
	if len(out) != 1 || out[0].id != "a" {
		t.Fatalf("低于阈值的候选即使有 BM25 加成也必须被排除: %+v", out)
	}
	// explain 细节字段齐全
	out = scoreAndRank(cands, map[string]float64{"a": 0.5}, map[string]float64{"a": 0.25}, 0.1, 10, true)
	d := out[0].details
	for _, k := range []string{"semantic_score", "bm25_score", "entity_boost", "raw_score", "max_possible_score", "final_score", "threshold"} {
		if _, ok := d[k]; !ok {
			t.Errorf("score_details 缺 %s", k)
		}
	}
	if float64(d["max_possible_score"].(jsonx.PyFloat)) != 2.5 {
		t.Errorf("semantic+bm25+entity 时 max=2.5: %v", d["max_possible_score"])
	}
}
