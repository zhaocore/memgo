package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
)

// APIKeyPrefix m0sk_ 前缀 (doc-02 §5)。
const APIKeyPrefix = "m0sk_"

// GenerateAPIKey 对齐 generate_api_key: 32 字节 urlsafe → 43 字符 raw。
// 返回 (full_key, prefix[:12], raw)。
func GenerateAPIKey() (full, prefix, raw string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", "", fmt.Errorf("auth: 生成 key 失败: %w", err)
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	full = APIKeyPrefix + raw
	prefix = full[:12]
	return full, prefix, raw, nil
}

// CompareLegacyAdminKey 恒时比较遗留 ADMIN_API_KEY。
func CompareLegacyAdminKey(given, adminKey string) bool {
	if adminKey == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(given), []byte(adminKey)) == 1
}

// KeyPrefixOf 取 key 前 12 字符 (不足取全量, 对齐 Python key[:12])。
func KeyPrefixOf(key string) string {
	if len(key) >= 12 {
		return key[:12]
	}
	return key
}

// NormalizeBearer 剥离 "Bearer " 前缀。
func NormalizeBearer(header string) string {
	return strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
}
