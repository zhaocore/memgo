package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/zhao-core/memgo/core/memory"
	"github.com/zhao-core/memgo/server/auth"
	"github.com/zhao-core/memgo/server/store"
)

// search POST /search (合同 §2.6; 顶层 deprecated id 并入 filters, T8)。
func (s *Server) search(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	var errs []pyErr
	query := validateStringField(&errs, fields, map[string]any{}, "query", true)
	for _, name := range []string{"user_id", "agent_id", "run_id"} {
		validateStringField(&errs, fields, map[string]any{}, name, false)
	}
	var threshold *float64
	if raw, present := fields["threshold"]; present && string(raw) != "null" {
		var f float64
		if err := json.Unmarshal(raw, &f); err != nil {
			errs = append(errs, pyErr{Type: "number_type", Loc: []any{"body", "threshold"},
				Msg: "Input should be a valid number", Input: json.RawMessage(raw)})
		} else {
			threshold = &f
		}
	}
	if len(errs) > 0 {
		write422(w, errs)
		return
	}

	filters := map[string]any{}
	if raw, present := fields["filters"]; present && string(raw) != "null" {
		_ = json.Unmarshal(raw, &filters)
	}
	deprecated := []string{}
	for _, name := range []string{"user_id", "agent_id", "run_id"} {
		if v := strFromFields(fields, name); v != "" {
			filters[name] = v
			deprecated = append(deprecated, name)
		}
	}
	if len(deprecated) > 0 {
		log.Printf("Top-level %s in /search is deprecated. Use filters instead.", joinComma(deprecated))
	}
	sp := memory.SearchParams{Query: *query, Filters: filters, TopK: 20, Threshold: threshold}
	if v, ok := intFromFields(fields, "top_k"); ok {
		sp.TopK = v
	}
	if v, ok := boolFromFields(fields, "explain"); ok {
		sp.Explain = v
	}
	if v, ok := boolFromFields(fields, "show_expired"); ok {
		sp.ShowExpired = v
	}
	res, err := s.state.Memory().Search(sp)
	if err != nil {
		s.writeCoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func joinComma(items []string) string {
	out := ""
	for i, x := range items {
		if i > 0 {
			out += ", "
		}
		out += x
	}
	return out
}

func intFromFields(fields map[string]json.RawMessage, name string) (int, bool) {
	raw, ok := fields[name]
	if !ok || string(raw) == "null" {
		return 0, false
	}
	var v int
	if json.Unmarshal(raw, &v) != nil {
		return 0, false
	}
	return v, true
}

func boolFromFields(fields map[string]json.RawMessage, name string) (bool, bool) {
	raw, ok := fields[name]
	if !ok || string(raw) == "null" {
		return false, false
	}
	var v bool
	if json.Unmarshal(raw, &v) != nil {
		return false, false
	}
	return v, true
}
