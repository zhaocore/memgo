package api

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// telemetryState 遥测状态文件 (对齐 telemetry.py: install_id + 一次性标记)。
// 网络发送省略 (PostHog 集成属可选); 事件语义与状态文件路径/格式对齐。
type telemetryState struct {
	InstallID               string `json:"install_id"`
	AdminRegisteredSent     bool   `json:"admin_registered_sent"`
	OnboardingCompletedSent bool   `json:"onboarding_completed_sent"`
}

var (
	telMu     sync.Mutex
	telLoaded bool
	telState  telemetryState
)

func telemetryEnabled() bool { return os.Getenv("MEMGO_TELEMETRY") != "false" }

func telemetryPath() string {
	if p := os.Getenv("MEMGO_TELEMETRY_STATE_PATH"); p != "" {
		return p
	}
	if h := os.Getenv("HISTORY_DB_PATH"); h != "" {
		return filepath.Join(filepath.Dir(h), "telemetry.json")
	}
	return "/app/history/telemetry.json"
}

func loadTelemetry() *telemetryState {
	telMu.Lock()
	defer telMu.Unlock()
	if telLoaded {
		return &telState
	}
	telLoaded = true
	raw, err := os.ReadFile(telemetryPath())
	if err == nil && len(raw) > 0 {
		_ = json.Unmarshal(raw, &telState)
	}
	if telState.InstallID == "" {
		telState.InstallID = newRequestUUID()
	}
	return &telState
}

func saveTelemetry() {
	telMu.Lock()
	defer telMu.Unlock()
	raw, err := json.MarshalIndent(telState, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(telemetryPath()), 0o755)
	_ = os.WriteFile(telemetryPath(), raw, 0o644)
}

// telemetryAdminRegistered 每安装至多一次。
func (s *Server) telemetryAdminRegistered(email string) {
	if !telemetryEnabled() {
		return
	}
	st := loadTelemetry()
	if st.AdminRegisteredSent {
		return
	}
	st.AdminRegisteredSent = true
	saveTelemetry()
	log.Printf("telemetry: admin_registered (domain=%s)", emailDomain(email))
}

// telemetryOnboarding 每安装至多一次。
func (s *Server) telemetryOnboarding(email, useCase string) {
	if !telemetryEnabled() {
		return
	}
	st := loadTelemetry()
	if st.OnboardingCompletedSent {
		return
	}
	st.OnboardingCompletedSent = true
	saveTelemetry()
	log.Printf("telemetry: onboarding_completed (domain=%s)", emailDomain(email))
}

func emailDomain(email string) string {
	for i := len(email) - 1; i >= 0; i-- {
		if email[i] == '@' {
			return email[i+1:]
		}
	}
	return email
}
