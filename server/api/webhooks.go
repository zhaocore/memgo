package api

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/zhao-core/memgo/server/auth"
	"github.com/zhao-core/memgo/server/store"
	"github.com/zhao-core/memgo/server/webhook"
)

// createWebhook POST /webhooks (admin; memgo 扩展: 事件通知端点注册)。
func (s *Server) createWebhook(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	var errs []pyErr
	targetURL := validateStringField(&errs, fields, map[string]any{}, "url", true)
	name := validateStringField(&errs, fields, map[string]any{}, "name", false)
	if len(errs) > 0 {
		write422(w, errs)
		return
	}
	eventTypes, ok := validateWebhookEventTypes(w, r, fields, false)
	if !ok {
		return
	}
	if err := validateWebhookURL(*targetURL); err != nil {
		writeDetail(w, http.StatusBadRequest, err.Error())
		return
	}
	wh := &store.Webhook{ID: newRequestUUID(), Name: *name, URL: *targetURL, EventTypes: eventTypes}
	if err := s.store.CreateWebhook(r.Context(), wh); err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	writeJSON(w, http.StatusCreated, webhookToJSON(wh))
}

// listWebhooks GET /webhooks (verify; 裸数组, 对齐 listKeys 形状)。
func (s *Server) listWebhooks(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	hooks, err := s.store.ListWebhooks(r.Context())
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	out := make([]map[string]any, 0, len(hooks))
	for i := range hooks {
		out = append(out, webhookToJSON(&hooks[i]))
	}
	writeJSON(w, http.StatusOK, out)
}

// updateWebhook PUT /webhooks/{webhook_id} (admin; 部分更新: 缺省字段保留原值)。
func (s *Server) updateWebhook(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	wh, ok := s.requireWebhook(w, r)
	if !ok {
		return
	}
	fields, ok := requireObject(w, r)
	if !ok {
		return
	}
	if raw, ok := fields["url"]; ok && string(raw) != "null" {
		var targetURL string
		if err := json.Unmarshal(raw, &targetURL); err != nil {
			write422(w, []pyErr{{Type: "string_type", Loc: []any{"body", "url"},
				Msg: "Input should be a valid string", Input: json.RawMessage(raw)}})
			return
		}
		if err := validateWebhookURL(targetURL); err != nil {
			writeDetail(w, http.StatusBadRequest, err.Error())
			return
		}
		wh.URL = targetURL
	}
	if raw, ok := fields["name"]; ok && string(raw) != "null" {
		var name string
		if err := json.Unmarshal(raw, &name); err != nil {
			write422(w, []pyErr{{Type: "string_type", Loc: []any{"body", "name"},
				Msg: "Input should be a valid string", Input: json.RawMessage(raw)}})
			return
		}
		wh.Name = name
	}
	if _, ok := fields["event_types"]; ok {
		types, ok := validateWebhookEventTypes(w, r, fields, true)
		if !ok {
			return
		}
		wh.EventTypes = types
	}
	if err := s.store.UpdateWebhook(r.Context(), wh); err != nil {
		s.writeCoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, webhookToJSON(wh))
}

// deleteWebhook DELETE /webhooks/{webhook_id} (admin)。
func (s *Server) deleteWebhook(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	if _, ok := s.requireWebhook(w, r); !ok {
		return
	}
	if err := s.store.DeleteWebhook(r.Context(), chiParam(r, "webhook_id")); err != nil {
		s.writeCoreError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Webhook deleted successfully"})
}

// requireWebhook 取路径 id 并确认存在 (不存在 → 404)。
func (s *Server) requireWebhook(w http.ResponseWriter, r *http.Request) (*store.Webhook, bool) {
	wh, err := s.store.GetWebhook(r.Context(), chiParam(r, "webhook_id"))
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return nil, false
	}
	if wh == nil {
		writeDetail(w, http.StatusNotFound, "Webhook with id "+chiParam(r, "webhook_id")+" not found")
		return nil, false
	}
	return wh, true
}

// validateWebhookEventTypes 校验 event_types: 缺省 = 全部订阅; 传值必须为已知事件。
// allowedEmpty 区分 create (缺省填全部) 与 update (显式空 = 清空订阅, 合法)。
func validateWebhookEventTypes(w http.ResponseWriter, r *http.Request, fields map[string]json.RawMessage, allowedEmpty bool) ([]string, bool) {
	raw, present := fields["event_types"]
	if !present || string(raw) == "null" {
		if allowedEmpty {
			return nil, true
		}
		return webhook.AllEventTypes(), true
	}
	var items []any
	if err := json.Unmarshal(raw, &items); err != nil {
		write422(w, []pyErr{{Type: "list_type", Loc: []any{"body", "event_types"},
			Msg: "Input should be a valid list", Input: json.RawMessage(raw)}})
		return nil, false
	}
	out := make([]string, 0, len(items))
	for i, item := range items {
		t, ok := item.(string)
		if !ok {
			write422(w, []pyErr{{Type: "string_type", Loc: []any{"body", "event_types", i},
				Msg: "Input should be a valid string", Input: item}})
			return nil, false
		}
		if !webhook.ValidEventType(t) {
			writeDetail(w, http.StatusBadRequest,
				"Invalid event type '"+t+"'. Valid types: memory_add, memory_update, memory_delete, memory_categorize.")
			return nil, false
		}
		out = append(out, t)
	}
	if len(out) == 0 && !allowedEmpty {
		return webhook.AllEventTypes(), true
	}
	return out, true
}

// validateWebhookURL 只接受 http/https 绝对 URL。
func validateWebhookURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return errStr("url must be a valid http(s) URL, got: " + raw)
	}
	return nil
}

// webhookToJSON wire 形状 (时间 Z 格式对齐 store.ZFormat)。
func webhookToJSON(wh *store.Webhook) map[string]any {
	types := wh.EventTypes
	if types == nil {
		types = []string{}
	}
	return map[string]any{
		"id":          wh.ID,
		"name":        wh.Name,
		"url":         wh.URL,
		"event_types": types,
		"created_at":  store.ZFormat(wh.CreatedAt),
	}
}
