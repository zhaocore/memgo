package errpkg

import (
	"errors"
	"strings"

	"github.com/zhao-core/memgo/core/llm"
)

// Classify 对齐 _classify (沿错误链向上归因, 命中即返)。
// 载体优先级: llm.StatusError (上游 HTTP 状态) → 错误串前缀 (pgvector: store)。
func Classify(err error) string {
	current := err
	for depth := 0; current != nil && depth < 8; depth++ {
		if se, ok := current.(*llm.StatusError); ok {
			switch {
			case se.Status == 401 || se.Status == 403:
				return CodeProviderAuth
			case se.Status == 429:
				return CodeProviderRateLimit
			case se.Status == 0:
				// 无状态载体 (解析失败/空响应) — 视为上游不可用
				return CodeProviderUnavail
			case se.Status >= 500:
				return CodeProviderUnavail
			case se.Status == 400 || se.Status == 422:
				return CodeProviderBadReq
			default:
				return CodeUnknown
			}
		}
		msg := current.Error()
		switch {
		case strings.HasPrefix(msg, "pgvector:"):
			return CodeVectorUnavail
		case strings.HasPrefix(msg, "appdb:"):
			return CodeDatastoreUnavail
		}
		current = errors.Unwrap(current)
	}
	return CodeUnknown
}
