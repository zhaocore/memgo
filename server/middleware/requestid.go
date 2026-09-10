// Package middleware: request-id / 请求日志 / CORS / 限流 (对齐 main.py:262-319, rate_limit.py)。
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type ridKey int

const requestIDKey ridKey = 1

// NewRequestID 8 位 hex (对齐 errors.new_request_id)。
func NewRequestID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// RequestID 包装: 生成 rid 注入 context 与响应头。
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := NewRequestID()
		w.Header().Set("X-Request-ID", rid)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, rid)))
	})
}

// RequestIDFrom 从 context 取。
func RequestIDFrom(r *http.Request) string {
	rid, _ := r.Context().Value(requestIDKey).(string)
	return rid
}
