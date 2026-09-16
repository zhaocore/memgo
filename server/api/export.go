package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zhao-core/memgo/server/auth"
	"github.com/zhao-core/memgo/server/store"
)

// exportLimit 导出单次上限 (与 entities 全量列表同量级)。
const exportLimit = 10000

// exportMemories GET /export (memgo 扩展: 记忆导出)。
// 带 scope 过滤 (user_id/agent_id/run_id 至少一个) = 常规鉴权面; 全量导出限 admin。
// format = json (默认) | csv | schema。
func (s *Server) exportMemories(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User) {
	q := r.URL.Query()
	userID, agentID, runID := q.Get("user_id"), q.Get("agent_id"), q.Get("run_id")
	format := q.Get("format")
	if format == "" {
		format = "json"
	}
	switch format {
	case "json", "csv", "schema":
	default:
		writeDetail(w, http.StatusBadRequest,
			"Invalid format '"+format+"'. Valid formats: json, csv, schema.")
		return
	}

	filters := map[string]any{}
	for key, val := range map[string]string{"user_id": userID, "agent_id": agentID, "run_id": runID} {
		if val != "" {
			filters[key] = val
		}
	}
	// 全量导出 (无 scope) 限 admin — 对齐 GET /memories 模式 B 的 admin 判定。
	if len(filters) == 0 {
		if ac != nil && ac.UserID != "" && ac.UserID != auth.BootstrapAdminID && ac.Role != "admin" && !ac.IsAdminLike() {
			forbidden(w, "Admin role required to export all memories. Provide user_id, agent_id, or run_id.")
			return
		}
	}

	listed, err := s.state.Memory().VectorStore.List(filters, exportLimit)
	if err != nil {
		writeUpstream(w, requestIDOf(r), err)
		return
	}
	rows := listed[0]
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, exportItemFromPayload(row.ID, row.Payload))
	}

	var body []byte
	var contentType, ext string
	switch format {
	case "csv":
		body = []byte(buildExportCSV(items))
		contentType, ext = "text/csv; charset=utf-8", "csv"
	case "schema":
		body = []byte(exportSchemaJSON)
		contentType, ext = "application/json; charset=utf-8", "schema.json"
	default:
		body, err = json.Marshal(map[string]any{"memories": items, "total": len(items)})
		if err != nil {
			writeUpstream(w, requestIDOf(r), err)
			return
		}
		contentType, ext = "application/json; charset=utf-8", "json"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"memgo-export-%s.%s\"", time.Now().UTC().Format("20060102"), ext))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// exportItemFromPayload 单条记忆 → 导出面 (id/content/scope/category/created_at; metadata 不导出)。
func exportItemFromPayload(rowID string, payload map[string]any) map[string]any {
	get := func(k string) any {
		if v, ok := payload[k]; ok && v != nil {
			return v
		}
		return nil
	}
	item := map[string]any{
		"id":         rowID,
		"content":    get("data"),
		"user_id":    get("user_id"),
		"created_at": get("created_at"),
	}
	for _, k := range []string{"agent_id", "run_id", "category"} {
		if v := get(k); v != nil && v != "" {
			item[k] = v
		}
	}
	return item
}

// buildExportCSV 固定列 CSV (encoding/csv 处理引号/换行转义)。
func buildExportCSV(items []map[string]any) string {
	var b strings.Builder
	w := csv.NewWriter(&b)
	columns := []string{"id", "content", "user_id", "agent_id", "run_id", "category", "created_at"}
	if err := w.Write(columns); err != nil {
		return ""
	}
	for _, item := range items {
		record := make([]string, 0, len(columns))
		for _, col := range columns {
			switch v := item[col].(type) {
			case nil:
				record = append(record, "")
			case string:
				record = append(record, v)
			default:
				record = append(record, fmt.Sprintf("%v", v))
			}
		}
		if err := w.Write(record); err != nil {
			return ""
		}
	}
	w.Flush()
	return b.String()
}

// exportSchemaJSON Pydantic Schema 格式: 记忆条目的 JSON Schema (类型定义, 非数据)。
const exportSchemaJSON = `{
  "title": "MemoryItem",
  "description": "MemGo memory export item",
  "type": "object",
  "properties": {
    "id": {"title": "Id", "type": "string", "format": "uuid"},
    "content": {"title": "Content", "type": "string"},
    "user_id": {"title": "User Id", "anyOf": [{"type": "string"}, {"type": "null"}]},
    "agent_id": {"title": "Agent Id", "anyOf": [{"type": "string"}, {"type": "null"}]},
    "run_id": {"title": "Run Id", "anyOf": [{"type": "string"}, {"type": "null"}]},
    "category": {"title": "Category", "anyOf": [{"type": "string"}, {"type": "null"}]},
    "created_at": {"title": "Created At", "type": "string", "format": "date-time"}
  },
  "required": ["id", "content", "created_at"]
}`
