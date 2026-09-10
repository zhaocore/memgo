package cli

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// Version 对齐 python __version__ (X-MemGo-Client-Version)。
const Version = "0.2.12"

// Backend 对齐 cli/python backend/base.py 接口面。
type Backend interface {
	Add(p AddParams) (map[string]any, error)
	Search(p SearchParams) ([]map[string]any, error)
	Get(memoryID string) (map[string]any, error)
	ListMemories(p ListParams) ([]map[string]any, error)
	Update(p UpdateParams) (map[string]any, error)
	Delete(p DeleteParams) (map[string]any, error)
	DeleteEntities(ids EntityIDs) (map[string]any, error)
	Status() map[string]any
	Entities(entityType string) ([]map[string]any, error)
	ListEvents() ([]map[string]any, error)
	GetEvent(eventID string) (map[string]any, error)
	Ping(timeout time.Duration) (map[string]any, error)
}

// EntityIDs 四类实体 id。
type EntityIDs struct {
	UserID  string
	AgentID string
	AppID   string
	RunID   string
}

type AddParams struct {
	Content                 string
	Messages                []map[string]any
	IDs                     EntityIDs
	Metadata                map[string]any
	Immutable               bool
	Infer                   bool
	Expires                 string
	CustomInstructions      string
	AgentCustomInstructions string
	CustomCategories        string // JSON 原文透传
	StructuredDataSchema    string // JSON 原文透传
	Timestamp               *int64
}

type SearchParams struct {
	Query         string
	IDs           EntityIDs
	TopK          int
	Threshold     float64
	Rerank        bool
	Keyword       bool
	FilterJSON    map[string]any
	Fields        []string
	ShowExpired   bool
	ReferenceDate string
	LatestOnly    bool
}

type ListParams struct {
	IDs         EntityIDs
	Page        int
	PageSize    int
	Category    string
	After       string
	Before      string
	ShowExpired bool
	LatestOnly  bool
}

type UpdateParams struct {
	MemoryID       string
	Content        string
	Metadata       map[string]any
	ExpirationDate string
	Timestamp      *int64
}

type DeleteParams struct {
	MemoryID     string
	All          bool
	IDs          EntityIDs
	DeleteLinked bool
}

// APIError 三类后端错误 (对齐 AuthError/NotFoundError/APIError)。
type APIError struct {
	Kind string // "auth" | "not_found" | "api"
	Msg  string
}

func (e *APIError) Error() string { return e.Msg }

func errAuth(msg string) error     { return &APIError{Kind: "auth", Msg: msg} }
func errNotFound(msg string) error { return &APIError{Kind: "not_found", Msg: msg} }
func errAPI(msg string) error      { return &APIError{Kind: "api", Msg: msg} }

// httpDo 共用 HTTP 助手 (Token 鉴权由 platform 层注入头)。
func httpDo(ctx context.Context, client *http.Client, method, url string, headers map[string]string, body []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader(body))
	if err != nil {
		return 0, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw := make([]byte, 0, 4096)
	buf := make([]byte, 8192)
	for {
		n, err := resp.Body.Read(buf)
		raw = append(raw, buf[:n]...)
		if err != nil {
			break
		}
	}
	return resp.StatusCode, raw, nil
}

func bodyReader(b []byte) *byteReader {
	return &byteReader{data: b}
}

type byteReader struct {
	data []byte
	off  int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.off >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.off:])
	r.off += n
	return n, nil
}

func jsonMap(raw []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func jsonList(raw []byte) ([]map[string]any, error) {
	var l []map[string]any
	if err := json.Unmarshal(raw, &l); err != nil {
		return nil, err
	}
	return l, nil
}
