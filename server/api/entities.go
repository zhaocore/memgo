package api

import (
	"net/http"
	"sort"
	"time"

	"github.com/zhao-core/memgo/core/memory"
	"github.com/zhao-core/memgo/server/auth"
	"github.com/zhao-core/memgo/server/store"
)

// entityInfo 聚合桶。
type entityInfo struct {
	TotalMemories int
	CreatedAt     *time.Time
	UpdatedAt     *time.Time
}

// listEntities GET /entities: 扫描 ≤10k 行按 (type,id) 聚合, 字典序 (合同 §6)。
// payload 时间戳 (isoformat +00:00) 重解析后以 pydantic Z 格式输出。
func (s *Server) listEntities(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	listed, err := s.state.Memory().VectorStore.List(nil, 10000)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	buckets := map[string]*entityInfo{}
	order := []string{}
	for _, rows := range listed {
		for _, row := range rows {
			payload := row.Payload
			created := parseEntityTS(payload["created_at"])
			updated := parseEntityTS(payload["updated_at"])
			if updated == nil {
				updated = created
			}
			for _, pair := range [][2]string{{"user", "user_id"}, {"agent", "agent_id"}, {"run", "run_id"}} {
				entityType, field := pair[0], pair[1]
				value, _ := payload[field].(string)
				if value == "" {
					continue
				}
				key := entityType + "\x00" + value
				bucket, ok := buckets[key]
				if !ok {
					bucket = &entityInfo{}
					buckets[key] = bucket
					order = append(order, key)
				}
				bucket.TotalMemories++
				if created != nil && (bucket.CreatedAt == nil || created.Before(*bucket.CreatedAt)) {
					bucket.CreatedAt = created
				}
				if updated != nil && (bucket.UpdatedAt == nil || updated.After(*bucket.UpdatedAt)) {
					bucket.UpdatedAt = updated
				}
			}
		}
	}
	sort.Strings(order)
	out := make([]map[string]any, 0, len(order))
	for _, key := range order {
		entityType := key[:indexByte(key, 0)]
		entityID := key[indexByte(key, 0)+1:]
		b := buckets[key]
		entry := map[string]any{
			"id": entityID, "type": entityType, "total_memories": b.TotalMemories,
			"created_at": nil, "updated_at": nil,
		}
		if b.CreatedAt != nil {
			entry["created_at"] = store.ZFormat(*b.CreatedAt)
		}
		if b.UpdatedAt != nil {
			entry["updated_at"] = store.ZFormat(*b.UpdatedAt)
		}
		out = append(out, entry)
	}
	writeJSON(w, http.StatusOK, out)
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// parseEntityTS 解析 payload 时间戳 (Z/+00:00/naive 兼容; 非法 → nil)。
func parseEntityTS(v any) *time.Time {
	s, ok := v.(string)
	if !ok || s == "" {
		return nil
	}
	for _, layout := range []string{
		"2006-01-02T15:04:05.999999Z07:00",
		"2006-01-02T15:04:05.999999",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

// deleteEntity DELETE /entities/{type}/{id} (admin; 等价 delete_all)。
func (s *Server) deleteEntity(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	entityType := chiParam(r, "entity_type")
	entityID := chiParam(r, "entity_id")
	field := ""
	switch entityType {
	case "user":
		field = "user_id"
	case "agent":
		field = "agent_id"
	case "run":
		field = "run_id"
	default:
		writeDetail(w, http.StatusNotFound, "Not Found")
		return
	}
	if _, err := s.state.Memory().DeleteAll(memory.DeleteAllParamsByField(field, entityID)); err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Entity deleted"})
}
