package cli

import (
	"net/url"
	"strings"
)

// hostOf 取 URL host (无 scheme 视为平台)。
func hostOf(raw string) string {
	if raw == "" {
		return ""
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return strings.ToLower(u.Hostname())
	}
	return strings.ToLower(strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://"))
}
