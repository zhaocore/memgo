// OpenAI 兼容打桩 server —— tests/contract 套件的确定性来源。
// 行为:
//   POST /v1/chat/completions  按请求 prompt 特征返回固定内容; 故障经 /_control 注入
//   POST /v1/embeddings        固定向量 (1536 维常量 → 任意文本余弦距离恒 0 → score 恒 1.0)
//   POST /_control             {"mode": normal|auth|rate_limit|bad_request|unavailable}
//   GET  /_control             探活/查当前模式
// provider_timeout 不支持: openai SDK 客户端默认 600s 超时, 黑盒等不起 (Go 侧分类器单测覆盖)。
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
)

const dims = 1536

var (
	mu   sync.Mutex
	mode = "normal"
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func respondOpenAIError(w http.ResponseWriter, status int, typ, code, msg string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{"message": msg, "type": typ, "code": code},
	})
}

func setMode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Mode string `json:"mode"`
	}
	b, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(b, &body); err != nil || body.Mode == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "mode required"})
		return
	}
	mu.Lock()
	mode = body.Mode
	mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"mode": body.Mode})
}

func getMode(w http.ResponseWriter, _ *http.Request) {
	mu.Lock()
	m := mode
	mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"mode": m})
}

type chatMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

func (c chatMessage) text() string {
	var s string
	if err := json.Unmarshal(c.Content, &s); err == nil {
		return s
	}
	// 多分段 content: [{"type":"text","text":"..."}]
	var parts []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(c.Content, &parts); err == nil {
		var b strings.Builder
		for _, p := range parts {
			b.WriteString(p.Text)
		}
		return b.String()
	}
	return ""
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	m := mode
	mu.Unlock()
	switch m {
	case "auth":
		respondOpenAIError(w, http.StatusUnauthorized, "invalid_request_error", "invalid_api_key", "Stub: invalid API key")
		return
	case "rate_limit":
		respondOpenAIError(w, http.StatusTooManyRequests, "rate_limit_error", "rate_limit_exceeded", "Stub: rate limit")
		return
	case "bad_request":
		respondOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "bad_request", "Stub: injected bad request")
		return
	case "unavailable":
		respondOpenAIError(w, http.StatusInternalServerError, "server_error", nil2str(), "Stub: injected server error")
		return
	}

	var req struct {
		Messages []chatMessage `json:"messages"`
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		respondOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "bad_request", "unreadable body")
		return
	}
	if err := json.Unmarshal(b, &req); err != nil {
		respondOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "bad_request", "malformed JSON")
		return
	}

	var joined strings.Builder
	for _, msg := range req.Messages {
		joined.WriteString(msg.text())
		joined.WriteString("\n")
	}
	text := joined.String()

	var content string
	switch {
	case strings.Contains(text, "You are configuring a memory system"):
		// generate-instructions: 服务端按 "INSTRUCTIONS:"/"TEST_MESSAGE:" 切分解析
		content = "INSTRUCTIONS: Prioritize workout routines, dietary preferences and schedule constraints.\nTEST_MESSAGE: I prefer morning runs and a vegetarian diet."
	default:
		// 抽取/决策类: mem0 期望 {"memory":[{"text":...}]} (json_object 模式)
		content = `{"memory":[{"text":"stub fact one"},{"text":"stub fact two"}]}`
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":      "chatcmpl-stub",
		"object":  "chat.completion",
		"created": 0,
		"model":   "stub-model",
		"choices": []map[string]any{{
			"index":         0,
			"message":       map[string]any{"role": "assistant", "content": content},
			"finish_reason": "stop",
		}},
		"usage": map[string]any{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
	})
}

func nil2str() string { return "server_error" }

func handleEmbed(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Input json.RawMessage `json:"input"`
	}
	b, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(b, &req); err != nil {
		respondOpenAIError(w, http.StatusBadRequest, "invalid_request_error", "bad_request", "malformed JSON")
		return
	}
	// input 兼容 string / []string / [float] (token 数组不区分, 一律按条数返回)
	count := 1
	var texts []string
	if json.Unmarshal(req.Input, &texts) == nil {
		count = len(texts)
	} else {
		var one string
		if json.Unmarshal(req.Input, &one) == nil {
			count = 1
		}
	}
	vec := make([]float64, dims)
	for i := range vec {
		vec[i] = 0.1
	}
	data := make([]map[string]any, count)
	for i := range data {
		data[i] = map[string]any{"object": "embedding", "index": i, "embedding": vec}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data":   data,
		"model":  "stub-embedding",
		"usage":  map[string]any{"prompt_tokens": 1, "total_tokens": 1},
	})
}

func main() {
	addr := "0.0.0.0:" + envOr("STUB_PORT", "8090")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /_control", setMode)
	mux.HandleFunc("GET /_control", getMode)
	mux.HandleFunc("POST /v1/chat/completions", handleChat)
	mux.HandleFunc("POST /v1/embeddings", handleEmbed)
	log.Printf("contract stub listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
