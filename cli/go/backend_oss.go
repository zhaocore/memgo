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

// OSSBackend 打自托管 MemGo server (doc-02 合同, X-API-Key)。
// 纯增量 (doc-01 §4.3): 命令层零改动, 工厂按 base_url 选择。
type OSSBackend struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// NewOSSBackend 构造。
func NewOSSBackend(cfg *Config) *OSSBackend {
	return &OSSBackend{
		baseURL: strings.TrimRight(cfg.Platform.BaseURL, "/"),
		apiKey:  cfg.Platform.APIKey,
		client:  &http.Client{Timeout: 600 * time.Second},
	}
}

func (b *OSSBackend) headers() map[string]string {
	return map[string]string{
		"X-API-Key":    b.apiKey,
		"Content-Type": "application/json",
	}
}

// request 对齐 doc-02 错误面 (400/401/403/404 detail 信封)。
func (b *OSSBackend) request(method, path string, payload any, query url.Values) (any, error) {
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
	status, raw, err := httpDo(context.Background(), b.client, method, full, b.headers(), body)
	if err != nil {
		return nil, errAPI(fmt.Sprintf("请求失败: %v", err))
	}
	switch status {
	case http.StatusUnauthorized:
		return nil, errAuth("Authentication failed. Your API key may be invalid or expired.")
	case http.StatusForbidden:
		return nil, errAuth(fmt.Sprintf("Forbidden: %s", ossDetail(raw)))
	case http.StatusNotFound:
		return nil, errNotFound(ossDetail(raw))
	}
	if status >= 400 {
		return nil, errAPI(fmt.Sprintf("HTTP %d from %s: %s", status, path, ossDetail(raw)))
	}
	var data any
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, errAPI(fmt.Sprintf("响应非 JSON: %s", string(raw)))
	}
	return data, nil
}

func ossDetail(raw []byte) string {
	if m, err := jsonMap(raw); err == nil {
		if d, ok := m["detail"].(string); ok {
			return d
		}
	}
	return string(raw)
}

// Add 对齐 doc-02 POST /memories (OSS 无 app_id/immutable/structured_data/timestamp)。
func (b *OSSBackend) Add(p AddParams) (map[string]any, error) {
	if p.IDs.AppID != "" {
		return nil, errAPI("app_id is not supported on the OSS backend")
	}
	if p.Immutable {
		return nil, errAPI("--immutable is not supported on the OSS backend")
	}
	if p.StructuredDataSchema != "" || p.CustomCategories != "" || p.AgentCustomInstructions != "" {
		return nil, errAPI("platform-only extraction options are not supported on the OSS backend")
	}
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
	if p.IDs.RunID != "" {
		payload["run_id"] = p.IDs.RunID
	}
	if p.Metadata != nil {
		payload["metadata"] = p.Metadata
	}
	if !p.Infer {
		payload["infer"] = false
	}
	if p.Expires != "" {
		payload["expiration_date"] = p.Expires
	}
	if p.CustomInstructions != "" {
		payload["prompt"] = p.CustomInstructions
	}
	data, err := b.request(http.MethodPost, "/memories", payload, nil)
	if err != nil {
		return nil, err
	}
	m, _ := data.(map[string]any)
	return m, nil
}

// Search 对齐 doc-02 POST /search (顶层 id deprecated → 直接进 filters)。
func (b *OSSBackend) Search(p SearchParams) ([]map[string]any, error) {
	if p.Keyword {
		return nil, errAPI("--keyword is not supported on the OSS backend")
	}
	if p.ReferenceDate != "" || p.LatestOnly {
		return nil, errAPI("--reference-date/--latest-only are not supported on the OSS backend")
	}
	payload := map[string]any{"query": p.Query}
	filters := map[string]any{}
	for k, v := range p.FilterJSON {
		filters[k] = v
	}
	if p.IDs.UserID != "" {
		filters["user_id"] = p.IDs.UserID
	}
	if p.IDs.AgentID != "" {
		filters["agent_id"] = p.IDs.AgentID
	}
	if p.IDs.RunID != "" {
		filters["run_id"] = p.IDs.RunID
	}
	if p.IDs.AppID != "" {
		return nil, errAPI("app_id is not supported on the OSS backend")
	}
	if len(filters) > 0 {
		payload["filters"] = filters
	}
	payload["top_k"] = p.TopK
	payload["threshold"] = p.Threshold
	if p.ShowExpired {
		payload["show_expired"] = true
	}
	data, err := b.request(http.MethodPost, "/search", payload, nil)
	if err != nil {
		return nil, err
	}
	return resultsOf(data), nil
}

// Get 对齐 doc-02 GET /memories/{id} (不存在 → null)。
func (b *OSSBackend) Get(memoryID string) (map[string]any, error) {
	data, err := b.request(http.MethodGet, "/memories/"+url.PathEscape(memoryID), nil, nil)
	if err != nil {
		return nil, err
	}
	m, _ := data.(map[string]any)
	return m, nil
}

// ListMemories 对齐 doc-02 GET /memories (模式 A; 分页/类目/日期 OSS 不支持)。
func (b *OSSBackend) ListMemories(p ListParams) ([]map[string]any, error) {
	if p.Page > 1 || p.Category != "" || p.After != "" || p.Before != "" || p.LatestOnly {
		return nil, errAPI("pagination/category/date filters are not supported on the OSS backend")
	}
	q := url.Values{}
	if p.IDs.UserID != "" {
		q.Set("user_id", p.IDs.UserID)
	}
	if p.IDs.AgentID != "" {
		q.Set("agent_id", p.IDs.AgentID)
	}
	if p.IDs.RunID != "" {
		q.Set("run_id", p.IDs.RunID)
	}
	if p.IDs.AppID != "" {
		return nil, errAPI("app_id is not supported on the OSS backend")
	}
	topK := p.PageSize
	if topK <= 0 {
		topK = 100
	}
	q.Set("top_k", fmt.Sprintf("%d", topK))
	if p.ShowExpired {
		q.Set("show_expired", "true")
	}
	data, err := b.request(http.MethodGet, "/memories", nil, q)
	if err != nil {
		return nil, err
	}
	return resultsOf(data), nil
}

// Update 对齐 doc-02 PUT /memories/{id} (部分更新)。
func (b *OSSBackend) Update(p UpdateParams) (map[string]any, error) {
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
		return nil, errAPI("--timestamp is not supported on the OSS backend")
	}
	data, err := b.request(http.MethodPut, "/memories/"+url.PathEscape(p.MemoryID), payload, nil)
	if err != nil {
		return nil, err
	}
	m, _ := data.(map[string]any)
	return m, nil
}

// Delete 对齐 doc-02 DELETE /memories/{id} 与 DELETE /memories?user_id= (admin)。
func (b *OSSBackend) Delete(p DeleteParams) (map[string]any, error) {
	if p.All {
		if p.IDs.UserID == "" && p.IDs.AgentID == "" && p.IDs.RunID == "" {
			return nil, errAPI("OSS --all requires at least one scope id (user/agent/run); project-wide reset uses POST /reset via server admin")
		}
		q := url.Values{}
		if p.IDs.UserID != "" {
			q.Set("user_id", p.IDs.UserID)
		}
		if p.IDs.AgentID != "" {
			q.Set("agent_id", p.IDs.AgentID)
		}
		if p.IDs.RunID != "" {
			q.Set("run_id", p.IDs.RunID)
		}
		data, err := b.request(http.MethodDelete, "/memories", nil, q)
		if err != nil {
			return nil, err
		}
		m, _ := data.(map[string]any)
		return m, nil
	}
	if p.MemoryID == "" {
		return nil, errAPI("Either memory_id or --all is required")
	}
	if p.DeleteLinked {
		return nil, errAPI("--delete-linked is not supported on the OSS backend")
	}
	data, err := b.request(http.MethodDelete, "/memories/"+url.PathEscape(p.MemoryID), nil, nil)
	if err != nil {
		return nil, err
	}
	m, _ := data.(map[string]any)
	return m, nil
}

// DeleteEntities 对齐 doc-02 DELETE /entities/{type}/{id} (OSS 无 app)。
func (b *OSSBackend) DeleteEntities(ids EntityIDs) (map[string]any, error) {
	if ids.AppID != "" {
		return nil, errAPI("app_id is not supported on the OSS backend")
	}
	typeMap := []struct{ t, v string }{{"user", ids.UserID}, {"agent", ids.AgentID}, {"run", ids.RunID}}
	results := map[string]any{}
	count := 0
	for _, e := range typeMap {
		if e.v == "" {
			continue
		}
		count++
		data, err := b.request(http.MethodDelete,
			fmt.Sprintf("/entities/%s/%s", url.PathEscape(e.t), url.PathEscape(e.v)), nil, nil)
		if err != nil {
			return nil, err
		}
		results[e.t] = data
	}
	if count == 0 {
		return nil, errAPI("At least one entity ID is required for delete_entities.")
	}
	return results, nil
}

// Status 对齐 OSS: setup-status 探活。
func (b *OSSBackend) Status() map[string]any {
	if _, err := b.Ping(5 * time.Second); err == nil {
		return map[string]any{"connected": true, "backend": "oss", "base_url": b.baseURL}
	}
	_, err := b.Ping(5 * time.Second)
	return map[string]any{"connected": false, "backend": "oss", "error": err.Error()}
}

// Entities 对齐 doc-02 GET /entities (type: user/agent/run)。
func (b *OSSBackend) Entities(entityType string) ([]map[string]any, error) {
	if entityType == "apps" {
		return nil, errAPI("app entities are not supported on the OSS backend")
	}
	data, err := b.request(http.MethodGet, "/entities", nil, nil)
	if err != nil {
		return nil, err
	}
	items := resultsOf(data)
	singular := strings.TrimSuffix(entityType, "s")
	out := make([]map[string]any, 0, len(items))
	for _, e := range items {
		if strings.EqualFold(strOf(e["type"]), singular) {
			out = append(out, e)
		}
	}
	return out, nil
}

// ListEvents / GetEvent: OSS 合同无事件端点 → 显式错误。
func (b *OSSBackend) ListEvents() ([]map[string]any, error) {
	return nil, errAPI("events are not supported on the OSS backend")
}

func (b *OSSBackend) GetEvent(eventID string) (map[string]any, error) {
	return nil, errAPI("events are not supported on the OSS backend")
}

// Ping 对齐 GET /auth/setup-status (唯一无鉴权端点)。
func (b *OSSBackend) Ping(timeout time.Duration) (map[string]any, error) {
	client := b.client
	if timeout > 0 {
		client = &http.Client{Timeout: timeout}
	}
	status, raw, err := httpDo(context.Background(), client, http.MethodGet,
		b.baseURL+"/auth/setup-status", b.headers(), nil)
	if err != nil {
		return nil, errAPI(fmt.Sprintf("ping 失败: %v", err))
	}
	if status >= 400 {
		return nil, errAPI(fmt.Sprintf("ping HTTP %d", status))
	}
	return jsonMap(raw)
}
