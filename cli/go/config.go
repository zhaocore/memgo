// Package cli: memgo CLI (第三实现, 对齐 cli/python 命令面)。
// 配置文件路径与 schema 对齐 ~/.mem0/config.json (与 python/node CLI 共享互认)。
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config 对齐 Mem0Config dataclass (schema 与 python CLI 一致)。
type Config struct {
	Version   int            `json:"version"`
	Defaults  DefaultsConfig `json:"defaults"`
	Platform  PlatformConfig `json:"platform"`
	Telemetry TelemetryCfg   `json:"telemetry"`
	AgentRush AgentRushCfg   `json:"agent_rush"`
}

type DefaultsConfig struct {
	UserID  string `json:"user_id"`
	AgentID string `json:"agent_id"`
	AppID   string `json:"app_id"`
	RunID   string `json:"run_id"`
}

type PlatformConfig struct {
	APIKey        string `json:"api_key"`
	BaseURL       string `json:"base_url"`
	UserEmail     string `json:"user_email"`
	AgentMode     bool   `json:"agent_mode"`
	CreatedVia    string `json:"created_via"`
	AgentCaller   string `json:"agent_caller"`
	ClaimedAt     string `json:"claimed_at"`
	DefaultUserID string `json:"default_user_id"`
}

type TelemetryCfg struct {
	AnonymousID string `json:"anonymous_id"`
}

type AgentRushCfg struct {
	AcknowledgedAt string `json:"acknowledged_at"`
}

// DefaultBaseURL 对齐 python DEFAULT_BASE_URL。
const DefaultBaseURL = "https://api.mem0.ai"

// ConfigFile 路径 (python 同款)。
func ConfigFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.json"
	}
	return filepath.Join(home, ".mem0", "config.json")
}

// LoadConfig 读配置 + env 覆盖 (优先级: flag > env > file > 默认)。
func LoadConfig() *Config {
	cfg := &Config{Version: 1, Platform: PlatformConfig{BaseURL: DefaultBaseURL}}
	raw, err := os.ReadFile(ConfigFile())
	if err == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, cfg)
		if cfg.Platform.BaseURL == "" {
			cfg.Platform.BaseURL = DefaultBaseURL
		}
	}
	if v := os.Getenv("MEM0_API_KEY"); v != "" {
		cfg.Platform.APIKey = v
	}
	if v := os.Getenv("MEM0_BASE_URL"); v != "" {
		cfg.Platform.BaseURL = v
	}
	if v := os.Getenv("MEM0_USER_ID"); v != "" {
		cfg.Defaults.UserID = v
	}
	if v := os.Getenv("MEM0_AGENT_ID"); v != "" {
		cfg.Defaults.AgentID = v
	}
	if v := os.Getenv("MEM0_APP_ID"); v != "" {
		cfg.Defaults.AppID = v
	}
	if v := os.Getenv("MEM0_RUN_ID"); v != "" {
		cfg.Defaults.RunID = v
	}
	return cfg
}

// SaveConfig 写盘 (0600)。
func SaveConfig(cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(ConfigFile()), 0o700); err != nil {
		return fmt.Errorf("cli: 建配置目录失败: %w", err)
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(ConfigFile(), raw, 0o600); err != nil {
		return fmt.Errorf("cli: 写配置失败: %w", err)
	}
	return nil
}

// RedactKey 对齐 redact_key。
func RedactKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if len(key) <= 8 {
		return key[:2] + "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
