package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter 按 IP 固定窗口限流 (slowapi 5/10/20 per minute 等价; 窗口语义差异由契约 429 体锁定兜底)。
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	limit   int
	window  time.Duration
}

type rateBucket struct {
	count     int
	windowEnd time.Time
}

// NewRateLimiter 构造 (limit per minute)。
func NewRateLimiter(limit int) *RateLimiter {
	return &RateLimiter{buckets: map[string]*rateBucket{}, limit: limit, window: time.Minute}
}

// Allow 判定并计数。
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok || now.After(b.windowEnd) {
		rl.buckets[key] = &rateBucket{count: 1, windowEnd: now.Add(rl.window)}
		return true
	}
	if b.count >= rl.limit {
		return false
	}
	b.count++
	return true
}

// RemoteIP 对齐 get_remote_address。
func RemoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// LimitMiddleware 挂载限流; 超限返回 slowapi 默认体 {"error": "Rate limit exceeded: N per 1 minute"}。
func LimitMiddleware(rl *RateLimiter, limitLabel string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow(RemoteIP(r)) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "Rate limit exceeded: " + limitLabel,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}
