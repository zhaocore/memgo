package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// PlatformBackend 对齐 cli/python backend/platform.py (api.mem0.ai, Token 鉴权, timeout 30s)。
type PlatformBackend struct {
	baseURL string
	apiKey  string
	agent   bool
	client  *http.Client
}

// NewPlatformBackend 构造。
func NewPlatformBackend(cfg *Config) *PlatformBackend {
	return &PlatformBackend{
		baseURL: strings.TrimRight(cfg.Platform.BaseURL, "/"),
		apiKey:  cfg.Platform.APIKey,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// SetAgentMode 切 Caller-Type 头。
func (b *PlatformBackend) SetAgentMode(v bool) { b.agent = v }

// headers 每请求组装 (X-Mem0-Caller-Type 随 agent 态变化)。
func (b *PlatformBackend) headers(ct string) map[string]string {
	caller := "user"
	if b.agent {
		caller = "agent"
	}
	return map[string]string{
		"Authorization":          "Token " + b.apiKey,
		"Content-Type":           "application/json",
		"X-Mem0-Source":          "cli",
		"X-Mem0-Client-Language": "go",
		"X-Mem0-Client-Version":  Version,
		"X-Mem0-Caller-Type":     caller,
	}
}

// request 对齐 _request: 401/404/400 分类, mem0_notice 捕获。
func (b *PlatformBackend) request(ctx context.Context, method, path string, payload any, query url.Values, timeout time.Duration) (any, error) {
	var body []byte
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, errAPI(fmt.Sprintf("序列化失败: %v", err))
		}
		body = raw
	}
	full := b.baseURL + path
	if len(query) > 0 {
		full += "?" + query.Encode()
	}
	client := b.client
	if timeout > 0 {
		client = &http.Client{Timeout: timeout}
	}
	status, raw, err := httpDo(ctx, client, method, full, b.headers("x"), body)
	if err != nil {
		return nil, errAPI(fmt.Sprintf("请求失败: %v", err))
	}
	switch status {
	case http.StatusUnauthorized:
		return nil, errAuth("Authentication failed. Your API key may be invalid or expired.")
	case http.StatusNotFound:
		return nil, errNotFound(fmt.Sprintf("Resource not found: %s", path))
	case http.StatusBadRequest:
		detail := string(raw)
		if m, err := jsonMap(raw); err == nil {
			if d, ok := m["detail"].(string); ok {
				detail = d
			}
		}
		return nil, errAPI(fmt.Sprintf("Bad request to %s: %s", path, detail))
	}
	if status >= 400 {
		return nil, errAPI(fmt.Sprintf("HTTP %d from %s: %s", status, path, string(raw)))
	}
	if status == http.StatusNoContent {
		return map[string]any{}, nil
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, errAPI(fmt.Sprintf("响应非 JSON: %s", string(raw)))
	}
	if m, ok := data.(map[string]any); ok {
		if n, has := m["mem0_notice"]; has {
			captureNotice(fmt.Sprintf("%v", n))
			delete(m, "mem0_notice")
		}
	}
	return data, nil
}

// captureNotice 暂存 notice (命令结束打印一次)。
var pendingNotice string

func captureNotice(n string) {
	if n != "" && n != "<nil>" {
		pendingNotice = n
	}
}

// TakeNotice 取走。
func TakeNotice() string {
	n := pendingNotice
	pendingNotice = ""
	return n
}

// resultsOf list/search 响应归一 (list | {"results"} | {"memories"})。
func resultsOf(data any) []map[string]any {
	switch v := data.(type) {
	case []any:
		return toMapList(v)
	case map[string]any:
		for _, key := range []string{"results", "memories"} {
			if l, ok := v[key].([]any); ok {
				return toMapList(l)
			}
		}
	}
	return nil
}

func toMapList(l []any) []map[string]any {
	out := make([]map[string]any, 0, len(l))
	for _, item := range l {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func strPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func anyToJSONString(v any) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

// Add 对齐 platform.add (POST /v3/memories/add/)。
func (b *PlatformBackend) Add(p AddParams) (map[string]any, error) {
	payload := map[string]any{}
	if len(p.Messages) > 0 {
		payload["messages"] = p.Messages
	} else if p.Content != "" {
		payload["messages"] = []map[string]any{{"role": "user", "content": p.Content}}
	}
	if p.IDs.UserID != "" {
		payload["user_id"] = p.IDs.UserID
	}
	if p.IDs.AgentID != "" {
		payload["agent_id"] = p.IDs.AgentID
	}
	if p.IDs.AppID != "" {
		payload["app_id"] = p.IDs.AppID
	}
	if p.IDs.RunID != "" {
		payload["run_id"] = p.IDs.RunID
	}
	if p.Metadata != nil {
		payload["metadata"] = p.Metadata
	}
	if p.Immutable {
		payload["immutable"] = true
	}
	if !p.Infer {
		payload["infer"] = false
	}
	if p.Expires != "" {
		payload["expiration_date"] = p.Expires
	}
	if p.CustomInstructions != "" {
		payload["custom_instructions"] = p.CustomInstructions
	}
	if p.AgentCustomInstructions != "" {
		payload["agent_custom_instructions"] = p.AgentCustomInstructions
	}
	if p.CustomCategories != "" {
		var v any
		if json.Unmarshal([]byte(p.CustomCategories), &v) == nil {
			payload["custom_categories"] = v
		}
	}
	if p.StructuredDataSchema != "" {
		var v any
		if json.Unmarshal([]byte(p.StructuredDataSchema), &v) == nil {
			payload["structured_data_schema"] = v
		}
	}
	if p.Timestamp != nil {
		payload["timestamp"] = *p.Timestamp
	}
	payload["source"] = "CLI"
	data, err := b.request(context.Background(), http.MethodPost, "/v3/memories/add/", payload, nil, 0)
	if err != nil {
		return nil, err
	}
	m, _ := data.(map[string]any)
	return m, nil
}

// buildFilters 对齐 _build_filters (实体 id AND; extra_filters 含 AND/OR 直通)。
func buildFilters(ids EntityIDs, extra map[string]any) any {
	if extra != nil {
		if _, has := extra["AND"]; has {
			return extra
		}
		if _, has := extra["OR"]; has {
			return extra
		}
	}
	var ands []map[string]any
	if ids.UserID != "" {
		ands = append(ands, map[string]any{"user_id": ids.UserID})
	}
	if ids.AgentID != "" {
		ands = append(ands, map[string]any{"agent_id": ids.AgentID})
	}
	if ids.AppID != "" {
		ands = append(ands, map[string]any{"app_id": ids.AppID})
	}
	if ids.RunID != "" {
		ands = append(ands, map[string]any{"run_id": ids.RunID})
	}
	for k, v := range extra {
		ands = append(ands, map[string]any{k: v})
	}
	if len(ands) == 1 {
		return ands[0]
	}
	if len(ands) > 1 {
		return map[string]any{"AND": ands}
	}
	return nil
}

// Search 对齐 platform.search (POST /v3/memories/search/)。
func (b *PlatformBackend) Search(p SearchParams) ([]map[string]any, error) {
	payload := map[string]any{"query": p.Query, "top_k": p.TopK, "threshold": p.Threshold}
	if f := buildFilters(p.IDs, p.FilterJSON); f != nil {
		payload["filters"] = f
	}
	if p.Rerank {
		payload["rerank"] = true
	}
	if p.Keyword {
		payload["keyword_search"] = true
	}
	if len(p.Fields) > 0 {
		payload["fields"] = p.Fields
	}
	if p.ShowExpired {
		payload["show_expired"] = true
	}
	if p.ReferenceDate != "" {
		payload["reference_date"] = p.ReferenceDate
	}
	if p.LatestOnly {
		payload["latest_only"] = true
	}
	payload["source"] = "CLI"
	data, err := b.request(context.Background(), http.MethodPost, "/v3/memories/search/", payload, nil, 0)
	if err != nil {
		return nil, err
	}
	return resultsOf(data), nil
}

// Get 对齐 platform.get (GET /v1/memories/{id}/)。
func (b *PlatformBackend) Get(memoryID string) (map[string]any, error) {
	data, err := b.request(context.Background(), http.MethodGet,
		"/v1/memories/"+url.PathEscape(memoryID)+"/", nil, url.Values{"source": {"CLI"}}, 0)
	if err != nil {
		return nil, err
	}
	m, _ := data.(map[string]any)
	return m, nil
}

// ListMemories 对齐 platform.list_memories (POST /v3/mories/ 分页; filters 装日期/类目)。
func (b *PlatformBackend) ListMemories(p ListParams) ([]map[string]any, error) {
	payload := map[string]any{}
	extra := map[string]any{}
	if p.Category != "" {
		extra["categories"] = map[string]any{"contains": p.Category}
	}
	if p.After != "" {
		ca, _ := extra["created_at"].(map[string]any)
		if ca == nil {
			ca = map[string]any{}
		}
		ca["gte"] = p.After
		extra["created_at"] = ca
	}
	if p.Before != "" {
		ca, _ := extra["created_at"].(map[string]any)
		if ca == nil {
			ca = map[string]any{}
		}
		ca["lte"] = p.Before
		extra["created_at"] = ca
	}
	var extraArg map[string]any
	if len(extra) > 0 {
		extraArg = extra
	}
	if f := buildFilters(p.IDs, extraArg); f != nil {
		payload["filters"] = f
	}
	if p.ShowExpired {
		payload["show_expired"] = true
	}
	if p.LatestOnly {
		payload["latest_only"] = true
	}
	payload["source"] = "CLI"
	q := url.Values{"page": {fmt.Sprintf("%d", p.Page)}, "page_size": {fmt.Sprintf("%d", p.PageSize)}}
	data, err := b.request(context.Background(), http.MethodPost, "/v3/memories/", payload, q, 0)
	if err != nil {
		return nil, err
	}
	return resultsOf(data), nil
}

// Update 对齐 platform.update (PUT /v1/memories/{id}/)。
func (b *PlatformBackend) Update(p UpdateParams) (map[string]any, error) {
	payload := map[string]any{}
	if p.Content != "" {
		payload["text"] = p.Content
	}
	if p.Metadata != nil {
		payload["metadata"] = p.Metadata
	}
	if p.ExpirationDate != "" {
		payload["expiration_date"] = p.ExpirationDate
	}
	if p.Timestamp != nil {
		payload["timestamp"] = *p.Timestamp
	}
	payload["source"] = "CLI"
	data, err := b.request(context.Background(), http.MethodPut,
		"/v1/memories/"+url.PathEscape(p.MemoryID)+"/", payload, nil, 0)
	if err != nil {
		return nil, err
	}
	m, _ := data.(map[string]any)
	return m, nil
}

// Delete 对齐 platform.delete (单条 / --all 按实体)。
func (b *PlatformBackend) Delete(p DeleteParams) (map[string]any, error) {
	if p.All {
		q := url.Values{"source": {"CLI"}}
		if p.IDs.UserID != "" {
			q.Set("user_id", p.IDs.UserID)
		}
		if p.IDs.AgentID != "" {
			q.Set("agent_id", p.IDs.AgentID)
		}
		if p.IDs.AppID != "" {
			q.Set("app_id", p.IDs.AppID)
		}
		if p.IDs.RunID != "" {
			q.Set("run_id", p.IDs.RunID)
		}
		data, err := b.request(context.Background(), http.MethodDelete, "/v1/memories/", nil, q, 0)
		if err != nil {
			return nil, err
		}
		m, _ := data.(map[string]any)
		return m, nil
	}
	if p.MemoryID == "" {
		return nil, errAPI("Either memory_id or --all is required")
	}
	q := url.Values{"source": {"CLI"}}
	if p.DeleteLinked {
		q.Set("delete_linked", "true")
	}
	data, err := b.request(context.Background(), http.MethodDelete,
		"/v1/memories/"+url.PathEscape(p.MemoryID)+"/", nil, q, 0)
	if err != nil {
		return nil, err
	}
	m, _ := data.(map[string]any)
	return m, nil
}

// DeleteEntities 对齐 platform.delete_entities (v2 端点逐实体)。
func (b *PlatformBackend) DeleteEntities(ids EntityIDs) (map[string]any, error) {
	typeMap := []struct {
		t, v string
	}{{"user", ids.UserID}, {"agent", ids.AgentID}, {"app", ids.AppID}, {"run", ids.RunID}}
	entities := map[string]string{}
	for _, e := range typeMap {
		if e.v != "" {
			entities[e.t] = e.v
		}
	}
	if len(entities) == 0 {
		return nil, errAPI("At least one entity ID is required for delete_entities.")
	}
	results := map[string]any{}
	for _, t := range []string{"user", "agent", "app", "run"} {
		id, ok := entities[t]
		if !ok {
			continue
		}
		data, err := b.request(context.Background(), http.MethodDelete,
			fmt.Sprintf("/v2/entities/%s/%s/", url.PathEscape(t), url.PathEscape(id)),
			nil, url.Values{"source": {"CLI"}}, 0)
		if err != nil {
			return nil, err
		}
		results[t] = data
	}
	return results, nil
}

// Status 对齐 platform.status。
func (b *PlatformBackend) Status() map[string]any {
	if _, err := b.Ping(0); err == nil {
		return map[string]any{"connected": true, "backend": "platform", "base_url": b.baseURL}
	}
	_, perr := b.Ping(0)
	return map[string]any{"connected": false, "backend": "platform", "error": perr.Error()}
}

// Entities 对齐 platform.entities (v1 全量拉取后客户端按类型筛)。
func (b *PlatformBackend) Entities(entityType string) ([]map[string]any, error) {
	data, err := b.request(context.Background(), http.MethodGet, "/v1/entities/", nil, nil, 0)
	if err != nil {
		return nil, err
	}
	items := resultsOf(data)
	typeMap := map[string]string{"users": "user", "agents": "agent", "apps": "app", "runs": "run"}
	target, ok := typeMap[entityType]
	if !ok {
		return items, nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, e := range items {
		if strings.EqualFold(strOf(e["type"]), target) {
			out = append(out, e)
		}
	}
	return out, nil
}

// ListEvents 对齐 platform.list_events (GET /v1/events/)。
func (b *PlatformBackend) ListEvents() ([]map[string]any, error) {
	data, err := b.request(context.Background(), http.MethodGet, "/v1/events/", nil, nil, 0)
	if err != nil {
		return nil, err
	}
	return resultsOf(data), nil
}

// GetEvent 对齐 platform.get_event (GET /v1/event/{id}/)。
func (b *PlatformBackend) GetEvent(eventID string) (map[string]any, error) {
	data, err := b.request(context.Background(), http.MethodGet,
		"/v1/event/"+url.PathEscape(eventID)+"/", nil, nil, 0)
	if err != nil {
		return nil, err
	}
	m, _ := data.(map[string]any)
	return m, nil
}

// Ping 对齐 platform.ping (5s 快速校验)。
func (b *PlatformBackend) Ping(timeout time.Duration) (map[string]any, error) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		client := &http.Client{Timeout: timeout}
		status, raw, err := httpDo(ctx, client, http.MethodGet, b.baseURL+"/v1/ping/", b.headers("u"), nil)
		if err != nil {
			return nil, errAPI(fmt.Sprintf("ping 失败: %v", err))
		}
		if status == http.StatusUnauthorized {
			return nil, errAuth("Authentication failed. Your API key may be invalid or expired.")
		}
		if status >= 400 {
			return nil, errAPI(fmt.Sprintf("ping HTTP %d", status))
		}
		return jsonMap(raw)
	}
	data, err := b.request(ctx, http.MethodGet, "/v1/ping/", nil, nil, 0)
	if err != nil {
		return nil, err
	}
	m, _ := data.(map[string]any)
	return m, nil
}

// strOf any → string。
func strOf(v any) string {
	s, _ := v.(string)
	return s
}
