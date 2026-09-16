package config

import (
	"fmt"
	"strings"
)

// ParseMemoryConfig 从自由 dict 解析 MemoryConfig (Python MemoryConfig(**dict) 等价)。
// 未知顶层键忽略; 子配置 provider 白名单由各 provider 解析时校验。
func ParseMemoryConfig(raw map[string]any) (*MemoryConfig, error) {
	cfg := &MemoryConfig{
		Version:       Version,
		HistoryDBPath: DefaultHistoryDBPath(),
		VectorStore:   ProviderConfig{Provider: "qdrant"},
		LLM:           ProviderConfig{Provider: "openai"},
		Embedder:      ProviderConfig{Provider: "openai"},
	}
	if raw == nil {
		return cfg, nil
	}
	if v, ok := raw["version"].(string); ok && v != "" {
		cfg.Version = v
	}
	if v, ok := raw["custom_instructions"].(string); ok {
		cfg.CustomInstructions = v
	}
	if v, ok := raw["custom_categories"]; ok {
		cats, err := ParseCategories(v)
		if err != nil {
			return nil, err
		}
		cfg.CustomCategories = cats
	}
	if v, ok := raw["history_db_path"].(string); ok && v != "" {
		cfg.HistoryDBPath = v
	}
	var err error
	if cfg.VectorStore, err = parseProvider(raw, "vector_store", "qdrant"); err != nil {
		return nil, err
	}
	if cfg.LLM, err = parseProvider(raw, "llm", "openai"); err != nil {
		return nil, err
	}
	if cfg.Embedder, err = parseProvider(raw, "embedder", "openai"); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ParseCategories 校验并解析 custom_categories 的 wire 形态
// ([{"分类名": "描述"}]: 每项必须是恰好一个键的对象, 键为非空分类名, 值为字符串描述)。
// 空列表合法 (表示显式清空)。错误信息含下标, 便于定位。
func ParseCategories(v any) ([]Category, error) {
	items, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("custom_categories must be a list of {\"category_name\": \"description\"} objects.")
	}
	out := make([]Category, 0, len(items))
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("custom_categories[%d] must be an object with exactly one \"category_name\": \"description\" pair.", i)
		}
		if len(m) != 1 {
			return nil, fmt.Errorf("custom_categories[%d] must contain exactly one \"category_name\": \"description\" pair, got %d keys.", i, len(m))
		}
		var name string
		var desc any
		for k, val := range m {
			name = k
			desc = val
		}
		d, ok := desc.(string)
		if !ok {
			return nil, fmt.Errorf("custom_categories[%d] description must be a string.", i)
		}
		if strings.TrimSpace(name) == "" {
			return nil, fmt.Errorf("custom_categories[%d] category name must be a non-empty string.", i)
		}
		out = append(out, Category{Name: name, Description: d})
	}
	return out, nil
}
