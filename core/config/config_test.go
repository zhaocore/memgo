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
