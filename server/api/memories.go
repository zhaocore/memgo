package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zhao-core/memgo/core/memory"
	"github.com/zhao-core/memgo/server/auth"
	"github.com/zhao-core/memgo/server/store"
)

// chiParam 路径参数。
func chiParam(r *http.Request, name string) string { return chi.URLParam(r, name) }

// addMemory POST /memories (合同 §2.1)。
func (s *Server) addMemory(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	// MemoryCreate 校验: messages 必填, 每项 {role,content} 必填 (pydantic 字段序)
	var errs []pyErr
	rawMessages, present := fields["messages"]
	if !present {
		errs = append(errs, missingErr([]any{"body", "messages"}, map[string]any{}))
	} else {
		var items []json.RawMessage
		if err := json.Unmarshal(rawMessages, &items); err != nil {
			var anyVal any
			_ = json.Unmarshal(rawMessages, &anyVal)
			errs = append(errs, pyErr{Type: "list_type", Loc: []any{"body", "messages"},
				Msg: "Input should be a valid list", Input: anyVal})
		} else {
			for i, item := range items {
				var one map[string]json.RawMessage
				if err := json.Unmarshal(item, &one); err != nil {
					errs = append(errs, pyErr{Type: "dict_type", Loc: []any{"body", "messages", i},
						Msg: "Input should be a valid dictionary", Input: json.RawMessage(item)})
					continue
				}
				for _, f := range []string{"role", "content"} {
					rawF, okF := one[f]
					if !okF || string(rawF) == "null" {
						var inputVal map[string]any
						_ = json.Unmarshal(item, &inputVal)
						errs = append(errs, missingErr([]any{"body", "messages", i, f}, inputVal))
						continue
					}
					var sVal string
					if err := json.Unmarshal(rawF, &sVal); err != nil {
						errs = append(errs, pyErr{Type: "string_type", Loc: []any{"body", "messages", i, f},
							Msg: "Input should be a valid string", Input: json.RawMessage(rawF)})
					}
				}
			}
		}
	}
	// 可选字符串字段
	for _, name := range []string{"user_id", "agent_id", "run_id", "expiration_date", "memory_type", "prompt"} {
		if raw, ok := fields[name]; ok && string(raw) != "null" {
			var sVal string
			if err := json.Unmarshal(raw, &sVal); err != nil {
				errs = append(errs, pyErr{Type: "string_type", Loc: []any{"body", name},
					Msg: "Input should be a valid string", Input: json.RawMessage(raw)})
			}
		}
	}
	if raw, ok := fields["infer"]; ok && string(raw) != "null" {
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			errs = append(errs, pyErr{Type: "bool_type", Loc: []any{"body", "infer"},
				Msg: "Input should be a valid boolean", Input: json.RawMessage(raw)})
		}
	}
	if raw, ok := fields["metadata"]; ok && string(raw) != "null" {
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			errs = append(errs, pyErr{Type: "object_type", Loc: []any{"body", "metadata"},
				Msg: "Input should be a valid dictionary or object", Input: json.RawMessage(raw)})
		}
	}
	if len(errs) > 0 {
		write422(w, errs)
		return
	}

	var params memory.AddParams
	params.UserID = strFromFields(fields, "user_id")
	params.AgentID = strFromFields(fields, "agent_id")
	params.RunID = strFromFields(fields, "run_id")
	params.Metadata = mapFromFields(fields, "metadata")
	if raw, ok := fields["expiration_date"]; ok && string(raw) != "null" {
		params.ExpirationDate = strFromFields(fields, "expiration_date")
	}
	params.Infer = true
	if raw, ok := fields["infer"]; ok && string(raw) != "null" {
		_ = json.Unmarshal(raw, &params.Infer)
	}
	params.MemoryType = strFromFields(fields, "memory_type")
	params.Prompt = strFromFields(fields, "prompt")

	var messages []map[string]any
	_ = json.Unmarshal(rawMessages, &messages)

	if params.UserID == "" && params.AgentID == "" && params.RunID == "" {
		writeDetail(w, http.StatusBadRequest, "At least one identifier (user_id, agent_id, run_id) is required.")
		return
	}
	results, err := s.state.Memory().Add(messages, params)
	if err != nil {
		s.writeCoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func strFromFields(fields map[string]json.RawMessage, name string) string {
	raw, ok := fields[name]
	if !ok || string(raw) == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return ""
	}
	return s
}

func mapFromFields(fields map[string]json.RawMessage, name string) map[string]any {
	raw, ok := fields[name]
	if !ok || string(raw) == "null" {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}

// writeCoreError 对齐 _client_error: not found → 404, 其余 ValueError 类 → 400, 其他 → 502。
func (s *Server) writeCoreError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case isNotFoundErr(err):
		writeDetail(w, http.StatusNotFound, err.Error())
	case isClientErr(err):
		writeDetail(w, http.StatusBadRequest, err.Error())
	default:
		writeUpstream(w, requestIDOf(r), err)
	}
}

// serializeModeB 对齐 _serialize_memory (保留键集不含 actor_id/role/attributed_to)。
func serializeModeB(rowID string, payload map[string]any) map[string]any {
	reserved := map[string]bool{
		"data": true, "user_id": true, "agent_id": true, "run_id": true, "hash": true,
		"created_at": true, "updated_at": true, "expiration_date": true,
	}
	meta := map[string]any{}
	for k, v := range payload {
		if !reserved[k] {
			meta[k] = v
		}
	}
	get := func(k string) any {
		if v, ok := payload[k]; ok {
			return v
		}
		return nil
	}
	return map[string]any{
		"id":              rowID,
		"memory":          get("data"),
		"user_id":         get("user_id"),
		"agent_id":        get("agent_id"),
		"run_id":          get("run_id"),
		"hash":            get("hash"),
		"expiration_date": get("expiration_date"),
		"metadata":        meta,
		"created_at":      get("created_at"),
		"updated_at":      get("updated_at"),
	}
}

// listMemories GET /memories 双模式 (合同 §2.2, T6)。
func (s *Server) listMemories(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	q := r.URL.Query()
	userID, agentID, runID := q.Get("user_id"), q.Get("agent_id"), q.Get("run_id")
	topK, okTopK := validateQueryInt(w, r, "top_k", 0, 0, 1000, true, true)
	if !okTopK {
		return
	}
	showExpired := q.Get("show_expired") == "true"
	mem := s.state.Memory()
	if userID == "" && agentID == "" && runID == "" {
		// 模式 B: admin 判定读 auth_type (main.py 411-440 语义)
		if ac != nil && ac.UserID != "" && ac.UserID != auth.BootstrapAdminID && ac.Role != "admin" && !ac.IsAdminLike() {
			forbidden(w, "Admin role required to list all memories.")
			return
		}
		limit := 1000
		if topK > 0 {
			limit = topK
		}
		listed, err := mem.VectorStore.List(nil, limit)
		if err != nil {
			writeUpstream(w, requestIDOf(r), err)
			return
		}
		rows := listed[0]
		results := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			results = append(results, serializeModeB(row.ID, row.Payload))
		}
		writeJSON(w, http.StatusOK, map[string]any{"results": results})
		return
	}
	filters := map[string]any{}
	if userID != "" {
		filters["user_id"] = userID
	}
	if agentID != "" {
		filters["agent_id"] = agentID
	}
	if runID != "" {
		filters["run_id"] = runID
	}
	gp := memory.GetAllParams{Filters: filters, TopK: 20, ShowExpired: showExpired}
	if topK > 0 {
		gp.TopK = topK
	}
	res, err := mem.GetAll(gp)
	if err != nil {
		s.writeCoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// getMemory GET /memories/{id} (不存在 → null)。
func (s *Server) getMemory(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	item, err := s.state.Memory().Get(chiParam(r, "memory_id"))
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	if item == nil {
		writeJSON(w, http.StatusOK, nil)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// updateMemory PUT /memories/{id} — 红线 T1: RawMessage presence 区分 absent/null。
func (s *Server) updateMemory(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	params := memory.UpdateParams{MemoryID: chiParam(r, "memory_id")}
	if raw, present := fields["text"]; present {
		if string(raw) == "null" {
			// fields_set 含 text 但值为 null → SDK update(data=None) → 400 (契约实测)
			if _, err := s.state.Memory().Update(params); err != nil {
				s.writeCoreError(w, r, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"message": "Memory updated successfully!"})
			return
		}
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			write422(w, []pyErr{{Type: "string_type", Loc: []any{"body", "text"},
				Msg: "Input should be a valid string", Input: json.RawMessage(raw)}})
			return
		}
		params.Text = &text
	}
	if raw, present := fields["metadata"]; present && string(raw) != "null" {
		params.Metadata = map[string]any{}
		if err := json.Unmarshal(raw, &params.Metadata); err != nil {
			write422(w, []pyErr{{Type: "object_type", Loc: []any{"body", "metadata"},
				Msg: "Input should be a valid dictionary or object", Input: json.RawMessage(raw)}})
			return
		}
	}
	if raw, present := fields["expiration_date"]; present {
		params.ExpirationSet = true
		if string(raw) != "null" {
			var exp string
			if err := json.Unmarshal(raw, &exp); err != nil {
				write422(w, []pyErr{{Type: "string_type", Loc: []any{"body", "expiration_date"},
					Msg: "Input should be a valid string", Input: json.RawMessage(raw)}})
				return
			}
			params.ExpirationDate = exp
		}
	}
	if _, err := s.state.Memory().Update(params); err != nil {
		s.writeCoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Memory updated successfully!"})
}

// deleteMemory DELETE /memories/{id}。
func (s *Server) deleteMemory(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	if _, err := s.state.Memory().Delete(chiParam(r, "memory_id")); err != nil {
		s.writeCoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Memory deleted successfully"})
}

// deleteAllMemories DELETE /memories (admin)。
func (s *Server) deleteAllMemories(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	q := r.URL.Query()
	p := memory.DeleteAllParams{UserID: q.Get("user_id"), AgentID: q.Get("agent_id"), RunID: q.Get("run_id")}
	if p.UserID == "" && p.AgentID == "" && p.RunID == "" {
		writeDetail(w, http.StatusBadRequest, "At least one identifier is required.")
		return
	}
	if _, err := s.state.Memory().DeleteAll(p); err != nil {
		s.writeCoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "All relevant memories deleted"})
}

// memoryHistory GET /memories/{id}/history (SQLite 行原样)。
func (s *Server) memoryHistory(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	rows, err := s.state.Memory().History(chiParam(r, "memory_id"))
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, h := range rows {
		row := map[string]any{
			"id": h.ID, "memory_id": h.MemoryID, "old_memory": h.OldMemory,
			"new_memory": h.NewMemory, "event": h.Event, "created_at": h.CreatedAt,
			"updated_at": h.UpdatedAt, "is_deleted": h.IsDeleted, "actor_id": h.ActorID, "role": h.Role,
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, out)
}
