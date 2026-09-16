package memory

// 分类打标测试: 目录解析顺序 (per-call 整体替换 config)、payload/响应落标、
// 功能关闭时零痕 (契约基线)、LLM 输出异常显式报错。

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/zhao-core/memgo/core/config"
	"github.com/zhao-core/memgo/core/history"
)

// newCategoryTestMemory: 与 newTestMemory 同构, 但注入自定义 LLM 桩 (队列响应 + 调用捕获)。
func newCategoryTestMemory(t *testing.T, llm *fakeLLM, cfg *config.MemoryConfig) *Memory {
	t.Helper()
	historyPath := filepath.Join(t.TempDir(), "history.db")
	db, err := history.NewManager(historyPath)
	if err != nil {
		t.Fatalf("history 初始化失败: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if cfg == nil {
		cfg = &config.MemoryConfig{
			Version: "v1.1",
			VectorStore: config.ProviderConfig{Provider: "pgvector", Config: map[string]any{
				"collection_name": "memories",
			}},
			LLM:      config.ProviderConfig{Provider: "openai"},
			Embedder: config.ProviderConfig{Provider: "openai"},
		}
	}
	return New(cfg, newFakeVectorStore(), &fakeEmbedder{dims: 8}, llm, db, nil)
}

// categoryClassResp 按 id 顺序生成分类 LLM 响应。
func categoryClassResp(cats ...string) string {
	var b string = `{"categories":[`
	for i, c := range cats {
		if i > 0 {
			b += ","
		}
		b += `{"id":` + itoa(i+1) + `,"category":"` + c + `"}`
	}
	return b + "]}"
}

func itoa(n int) string { return strconv.Itoa(n) }

// TestAddCategoriesPerCallOverridesConfig: per-call 非空目录整体替换 config 级, 不合并。
func TestAddCategoriesPerCallOverridesConfig(t *testing.T) {
	configCats := []config.Category{
		{Name: "food", Description: "CONFIG-LEVEL-DESC-food"},
		{Name: "travel", Description: "CONFIG-LEVEL-DESC-travel"},
	}
	llm := &fakeLLM{
		queue: []string{extractionResponse, categoryClassResp("food", "travel")},
	}
	m := newCategoryTestMemory(t, llm, &config.MemoryConfig{
		Version: "v1.1",
		VectorStore: config.ProviderConfig{Provider: "pgvector", Config: map[string]any{
			"collection_name": "memories",
		}},
		LLM:              config.ProviderConfig{Provider: "openai"},
		Embedder:         config.ProviderConfig{Provider: "openai"},
		CustomCategories: configCats,
	})
	results, err := m.Add([]map[string]any{
		{"role": "user", "content": "I love ramen"},
		{"role": "assistant", "content": "Noted"},
	}, AddParams{
		UserID: "alice",
		Infer:  true,
		CustomCategories: []config.Category{
			{Name: "food", Description: "PER-CALL-DESC-food"},
			{Name: "travel", Description: "PER-CALL-DESC-travel"},
		},
	})
	if err != nil {
		t.Fatalf("add 失败: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("期望 2 条新记忆, 得 %d", len(results))
	}
	if results[0]["category"] != "food" {
		t.Errorf("响应应含打标结果, 得 %v", results[0]["category"])
	}
	row, _ := m.VectorStore.Get(results[0]["id"].(string))
	if row.Payload["category"] != "food" {
		t.Errorf("payload 应含 category=food, 得 %v", row.Payload["category"])
	}
	if len(llm.calls) != 2 {
		t.Fatalf("期望 抽取+分类 两次 LLM 调用, 得 %d", len(llm.calls))
	}
	// 分类 prompt 必须用 per-call 目录 (整体替换), 不得混入 config 级描述。
	if len(llm.calls) != 2 {
		t.Fatalf("期望 2 次 LLM 调用, 得 %d", len(llm.calls))
	}
	classPrompt := fmt.Sprintf("%v", llm.calls[1][1].Content)
	if strings.Contains(classPrompt, "CONFIG-LEVEL-DESC") {
		t.Errorf("per-call 传入后分类 prompt 不应包含 config 级描述")
	}
	if !strings.Contains(classPrompt, "PER-CALL-DESC-food") || !strings.Contains(classPrompt, "PER-CALL-DESC-travel") {
		t.Errorf("分类 prompt 应包含 per-call 目录: %s", classPrompt)
	}
}

// TestAddCategoriesConfigLevel: 未传 per-call 时回落 config 级目录, payload/响应均落标。
func TestAddCategoriesConfigLevel(t *testing.T) {
	llm := &fakeLLM{
		queue: []string{extractionResponse, categoryClassResp("food", "travel")},
	}
	m := newCategoryTestMemory(t, llm, &config.MemoryConfig{
		Version: "v1.1",
		VectorStore: config.ProviderConfig{Provider: "pgvector", Config: map[string]any{
			"collection_name": "memories",
		}},
		LLM:      config.ProviderConfig{Provider: "openai"},
		Embedder: config.ProviderConfig{Provider: "openai"},
		CustomCategories: []config.Category{
			{Name: "food", Description: "dining preferences"},
			{Name: "travel", Description: "trips and destinations"},
		},
	})
	results, err := m.Add([]map[string]any{
		{"role": "user", "content": "I love ramen"},
		{"role": "assistant", "content": "Noted"},
	}, AddParams{UserID: "alice", Infer: true})
	if err != nil {
		t.Fatalf("add 失败: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("期望 2 条, 得 %d", len(results))
	}
	if results[1]["category"] != "travel" {
		t.Errorf("响应应含 config 级打标, 得 %v", results[1]["category"])
	}
	row, _ := m.VectorStore.Get(results[1]["id"].(string))
	if row.Payload["category"] != "travel" {
		t.Errorf("payload 应含 category=travel, 得 %v", row.Payload["category"])
	}
	if len(llm.calls) != 2 {
		t.Errorf("期望 2 次 LLM 调用, 得 %d", len(llm.calls))
	}
}

// TestAddCategoriesDisabled: 目录未配置 = 功能关闭 — 仅 1 次 LLM 调用 (无分类),
// payload/响应无 category 键, 与 Python 契约基线形状逐字节一致。
func TestAddCategoriesDisabled(t *testing.T) {
	llm := &fakeLLM{response: extractionResponse}
	m := newCategoryTestMemory(t, llm, nil)
	results, err := m.Add([]map[string]any{
		{"role": "user", "content": "I love ramen"},
	}, AddParams{UserID: "alice", Infer: true})
	if err != nil {
		t.Fatalf("add 失败: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("期望 2 条, 得 %d", len(results))
	}
	for _, r := range results {
		if _, has := r["category"]; has {
			t.Errorf("功能关闭时响应不应含 category 键")
		}
		row, _ := m.VectorStore.Get(r["id"].(string))
		if _, has := row.Payload["category"]; has {
			t.Errorf("功能关闭时 payload 不应含 category 键")
		}
	}
	if len(llm.calls) != 1 {
		t.Errorf("功能关闭应只有抽取 1 次 LLM 调用, 得 %d", len(llm.calls))
	}
}

// TestAddCategoriesBadLLMOutput: 分类输出缺项 → 整个 add 显式失败, 不静默落库。
func TestAddCategoriesBadLLMOutput(t *testing.T) {
	llm := &fakeLLM{
		queue: []string{
			extractionResponse,
			`{"categories": [{"id": 1, "category": "food"}]}`,
		},
	}
	m := newCategoryTestMemory(t, llm, &config.MemoryConfig{
		Version: "v1.1",
		VectorStore: config.ProviderConfig{Provider: "pgvector", Config: map[string]any{
			"collection_name": "memories",
		}},
		LLM:      config.ProviderConfig{Provider: "openai"},
		Embedder: config.ProviderConfig{Provider: "openai"},
		CustomCategories: []config.Category{
			{Name: "food", Description: "dining preferences"},
			{Name: "travel", Description: "trips and destinations"},
		},
	})
	_, err := m.Add([]map[string]any{
		{"role": "user", "content": "I love ramen"},
		{"role": "assistant", "content": "Noted"},
	}, AddParams{UserID: "alice", Infer: true})
	if err == nil {
		t.Fatalf("分类输出缺 memory 2 条目, 必须显式报错")
	}
	if !strings.Contains(err.Error(), "missing entry for memory 2") {
		t.Errorf("错误信息应指明缺失条目, 得: %v", err)
	}
}
