package memory

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/zhao-core/memgo/core/jsonx"
)

// memHash 对齐 hashlib.md5(data.encode()).hexdigest()。
func memHash(data string) string {
	sum := md5.Sum([]byte(data))
	return hex.EncodeToString(sum[:])
}

// identityKeys 对齐 _IDENTITY_KEYS。
var identityKeys = map[string]bool{"user_id": true, "agent_id": true, "run_id": true, "actor_id": true}

// stripIdentityKeys 对齐 _strip_identity_keys: 创建路径 existing 传空, 全部丢弃并告警。
func stripIdentityKeys(metadata map[string]any, existing map[string]any, warn func(string)) map[string]any {
	clean := map[string]any{}
	for k, v := range metadata {
		if !identityKeys[k] {
			clean[k] = v
			continue
		}
		if fmt.Sprintf("%v", v) != fmt.Sprintf("%v", existing[k]) && warn != nil {
			warn(fmt.Sprintf("ignoring metadata[%q] - identity fields cannot be set through metadata", k))
		}
	}
	return clean
}

// validateAndTrimEntityID 对齐 _validate_and_trim_entity_id。
func validateAndTrimEntityID(value any, name string) (string, error) {
	if value == nil {
		return "", nil
	}
	s := fmt.Sprintf("%v", value)
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "", fmt.Errorf("Invalid %s: cannot be empty or whitespace-only. Provide a valid identifier.", name)
	}
	for _, r := range trimmed {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\v' || r == '\f' ||
			(r >= 0x85 && r <= 0x95 && (r == 0x85 || r == 0xa0)) || r == 0x1680 ||
			(r >= 0x2000 && r <= 0x200a) || r == 0x2028 || r == 0x2029 || r == 0x202f || r == 0x205f || r == 0x3000 {
			return "", fmt.Errorf("Invalid %s: cannot contain whitespace. Provide a valid identifier without spaces.", name)
		}
	}
	return trimmed, nil
}

// validateSearchParams 对齐 _validate_search_params。
func validateSearchParams(threshold *float64, topK *int) error {
	if threshold != nil && (*threshold < 0 || *threshold > 1) {
		return fmt.Errorf("Invalid threshold: %s. Must be between 0 and 1 (inclusive).", jsonx.Repr(*threshold))
	}
	if topK != nil && *topK < 0 {
		return fmt.Errorf("Invalid top_k: %d. Must be a non-negative integer.", *topK)
	}
	return nil
}

// validateAndTrimSearchQuery 对齐 _validate_and_trim_search_query。
func validateAndTrimSearchQuery(query string) (string, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return "", fmt.Errorf("Invalid query: cannot be empty or whitespace-only.")
	}
	return trimmed, nil
}

// normalizeExpirationDate 对齐 _normalize_expiration_date (仅字符串输入; 日期对象不经 HTTP 出现)。
func normalizeExpirationDate(value any) (string, error) {
	if value == nil {
		return "", nil
	}
	s, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("expiration_date must be a date string in YYYY-MM-DD format.")
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return "", fmt.Errorf("expiration_date must be a valid date in YYYY-MM-DD format.")
	}
	return t.Format("2006-01-02"), nil
}

// payloadIsExpired 对齐 _payload_is_expired (UTC 日期比较)。
func payloadIsExpired(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	exp, ok := payload["expiration_date"].(string)
	if !ok || exp == "" {
		return false
	}
	t, err := time.Parse("2006-01-02", exp)
	if err != nil {
		return false
	}
	return t.Before(time.Now().UTC().Truncate(24 * time.Hour))
}

// normalizeISOTimestampToUTC 对齐 _normalize_iso_timestamp_to_utc。
func normalizeISOTimestampToUTC(ts string) string {
	if ts == "" {
		return ts
	}
	for _, layout := range []string{
		"2006-01-02T15:04:05.999999Z07:00", "2006-01-02T15:04:05.999999",
	} {
		if t, err := time.Parse(layout, ts); err == nil {
			if t.Location() == time.UTC && layout == "2006-01-02T15:04:05.999999Z07:00" {
				off := t.Format("-07:00")
				if off == "+00:00" {
					return ts // naive-like: 带 +00:00 的视为已 UTC, 原样返回
				}
			}
			if t.Location() != time.UTC && layout != "2006-01-02T15:04:05.999999" {
				return t.UTC().Format("2006-01-02T15:04:05.999999") + "+00:00"
			}
			return ts
		}
	}
	return ts
}

// escapeScopeValue 对齐 _escape_scope_value。
func escapeScopeValue(val string) string {
	r := strings.NewReplacer("%", "%25", "&", "%26", "=", "%3D")
	return r.Replace(val)
}

// buildSessionScope 对齐 _build_session_scope (键字典序)。
func buildSessionScope(filters map[string]any) string {
	keys := []string{"agent_id", "run_id", "user_id"} // sorted
	var parts []string
	for _, k := range keys {
		if v, ok := filters[k].(string); ok && v != "" {
			parts = append(parts, k+"="+escapeScopeValue(v))
		}
	}
	return strings.Join(parts, "&")
}

// buildFiltersAndMetadata 对齐 _build_filters_and_metadata (actor_id 暂不支持, HTTP 面未暴露)。
func buildFiltersAndMetadata(userID, agentID, runID string, inputMetadata, inputFilters map[string]any, warn func(string)) (map[string]any, map[string]any, error) {
	base := map[string]any{}
	for k, v := range inputMetadata {
		base[k] = v
	}
	base = stripIdentityKeys(base, map[string]any{}, warn)
	effective := map[string]any{}
	for k, v := range inputFilters {
		effective[k] = v
	}
	provided := 0
	if userID != "" {
		base["user_id"] = userID
		effective["user_id"] = userID
		provided++
	}
	if agentID != "" {
		base["agent_id"] = agentID
		effective["agent_id"] = agentID
		provided++
	}
	if runID != "" {
		base["run_id"] = runID
		effective["run_id"] = runID
		provided++
	}
	if provided == 0 {
		return nil, nil, fmt.Errorf("At least one of 'user_id', 'agent_id', or 'run_id' must be provided.")
	}
	return base, effective, nil
}

// scopeFiltersOnly 抽取仅 user/agent/run 非空键 (对齐 {k: v for ...})。
func scopeFiltersOnly(filters map[string]any) map[string]any {
	out := map[string]any{}
	for _, k := range []string{"user_id", "agent_id", "run_id"} {
		if v, ok := filters[k].(string); ok && v != "" {
			out[k] = v
		}
	}
	return out
}

// payloadKeys 复刻 Python promoted/core 键集合逻辑 (get/get_all/search 共用)。
var promotedPayloadKeys = []string{"user_id", "agent_id", "run_id", "actor_id", "role", "attributed_to", "expiration_date"}

func isCoreOrPromotedKey(k string) bool {
	switch k {
	case "data", "hash", "created_at", "updated_at", "id", "text_lemmatized", "attributed_to":
		return true
	}
	for _, p := range promotedPayloadKeys {
		if k == p {
			return true
		}
	}
	return false
}

// entityBoostWeight 对齐 scoring.ENTITY_BOOST_WEIGHT。
const entityBoostWeight = 0.5
