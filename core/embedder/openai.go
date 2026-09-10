package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OpenAIConfig 对齐 Python OpenAIEmbedding 实际使用字段。
type OpenAIConfig struct {
	Model         string
	APIKey        string
	BaseURL       string
	EmbeddingDims int  // 0 → 默认 1536
	DimsExplicit  bool // 用户显式设置 embedding_dims 时 API 才带 dimensions (对齐 _pass_dimensions_to_api)
	Timeout       time.Duration
	HTTPClient    *http.Client
}

// NewOpenAI 构造; model 默认 text-embedding-3-small; dims 默认 1536。
// base_url 回退链对齐 Python: config → OPENAI_API_BASE(弃) → OPENAI_BASE_URL → 官方。
func NewOpenAI(cfg OpenAIConfig) (*OpenAI, error) {
	if cfg.Model == "" {
		cfg.Model = "text-embedding-3-small"
	}
	if cfg.EmbeddingDims == 0 {
		cfg.EmbeddingDims = 1536
		cfg.DimsExplicit = false
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = os.Getenv("OPENAI_API_BASE")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = os.Getenv("OPENAI_BASE_URL")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 600 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &OpenAI{cfg: cfg, client: client, passDims: cfg.DimsExplicit}, nil
}

// OpenAI 实现 Embedder。
type OpenAI struct {
	cfg      OpenAIConfig
	client   *http.Client
	passDims bool
}

type oaEmbResp struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Embed 实现 Embedder: 换行转空格, 单条请求。
func (o *OpenAI) Embed(text string, memoryAction string) ([]float64, error) {
	vecs, err := o.EmbedBatch([]string{text}, memoryAction)
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// EmbedBatch 实现 Embedder: 100 条一批 (对齐 Python MAX_BATCH)。
func (o *OpenAI) EmbedBatch(texts []string, memoryAction string) ([][]float64, error) {
	const maxBatch = 100
	cleaned := make([]string, 0, len(texts))
	for _, t := range texts {
		cleaned = append(cleaned, replaceNewlines(t))
	}
	all := make([][]float64, 0, len(cleaned))
	for start := 0; start < len(cleaned); start += maxBatch {
		end := start + maxBatch
		if end > len(cleaned) {
			end = len(cleaned)
		}
		chunk := cleaned[start:end]
		vecs, err := o.embedChunk(chunk)
		if err != nil {
			return nil, err
		}
		all = append(all, vecs...)
	}
	if len(all) != len(cleaned) {
		return nil, fmt.Errorf("openai: embed_batch 返回 %d 条向量, 期望 %d (model=%s)", len(all), len(cleaned), o.cfg.Model)
	}
	return all, nil
}

// embedChunk 发单批 embeddings 请求 (input 顺序即 index, Python 按 index 重排)。
func (o *OpenAI) embedChunk(input []string) ([][]float64, error) {
	body := map[string]any{
		"input":           input,
		"model":           o.cfg.Model,
		"encoding_format": "float",
	}
	if o.passDims {
		body["dimensions"] = o.cfg.EmbeddingDims
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("openai: 序列化失败: %w", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, o.cfg.BaseURL+"/embeddings", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("openai: 构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai: 请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("openai: 读响应失败: %w", err)
	}
	var parsed oaEmbResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("openai: 响应非 JSON status=%d", resp.StatusCode)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("openai: 上游错误 type=%s msg=%s", parsed.Error.Type, parsed.Error.Message)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai: 上游错误 status=%d", resp.StatusCode)
	}
	out := make([][]float64, len(parsed.Data))
	for _, d := range parsed.Data {
		if d.Index < 0 || d.Index >= len(out) {
			return nil, fmt.Errorf("openai: embedding index 越界: %d", d.Index)
		}
		out[d.Index] = d.Embedding
	}
	return out, nil
}
