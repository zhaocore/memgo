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
	VectorStore        ProviderConfig    `json:"vector_store"`
	LLM                ProviderConfig    `json:"llm"`
	Embedder           ProviderConfig    `json:"embedder"`
	HistoryDBPath      string            `json:"history_db_path"`
	Reranker           *ProviderConfig   `json:"reranker"`
	Version            string            `json:"version"`
	CustomInstructions string            `json:"custom_instructions"`
	CustomCategories   []Category        `json:"custom_categories"`
	GraphMemory        GraphMemoryConfig `json:"graph_memory"`
}

// GraphMemoryConfig 图记忆多跳检索参数。对齐 v3 native Graph Memory。
type GraphMemoryConfig struct {
	// EnableMultiHop 是否启用多跳图遍历扩展 (默认 true)。false 时仅保留现有 entity boost。
	EnableMultiHop bool `json:"enable_multi_hop"`
	// MaxHops 最大跳数 (默认 2)。0=仅直达, 1=直达+1跳邻居, 2=直达+2跳邻居。
	MaxHops int `json:"max_hops"`
	// DecayFactor 逐跳衰减因子 (默认 0.5)。跳数 n 的 boost = decayFactor^n。
	DecayFactor float64 `json:"decay_factor"`
}

// DefaultGraphMemoryConfig 返回默认图记忆配置。
func DefaultGraphMemoryConfig() GraphMemoryConfig {
	return GraphMemoryConfig{
		EnableMultiHop: true,
		MaxHops:        2,
		DecayFactor:    0.5,
	}
}

// Category 自定义分类条目 (wire 形态 [{"分类名": "描述"}])。
type Category struct {
	Name        string
	Description string
}

// defaultCategoryDescriptions 内置默认 15 分类目录。
// 注意: MemGo OSS 面默认不启用分类打标 (打标需 LLM 调用, 且会改 payload/响应形状,
// 破坏与 Python 契约基线的对拍) —— 目录仅在 custom_categories 配置/传参后生效。
var defaultCategoryDescriptions = []Category{
	{"personal_details", "Basic personal information about the user"},
	{"family", "Family members and relationships"},
	{"professional_details", "Work, career, and professional background"},
	{"sports", "Sports activities and interests"},
	{"travel", "Trips, destinations, and travel preferences"},
	{"food", "Food, drinks, and dining preferences"},
	{"music", "Music tastes and related activities"},
	{"health", "Health conditions, fitness, and wellness"},
	{"technology", "Devices, software, and tech interests"},
	{"hobbies", "Hobbies and leisure activities"},
	{"fashion", "Clothing, style, and fashion preferences"},
	{"entertainment", "Movies, TV, games, and other entertainment"},
	{"milestones", "Important life events and achievements"},
	{"user_preferences", "General likes, dislikes, and preferences"},
	{"misc", "Anything that does not fit another category"},
}

// DefaultCategories 返回内置默认目录副本 (避免调用方改动共享切片)。
func DefaultCategories() []Category {
	out := make([]Category, len(defaultCategoryDescriptions))
	copy(out, defaultCategoryDescriptions)
	return out
}

// ProviderConfig 是 wire 层的 provider 声明 (对齐 Python LlmConfig: provider + config dict)。
type ProviderConfig struct {
	Provider string         `json:"provider"`
	Config   map[string]any `json:"config"`
}
