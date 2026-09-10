package api

import (
	"net/http"

	"github.com/zhao-core/memgo/core/jsonx"
	"github.com/zhao-core/memgo/server/auth"
	"github.com/zhao-core/memgo/server/store"
)

// listKeys GET /api-keys (当前用户未撤销, 无明文)。
func (s *Server) listKeys(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	keys, err := s.store.ListAPIKeysByCreator(r.Context(), user.ID)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		entry := map[string]any{
			"id": k.ID, "label": k.Label, "key_prefix": k.KeyPrefix,
			"created_at": store.ZFormat(k.CreatedAt), "last_used_at": nil,
		}
		if k.LastUsedAt != nil {
			entry["last_used_at"] = store.ZFormat(*k.LastUsedAt)
		}
		out = append(out, entry)
	}
	writeJSON(w, http.StatusOK, out)
}

// createKey POST /api-keys (201; 明文仅此一次)。
func (s *Server) createKey(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	var errs []pyErr
	label := validateStringField(&errs, fields, map[string]any{}, "label", true)
	if len(errs) > 0 {
		write422(w, errs)
		return
	}
	full, prefix, _, err := auth.GenerateAPIKey()
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	hash, err := auth.HashPassword(full)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	k := &store.APIKey{ID: newRequestUUID(), KeyPrefix: prefix, KeyHash: hash, Label: *label, CreatedBy: user.ID}
	if err := s.store.CreateAPIKey(r.Context(), k); err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	created, err := s.store.GetAPIKey(r.Context(), k.ID)
	if err != nil || created == nil {
		writeUpstream(w, requestIDOf(r), errNoKeyRow)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id": created.ID, "key": full, "label": created.Label,
		"key_prefix": created.KeyPrefix, "created_at": store.ZFormat(created.CreatedAt),
	})
}

var errNoKeyRow = errNil("no key row")

// revokeKey DELETE /api-keys/{id} (软删; 非本人/不存在 → 404; 重复撤销 → 400)。
func (s *Server) revokeKey(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	k, err := s.store.GetAPIKey(r.Context(), chiParam(r, "key_id"))
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	if k == nil || k.CreatedBy != user.ID {
		writeDetail(w, http.StatusNotFound, "API key not found.")
		return
	}
	if k.RevokedAt != nil {
		writeDetail(w, http.StatusBadRequest, "API key is already revoked.")
		return
	}
	if err := s.store.RevokeAPIKey(r.Context(), k.ID); err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "API key revoked."})
}

// listRequests GET /requests (admin; 仅 api_key 类; limit 边界)。
func (s *Server) listRequests(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	limit, ok := validateQueryInt(w, r, "limit", 50, 1, 200, true, true)
	if !ok {
		return
	}
	logs, err := s.store.ListRequestLogs(r.Context(), limit)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	out := make([]map[string]any, 0, len(logs))
	for _, l := range logs {
		out = append(out, map[string]any{
			"id": l.ID, "method": l.Method, "path": l.Path, "status_code": l.StatusCode,
			"latency_ms": jsonx.PyFloat(l.LatencyMS), "auth_type": l.AuthType, "created_at": store.ZFormat(l.CreatedAt),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
