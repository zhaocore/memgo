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

// GeminiConfig: 上游默认 model=models/gemini-embedding-001, dims 默认 768。
type GeminiConfig struct {
	Model         string
	APIKey        string
	BaseURL       string
	EmbeddingDims int
	Timeout       time.Duration
	HTTPClient    *http.Client
}

// NewGemini 构造; api_key 回退 GEMINI_API_KEY / GOOGLE_API_KEY。
func NewGemini(cfg GeminiConfig) (*Gemini, error) {
	if cfg.Model == "" {
		cfg.Model = "models/gemini-embedding-001"
	}
	if cfg.EmbeddingDims == 0 {
		cfg.EmbeddingDims = 768
	}
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("GEMINI_API_KEY")
		if cfg.APIKey == "" {
			cfg.APIKey = os.Getenv("GOOGLE_API_KEY")
		}
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://generativelanguage.googleapis.com"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 600 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &Gemini{cfg: cfg, client: client}, nil
}

// Gemini 实现 Embedder (batchEmbedContents REST)。
type Gemini struct {
	cfg    GeminiConfig
	client *http.Client
}

type gemEmbResp struct {
	Embeddings []struct {
		Values []float64 `json:"values"`
	} `json:"embeddings"`
	Error *struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// Embed 实现 Embedder (单条走批量通道)。
func (g *Gemini) Embed(text string, memoryAction string) ([]float64, error) {
	vecs, err := g.EmbedBatch([]string{text}, memoryAction)
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

// EmbedBatch 实现 Embedder。
func (g *Gemini) EmbedBatch(texts []string, memoryAction string) ([][]float64, error) {
	if len(texts) == 0 {
		return [][]float64{}, nil
	}
	cleaned := make([]string, 0, len(texts))
	for _, t := range texts {
		cleaned = append(cleaned, replaceNewlines(t))
	}
	requests := make([]map[string]any, 0, len(cleaned))
	for _, t := range cleaned {
		requests = append(requests, map[string]any{
			"model":    g.cfg.Model,
			"content":  map[string]any{"parts": []map[string]any{{"text": t}}},
			"taskType": taskType(memoryAction),
		})
	}
	body := map[string]any{
		"requests":             requests,
		"outputDimensionality": g.cfg.EmbeddingDims,
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("gemini: 序列化失败: %w", err)
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:batchEmbedContents?key=%s", g.cfg.BaseURL, g.cfg.Model, g.cfg.APIKey)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("gemini: 构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini: 请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("gemini: 读响应失败: %w", err)
	}
	var parsed gemEmbResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("gemini: 响应非 JSON status=%d", resp.StatusCode)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("gemini: 上游错误 status=%s msg=%s", parsed.Error.Status, parsed.Error.Message)
	}
	if len(parsed.Embeddings) != len(cleaned) {
		return nil, fmt.Errorf("gemini: 返回 %d 条向量, 期望 %d", len(parsed.Embeddings), len(cleaned))
	}
	out := make([][]float64, 0, len(parsed.Embeddings))
	for _, e := range parsed.Embeddings {
		out = append(out, e.Values)
	}
	return out, nil
}

// taskType 对齐 Python EmbedContentConfig (embed 未显式区分, 统一默认)。
func taskType(memoryAction string) string {
	return "RETRIEVAL_DOCUMENT"
}
