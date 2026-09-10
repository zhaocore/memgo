package cli

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// telemetry 常量对齐 cli/python telemetry.py。
const (
	posthogAPIKey = "phc_hgJkUVJFYtmaJqrvf6CYN67TIQ8yhXAkWzUn9AMU4yX"
	posthogHost   = "https://us.i.posthog.com/i/v0/e/"
)

// telemetryEnabled MEM0_TELEMETRY=false 关闭。
func telemetryEnabled() bool {
	v := os.Getenv("MEM0_TELEMETRY")
	return v == "" || (v != "false" && v != "0" && v != "no")
}

// distinctID 对齐 _get_distinct_id: user_email > md5(api_key) > anonymous_id。
func distinctID(cfg *Config) string {
	if cfg.Platform.UserEmail != "" {
		return cfg.Platform.UserEmail
	}
	if cfg.Platform.APIKey != "" {
		sum := md5.Sum([]byte(cfg.Platform.APIKey))
		return hex.EncodeToString(sum[:])
	}
	if cfg.Telemetry.AnonymousID != "" {
		return cfg.Telemetry.AnonymousID
	}
	id := fmt.Sprintf("cli-anon-%x", time.Now().UTC().UnixNano())
	cfg.Telemetry.AnonymousID = id
	_ = SaveConfig(cfg)
	return id
}

// fireTelemetry 异步发事件 (fire-and-forget, 2s 超时, 永不阻塞/失败)。
func fireTelemetry(cfg *Config, event string, props map[string]any) {
	if !telemetryEnabled() {
		return
	}
	go func() {
		payload := map[string]any{
			"api_key":     posthogAPIKey,
			"event":       event,
			"distinct_id": distinctID(cfg),
			"properties":  props,
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return
		}
		client := &http.Client{Timeout: 2 * time.Second}
		req, err := http.NewRequest(http.MethodPost, posthogHost, bytes.NewReader(raw))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		_, _ = client.Do(req)
	}()
}
