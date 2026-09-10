package auth

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strings"
)

// AuthType 四类凭据标记 (doc-02 §3.2)。
const (
	AuthTypeBearer      = "bearer"
	AuthTypeAPIKey      = "api_key"
	AuthTypeAdminAPIKey = "admin_api_key"
	AuthTypeDisabled    = "disabled"
)

// BootstrapAdminID 哨兵用户 UUID(int=0) 字面量 (doc-03 §3.3)。
const BootstrapAdminID = "00000000-0000-0000-0000-000000000000"

// Context 鉴权结果 (verify_auth → User|None 的结构化等价)。
// User 为 nil 表示 admin_api_key/disabled 且未落到具体用户。
type Context struct {
	AuthType string
	UserID   string // 具体用户 id; 哨兵路径 = BootstrapAdminID
	Role     string
}

// IsAdminLike: admin_api_key / disabled (require_admin 的无库回退依据)。
func (c *Context) IsAdminLike() bool {
	return c.AuthType == AuthTypeAdminAPIKey || c.AuthType == AuthTypeDisabled
}

// 存放 key: 供 handler 与日志中间件读取 auth_type。
type ctxKey int

const authCtxKey ctxKey = 1

// WithContext 注入。
func WithContext(ctx context.Context, c *Context) context.Context {
	return context.WithValue(ctx, authCtxKey, c)
}

// FromContext 读取 (未鉴权返回 nil)。
func FromContext(ctx context.Context) *Context {
	c, _ := ctx.Value(authCtxKey).(*Context)
	return c
}

// HeadersOf 提取两凭据头。
func HeadersOf(r *http.Request) (bearer, apiKey string) {
	authz := r.Header.Get("Authorization")
	if strings.HasPrefix(authz, "Bearer ") {
		bearer = strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
	}
	apiKey = r.Header.Get("X-API-Key")
	return bearer, apiKey
}

// ConstantEqual 恒时比较包装。
func ConstantEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
