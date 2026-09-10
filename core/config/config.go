// Package config 提供 MemoryConfig v1.1 的解析/校验与 deep-merge/redact 工具。
package config

// Version 常量 (Python MemoryConfig.version 默认)。
const Version = "v1.1"

// SensitiveConfigKeys 对齐 server/main.py SENSITIVE_CONFIG_KEYS (redact 用, 大小写不敏感)。
var SensitiveConfigKeys = []string{
	"admin_api_key", "api_key", "authorization", "jwt_secret",
	"password", "password_hash", "secret", "token",
}

// BundledProviderLimit: Go 内置 provider 集 (计划 D1 裁剪)。
var (
	BundledLLMProviders      = []string{"openai", "anthropic", "gemini"}
	BundledEmbedderProviders = []string{"openai", "gemini"}
)

// MemoryConfig 对齐 Python MemoryConfig; 子配置保留原始 dict 形态 (与 Python LlmConfig.config
// 为自由 dict 一致), 由 provider 实现解析自己需要的键 —— 未知键不报错 (pydantic dict 语义)。
type MemoryConfig struct {
	VectorStore        ProviderConfig  `json:"vector_store"`
	LLM                ProviderConfig  `json:"llm"`
	Embedder           ProviderConfig  `json:"embedder"`
	HistoryDBPath      string          `json:"history_db_path"`
	Reranker           *ProviderConfig `json:"reranker"`
	Version            string          `json:"version"`
	CustomInstructions string          `json:"custom_instructions"`
}

// ProviderConfig 是 wire 层的 provider 声明 (对齐 Python LlmConfig: provider + config dict)。
type ProviderConfig struct {
	Provider string         `json:"provider"`
	Config   map[string]any `json:"config"`
}
