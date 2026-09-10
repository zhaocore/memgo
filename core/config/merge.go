package config

import (
	"strings"
)

// DeepMerge 对齐 Python deepmerge: dict 递归合并, 标量与数组整体覆盖。
// 返回新 map, 不改入参。
func DeepMerge(base, updates map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(updates))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range updates {
		if existing, ok := out[k].(map[string]any); ok {
			if incoming, ok := v.(map[string]any); ok {
				out[k] = DeepMerge(existing, incoming)
				continue
			}
		}
		out[k] = v
	}
	return out
}

// Redact 对齐 server/main.py _redact_config: 敏感键(大小写不敏感)值替换 "[redacted]"
// (值非空时), dict/list 递归, list 继承父键名。
func Redact(v any, key string) any {
	switch x := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = Redact(val, k)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = Redact(item, key)
		}
		return out
	}
	if key != "" && isSensitiveKey(key) {
		// Python: "[redacted]" if value else value —— 空值保持原样
		switch x := v.(type) {
		case string:
			if x != "" {
				return "[redacted]"
			}
		case nil:
			return nil
		default:
			if x != false && x != nil {
				return "[redacted]"
			}
		}
	}
	return v
}

func isSensitiveKey(key string) bool {
	k := strings.ToLower(key)
	for _, s := range SensitiveConfigKeys {
		if k == s {
			return true
		}
	}
	return false
}
