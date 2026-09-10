package llm

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

// GeminiConfig: 上游默认 model=gemini-2.0-flash, temp 0.1, max_tokens 2000。
type GeminiConfig struct {
	Model       string
	APIKey      string
	BaseURL     string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
	HTTPClient  *http.Client
}

// DefaultGeminiConfig 对齐上游默认 (gemini-2.0-flash, temp 0.1, max_tokens 2000)。
func DefaultGeminiConfig() GeminiConfig {
	return GeminiConfig{Model: "gemini-2.0-flash", Temperature: 0.1, MaxTokens: 2000}
}

// NewGemini 构造; api_key 回退 GEMINI_API_KEY / GOOGLE_API_KEY env。
func NewGemini(cfg GeminiConfig) (*Gemini, error) {
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

// Gemini 实现 LLM 接口 (generateContent REST)。
type Gemini struct {
	cfg    GeminiConfig
	client *http.Client
}

type gemResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}

// GenerateResponse 实现 LLM 接口; JSON mode 走 responseMimeType。
func (g *Gemini) GenerateResponse(messages []Message, opts GenerateOptions) (string, error) {
	contents := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		text, _ := m.Content.(string)
		contents = append(contents, map[string]any{
			"role":  role,
			"parts": []map[string]any{{"text": text}},
		})
	}
	genCfg := map[string]any{
		"temperature":     g.cfg.Temperature,
		"maxOutputTokens": g.cfg.MaxTokens,
	}
	if opts.ResponseFormat != nil && opts.ResponseFormat.Type == "json_object" {
		genCfg["responseMimeType"] = "application/json"
	}
	body := map[string]any{"contents": contents, "generationConfig": genCfg}
	payload, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("gemini: 序列化失败: %w", err)
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", g.cfg.BaseURL, g.cfg.Model, g.cfg.APIKey)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("gemini: 构造请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := g.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("gemini: 请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("gemini: 读响应失败: %w", err)
	}
	var parsed gemResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("gemini: 响应非 JSON status=%d", resp.StatusCode)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("gemini: 上游错误 status=%s msg=%s", parsed.Error.Status, parsed.Error.Message)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini: 响应无内容")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, nil
}
