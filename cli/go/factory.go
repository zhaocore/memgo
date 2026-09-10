package cli

// GetBackend 工厂: base_url 非平台域名 → OSS backend (doc-01 §4.3 检测规则)。
func GetBackend(cfg *Config) Backend {
	if isPlatformURL(cfg.Platform.BaseURL) {
		return NewPlatformBackend(cfg)
	}
	return NewOSSBackend(cfg)
}

// isPlatformURL 判定平台域名。
func isPlatformURL(baseURL string) bool {
	host := hostOf(baseURL)
	return host == "api.mem0.ai" || host == ""
}
