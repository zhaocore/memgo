package api

import (
	"encoding/json"
	"net/http"

	"github.com/zhao-core/memgo/server/errpkg"
	"github.com/zhao-core/memgo/server/middleware"
)

// 鉴权链 sentinel 错误标记 (不落响应, 仅内部流转)。
var (
	errType  = errNil("type")
	errUser  = errNil("user")
	errOwner = errNil("owner")
	errKey   = errNil("key")
	errNone  = errNil("none")
)

type errNil string

func (e errNil) Error() string { return string(e) }

// writeJSON 输出。
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeDetail 通用 {"detail": ...} 信封。
func writeDetail(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]any{"detail": detail})
}

// unauthorized 401 + WWW-Authenticate: Bearer。
func unauthorized(w http.ResponseWriter, detail string) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	writeDetail(w, http.StatusUnauthorized, detail)
}

func forbidden(w http.ResponseWriter, detail string) {
	writeDetail(w, http.StatusForbidden, detail)
}

// writeUpstream 502 信封 (detail/code/request_id + 头)。
func writeUpstream(w http.ResponseWriter, rid string, cause error) {
	code := errpkg.Classify(cause)
	up := errpkg.New(code, rid)
	w.Header().Set("X-Request-ID", rid)
	writeJSON(w, http.StatusBadGateway, map[string]any{
		"detail": up.Detail, "code": up.Code, "request_id": up.RequestID,
	})
}

// requestIDOf 取请求 rid (middleware 已生成)。
func requestIDOf(r *http.Request) string { return middleware.RequestIDFrom(r) }
