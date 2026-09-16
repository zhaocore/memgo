package config

import "testing"

// TestDeepMergeNested: POST /configure 语义 — dict 递归合并, 标量/数组整体覆盖。
func TestDeepMergeNested(t *testing.T) {
	base := map[string]any{
		"llm":  map[string]any{"provider": "openai", "config": map[string]any{"model": "gpt-5-mini", "temperature": 0.2}},
		"tags": []any{"a", "b"},
	}
	merged := DeepMerge(base, map[string]any{
		"llm":  map[string]any{"config": map[string]any{"model": "stub-model"}},
		"tags": []any{"c"},
	})
	llmCfg := merged["llm"].(map[string]any)["config"].(map[string]any)
	if llmCfg["model"] != "stub-model" {
		t.Errorf("嵌套键应被覆盖: %v", llmCfg)
	}
	if llmCfg["temperature"] != 0.2 {
		t.Errorf("兄弟键必须保留 (深合并非替换): %v", llmCfg)
	}
	if merged["llm"].(map[string]any)["provider"] != "openai" {
		t.Errorf("上层键必须保留")
	}
	if len(merged["tags"].([]any)) != 1 {
		t.Errorf("数组必须整体替换: %v", merged["tags"])
	}
	if base["tags"].([]any)[0] != "a" {
		t.Errorf("DeepMerge 不得改入参")
	}
}

// TestRedactSensitiveKeys: 敏感键大小写不敏感 → "[redacted]" (GET /configure 合同)。
func TestRedactSensitiveKeys(t *testing.T) {
	cfg := map[string]any{
		"api_key": "sk-secret",
		"vector_store": map[string]any{"config": map[string]any{
			"password": "pw", "host": "postgres",
		}},
		"KEYS": []any{map[string]any{"Token": "t1"}, "plain"},
		"note": "",
	}
	out := Redact(cfg, "").(map[string]any)
	if out["api_key"] != "[redacted]" {
		t.Errorf("api_key 应脱敏: %v", out["api_key"])
	}
	vsc := out["vector_store"].(map[string]any)["config"].(map[string]any)
	if vsc["password"] != "[redacted]" || vsc["host"] != "postgres" {
		t.Errorf("嵌套脱敏不符: %v", vsc)
	}
	keys := out["KEYS"].([]any)
	if keys[0].(map[string]any)["Token"] != "[redacted]" {
		t.Errorf("list 内 dict 递归脱敏: %v", keys)
	}
	if keys[1] != "plain" {
		t.Errorf("list 内标量不脱敏")
	}
}

// TestParseMemoryConfigDefaults: 缺省 provider 与 history 路径回退。
func TestParseMemoryConfigDefaults(t *testing.T) {
	cfg, err := ParseMemoryConfig(nil)
	if err != nil {
		t.Fatalf("nil 配置必须可解析: %v", err)
	}
	if cfg.Version != "v1.1" {
		t.Errorf("默认 version v1.1: %v", cfg.Version)
	}
	if cfg.LLM.Provider != "openai" || cfg.Embedder.Provider != "openai" {
		t.Errorf("默认 provider openai: %v/%v", cfg.LLM.Provider, cfg.Embedder.Provider)
	}
	if cfg.HistoryDBPath == "" {
		t.Errorf("history_db_path 必须有默认值")
	}
	if _, err := ParseMemoryConfig(map[string]any{"vector_store": "not-a-map"}); err == nil {
		t.Errorf("非法 vector_store 必须报错")
	}
}

// TestParseCategories: wire 形态 [{"分类名": "描述"}] 的合法/非法面。
func TestParseCategories(t *testing.T) {
	ok, err := ParseCategories([]any{
		map[string]any{"lifestyle_management": "Tracks daily routines"},
		map[string]any{"seeking_structure": "Documents goals"},
	})
	if err != nil || len(ok) != 2 || ok[0].Name != "lifestyle_management" {
		t.Fatalf("合法输入应解析: %v %v", ok, err)
	}
	if ok[1].Description != "Documents goals" {
		t.Errorf("描述应回填: %v", ok[1])
	}
	empty, err := ParseCategories([]any{})
	if err != nil || len(empty) != 0 {
		t.Fatalf("空列表合法 (显式清空): %v %v", empty, err)
	}
	if _, err := ParseCategories("not-a-list"); err == nil {
		t.Errorf("非列表必须报错")
	}
	if _, err := ParseCategories([]any{"item"}); err == nil {
		t.Errorf("列表内非对象必须报错")
	}
	if _, err := ParseCategories([]any{map[string]any{"a": "1", "b": "2"}}); err == nil {
		t.Errorf("多项对象必须报错 (恰好一个键)")
	}
	if _, err := ParseCategories([]any{map[string]any{"cat": 42}}); err == nil {
		t.Errorf("非字符串描述必须报错")
	}
	if _, err := ParseCategories([]any{map[string]any{"  ": "desc"}}); err == nil {
		t.Errorf("空白分类名必须报错")
	}
}
