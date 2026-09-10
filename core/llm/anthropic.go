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

// AnthropicConfig: 上游默认 model=claude-sonnet-4-6, max_tokens 2000, temp 0.1。
// temperature 与 top_p 互斥 (Anthropic 拒绝同时传), 默认只传 temperature。
type AnthropicConfig struct {
	Model       string
	APIKey      string
	BaseURL     string
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
	HTTPClient  *http.Client
}

// DefaultAnthropicConfig 对齐上游默认 (claude-sonnet-4-6, temp 0.1, max_tokens 2000)。
func DefaultAnthropicConfig() AnthropicConfig {
	return AnthropicConfig{Model: "claude-sonnet-4-6", Temperature: 0.1, MaxTokens: 2000}
}

// NewAnthropic 构造; api_key 回退 ANTHROPIC_API_KEY env, base_url 回退官方端点。
func NewAnthropic(cfg AnthropicConfig) (*Anthropic, error) {
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.anthropic.com"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 600 * time.Second
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	return &Anthropic{cfg: cfg, client: client}, nil
}

// Anthropic 实现 LLM 接口 (Messages API)。
type Anthropic struct {
	cfg    AnthropicConfig
	client *http.Client
}

type anMsgResp struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// GenerateResponse 实现 LLM 接口; system 消息拆到顶层, JSON mode 以提示词补丁实现
// (Messages API 无 response_format)。
func (a *Anthropic) GenerateResponse(messages []Message, opts GenerateOptions) (string, error) {
	system := ""
	rest := make([]Message, 0, len(messages))
	for _, m := range messages {
		if m.Role == "system" {
			system = fmt.Sprintf("%v", m.Content)
			continue
		}
		rest = append(rest, m)
	}
	body := map[string]any{
		"model":       a.cfg.Model,
		"max_tokens":  a.cfg.MaxTokens,
		"temperature": a.cfg.Temperature,
		"messages":    rest,
	}
	if system != "" {
		body["system"] = system
	}
	if opts.ResponseFormat != nil && opts.ResponseFormat.Type == "json_object" {
		sys, _ := body["system"].(string)
		body["system"] = sys + "\n\nRespond ONLY with a valid JSON object."
	}
	raw, err := a.post(body)
	if err != nil {
		return "", err
	}
	var parsed anMsgResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("anthropic: 响应非 JSON")
	}
	if parsed.Error != nil {
		return "", NewStatusError(0, "anthropic: 上游错误 type=%s msg=%s", parsed.Error.Type, parsed.Error.Message)
	}
	for _, c := range parsed.Content {
		if c.Type == "text" {
			return c.Text, nil
		}
	}
	return "", fmt.Errorf("anthropic: 响应无 text 内容")
}

// post 发单次 Messages API 请求 (重试语义同 OpenAI: 由调用方 GenerateResponse 内层简化为直连)。
func (a *Anthropic) post(body map[string]any) ([]byte, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: 序列化失败: %w", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, a.cfg.BaseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("anthropic: 构造请求失败: %w", err)
	}
	req.Header.Set("x-api-key", a.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic: 请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("anthropic: 读响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return raw, NewStatusError(resp.StatusCode, "anthropic: 上游错误 status=%d body=%s", resp.StatusCode, string(raw))
	}
	return raw, nil
}
