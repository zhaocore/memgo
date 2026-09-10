// Package llm: LLM 接口与内置 provider (openai/anthropic/gemini)。
// 对齐上游 mem0/llms/base.py 接口面 (chat + structured JSON 两个能力, 计划 D6)。
package llm

// Message 与 OpenAI chat 消息形状一致 (role/content)。
type Message struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string 或分段数组
}

// ResponseFormat: nil = 文本; JSON = {"type":"json_object"}。
type ResponseFormat struct {
	Type string `json:"type"`
}

// JSONFormat 是 mem0 add 流水线使用的 response_format。
func JSONFormat() *ResponseFormat {
	return &ResponseFormat{Type: "json_object"}
}

// GenerateOptions 汇总 generate_response 的可选参数。
type GenerateOptions struct {
	ResponseFormat *ResponseFormat
	// Extra 直接并入请求体 (provider 特有字段)。
	Extra map[string]any
}

// LLM 对齐 Python LLMBase.generate_response(messages, response_format=None, **kwargs)。
type LLM interface {
	GenerateResponse(messages []Message, opts GenerateOptions) (string, error)
}
