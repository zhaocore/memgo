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

// OpenAIConfig 对齐 Python OpenAIConfig 实际使用的字段。
type OpenAIConfig struct {
	Model       string
	APIKey      string
	BaseURL     string // openai_base_url 配置或 OPENAI_BASE_URL env
	Temperature float64
	MaxTokens   int
	TopP        float64
	Timeout     time.Duration
	HTTPClient  *http.Client
}

// DefaultOpenAIConfig 对齐 Python OpenAIConfig 默认 (model gpt-5-mini, temp 0.1, max_tokens 2000, top_p 0.1)。
// 但 server DEFAULT_CONFIG 显式传 temperature 0.2 —— 默认值与此处独立。
func DefaultOpenAIConfig() OpenAIConfig {
	return OpenAIConfig{
		Model:       "gpt-5-mini",
		Temperature: 0.1,
		MaxTokens:   2000,
		TopP:        0.1,
		Timeout:     600 * time.Second,
	}
}

// OpenAI 基于小型类型化 REST 客户端 (计划 D6: 不引重框架)。
type OpenAI struct {
	cfg    OpenAIConfig
	client *http.Client
}

// NewOpenAI 构造; api_key/base_url 回退 env (OPENAI_API_KEY / OPENAI_BASE_URL)。
func NewOpenAI(cfg OpenAIConfig) (*OpenAI, error) {
	if cfg.APIKey == "" {
		cfg.APIKey = os.Getenv("OPENAI_API_KEY")
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
	return &OpenAI{cfg: cfg, client: client}, nil
}

type oaChatResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *oaErr `json:"error"`
}

type oaErr struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"`
}

// GenerateResponse 实现 LLM 接口 (chat completions; 429/5xx/网络错误重试 2 次后上抛)。
func (o *OpenAI) GenerateResponse(messages []Message, opts GenerateOptions) (string, error) {
	body := map[string]any{
		"model":       o.cfg.Model,
		"messages":    messages,
		"temperature": o.cfg.Temperature,
		"max_tokens":  o.cfg.MaxTokens,
		"top_p":       o.cfg.TopP,
	}
	for k, v := range opts.Extra {
		body[k] = v
	}
	if opts.ResponseFormat != nil {
		body["response_format"] = opts.ResponseFormat
	}
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}
		content, retryable, err := o.call(body)
		if err == nil {
			return content, nil
		}
		lastErr = err
		if !retryable {
			return "", err
		}
	}
	return "", lastErr
}

// call 发单次请求; retryable 标记 429/5xx/网络错误。
func (o *OpenAI) call(body map[string]any) (string, bool, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return "", false, fmt.Errorf("openai: 序列化请求失败: %w", err)
	}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, o.cfg.BaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", false, fmt.Errorf("openai: 构造请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := o.client.Do(req)
	if err != nil {
		return "", true, fmt.Errorf("openai: 请求失败: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", true, fmt.Errorf("openai: 读响应失败: %w", err)
	}
	var parsed oaChatResp
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", resp.StatusCode >= 500 || resp.StatusCode == 429, fmt.Errorf("openai: 响应非 JSON (status=%d)", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK || parsed.Error != nil {
		msg := fmt.Sprintf("openai: 上游错误 status=%d", resp.StatusCode)
		if parsed.Error != nil {
			msg = fmt.Sprintf("openai: 上游错误 status=%d type=%s msg=%s", resp.StatusCode, parsed.Error.Type, parsed.Error.Message)
		}
		retryable := resp.StatusCode == 429 || resp.StatusCode >= 500
		return "", retryable, NewStatusError(resp.StatusCode, "%s", msg)
	}
	if len(parsed.Choices) == 0 {
		return "", false, fmt.Errorf("openai: 响应无 choices")
	}
	return parsed.Choices[0].Message.Content, false, nil
}
