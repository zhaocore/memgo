package store

import (
	"time"
)

// User 对齐 models.User (JSON 序列化仅 /auth/me 使用, 时间 Z 格式)。
type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
	LastLoginAt  *time.Time
}

// APIKey 对齐 models.APIKey。
type APIKey struct {
	ID         string
	KeyPrefix  string
	KeyHash    string
	Label      string
	CreatedBy  string
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

// RequestLog 对齐 models.RequestLog。
type RequestLog struct {
	ID         string
	Method     string
	Path       string
	StatusCode int
	LatencyMS  float64
	AuthType   string
	CreatedAt  time.Time
}

// ZFormat 复刻 pydantic 对 UTC datetime 的序列化: 微秒 6 位 + "Z"。
func ZFormat(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000000") + "Z"
}
