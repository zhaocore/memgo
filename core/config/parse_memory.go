package config

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
