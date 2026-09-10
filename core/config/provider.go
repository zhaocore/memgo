package config

import (
	"fmt"
)

// parseProvider 抽取 raw[key] 为 ProviderConfig (缺省 provider=default)。
func parseProvider(raw map[string]any, key string, def string) (ProviderConfig, error) {
	pc := ProviderConfig{Provider: def}
	v, ok := raw[key]
	if !ok || v == nil {
		return pc, nil
	}
	m, ok := v.(map[string]any)
	if !ok {
		return pc, fmt.Errorf("config %s: 必须是对象", key)
	}
	if p, ok := m["provider"].(string); ok && p != "" {
		pc.Provider = p
	}
	if c, ok := m["config"].(map[string]any); ok && c != nil {
		pc.Config = c
	}
	return pc, nil
}
