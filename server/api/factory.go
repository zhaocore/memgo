package api

import (
	"fmt"

	"github.com/zhao-core/memgo/core/config"
	"github.com/zhao-core/memgo/core/embedder"
	"github.com/zhao-core/memgo/core/llm"
	"github.com/zhao-core/memgo/core/vectorstore"
)

// buildVectorStore: 内置集仅 pgvector (计划 D1)。
func buildVectorStore(pc config.ProviderConfig) (vectorstore.VectorStore, error) {
	switch pc.Provider {
	case "pgvector":
		return vectorstore.NewPGVector(vectorstore.PGVectorConfig{
			DBName:             parseStr(pc.Config, "dbname", "postgres"),
			CollectionName:     parseStr(pc.Config, "collection_name", "mem0"),
			EmbeddingModelDims: parseInt(pc.Config, "embedding_model_dims", 1536),
			User:               parseStr(pc.Config, "user", "postgres"),
			Password:           parseStr(pc.Config, "password", ""),
			Host:               parseStr(pc.Config, "host", "localhost"),
			Port:               parseInt(pc.Config, "port", 5432),
			DiskANN:            parseBool(pc.Config, "diskann", false),
			HNSW:               parseBool(pc.Config, "hnsw", true),
		})
	default:
		return nil, fmt.Errorf("向量库 provider '%s' 不在 Go 内置集 (pgvector)", pc.Provider)
	}
}

// buildLLM: 内置集 openai/anthropic/gemini。
func buildLLM(pc config.ProviderConfig) (llm.LLM, error) {
	apiKey := parseStr(pc.Config, "api_key", "")
	model := parseStr(pc.Config, "model", "")
	switch pc.Provider {
	case "openai":
		cfg := llm.DefaultOpenAIConfig()
		cfg.APIKey = apiKey
		cfg.Model = orDefault(model, cfg.Model)
		cfg.BaseURL = parseStr(pc.Config, "openai_base_url", "")
		if t, ok := pc.Config["temperature"].(float64); ok {
			cfg.Temperature = t
		}
		return llm.NewOpenAI(cfg)
	case "anthropic":
		cfg := llm.DefaultAnthropicConfig()
		cfg.APIKey = apiKey
		cfg.Model = orDefault(model, cfg.Model)
		cfg.BaseURL = parseStr(pc.Config, "anthropic_base_url", "")
		if t, ok := pc.Config["temperature"].(float64); ok {
			cfg.Temperature = t
		}
		return llm.NewAnthropic(cfg)
	case "gemini":
		cfg := llm.DefaultGeminiConfig()
		cfg.APIKey = apiKey
		cfg.Model = orDefault(model, cfg.Model)
		if t, ok := pc.Config["temperature"].(float64); ok {
			cfg.Temperature = t
		}
		return llm.NewGemini(cfg)
	default:
		return nil, fmt.Errorf("LLM provider '%s' 不在 Go 内置集 (openai/anthropic/gemini)", pc.Provider)
	}
}

// buildEmbedder: 内置集 openai/gemini。
func buildEmbedder(pc config.ProviderConfig) (embedder.Embedder, error) {
	apiKey := parseStr(pc.Config, "api_key", "")
	model := parseStr(pc.Config, "model", "")
	switch pc.Provider {
	case "openai":
		cfg := embedder.OpenAIConfig{APIKey: apiKey, BaseURL: parseStr(pc.Config, "openai_base_url", "")}
		if model != "" {
			cfg.Model = model
		}
		if d, ok := pc.Config["embedding_dims"].(float64); ok {
			cfg.EmbeddingDims = int(d)
			cfg.DimsExplicit = true
		}
		return embedder.NewOpenAI(cfg)
	case "gemini":
		cfg := embedder.GeminiConfig{APIKey: apiKey}
		if model != "" {
			cfg.Model = model
		}
		if d, ok := pc.Config["embedding_dims"].(float64); ok {
			cfg.EmbeddingDims = int(d)
		}
		return embedder.NewGemini(cfg)
	default:
		return nil, fmt.Errorf("Embedder provider '%s' 不在 Go 内置集 (openai/gemini)", pc.Provider)
	}
}
