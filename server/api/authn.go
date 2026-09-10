package api

import (
	"net/http"

	"github.com/zhao-core/memgo/server/auth"
	"github.com/zhao-core/memgo/server/store"
)

// 鉴权级别。
const (
	authRequireNone = iota // 不鉴权
	authVerify             // verify_auth
	authRequireAuth        // require_auth
	authAdmin              // require_admin
)

// sentinelUser 哨兵 (UUID(int=0), role=admin)。
func sentinelUser() *store.User {
	return &store.User{ID: auth.BootstrapAdminID, Role: "admin"}
}

// resolve 对齐 verify_auth/require_auth/require_admin 三层回退链 (doc-03 §3.3)。
// 写出 401/403 响应并返回 ok=false。
func (s *Server) resolve(w http.ResponseWriter, r *http.Request, level int) (*auth.Context, *store.User, bool) {
	ac, err := s.verifyAuth(w, r)
	if err != nil {
		return nil, nil, false
	}
	ctxOut := auth.WithContext(r.Context(), ac)
	*r = *r.WithContext(ctxOut)

	switch level {
	case authRequireNone:
		return ac, userOrNil(ac), true
	case authVerify:
		return ac, userOrNil(ac), true
	case authRequireAuth:
		user, ok := s.requireAuthUser(w, r, ac)
		if !ok {
			return nil, nil, false
		}
		return ac, user, true
	case authAdmin:
		user, ok := s.requireAdminUser(w, r, ac)
		if !ok {
			return nil, nil, false
		}
		return ac, user, true
	}
	http.Error(w, "unknown auth level", http.StatusInternalServerError)
	return nil, nil, false
}

func userOrNil(ac *auth.Context) *store.User {
	if ac == nil || ac.UserID == "" || ac.UserID == auth.BootstrapAdminID {
		return nil
	}
	// 具体用户由调用方按需查库; 此处返回哨兵化占位不适用, 直接返回 nil 并由 requireAuthUser 查
	return nil
}

// verifyAuth 执行凭据解析 (失败时已写 401)。
func (s *Server) verifyAuth(w http.ResponseWriter, r *http.Request) (*auth.Context, error) {
	bearer, apiKey := auth.HeadersOf(r)
	if bearer != "" {
		claims, err := s.tokens.Decode(bearer)
		if err != nil {
			unauthorized(w, "Invalid or expired token.")
			return nil, err
		}
		if claims.Type != "access" {
			unauthorized(w, "Invalid token type.")
			return nil, errType
		}
		user, err := s.store.GetUser(r.Context(), claims.Subject)
		if err != nil || user == nil {
			unauthorized(w, "User not found.")
			return nil, errUser
		}
		return &auth.Context{AuthType: auth.AuthTypeBearer, UserID: user.ID, Role: user.Role}, nil
	}
	if apiKey != "" {
		if auth.CompareLegacyAdminKey(apiKey, s.cfg.AdminAPIKey) {
			return &auth.Context{AuthType: auth.AuthTypeAdminAPIKey}, nil
		}
		prefix := auth.KeyPrefixOf(apiKey)
		candidates, err := s.store.APIKeyCandidatesByPrefix(r.Context(), prefix)
		if err != nil {
			unauthorized(w, "Invalid API key.")
			return nil, err
		}
		for _, cand := range candidates {
			if auth.VerifyPassword(apiKey, cand.KeyHash) {
				_ = s.store.TouchAPIKeyLastUsed(r.Context(), cand.ID)
				owner, err := s.store.GetUser(r.Context(), cand.CreatedBy)
				if err != nil || owner == nil {
					unauthorized(w, "API key owner not found.")
					return nil, errOwner
				}
				return &auth.Context{AuthType: auth.AuthTypeAPIKey, UserID: owner.ID, Role: owner.Role}, nil
			}
		}
		unauthorized(w, "Invalid API key.")
		return nil, errKey
	}
	if s.cfg.AuthDisabled {
		return &auth.Context{AuthType: auth.AuthTypeDisabled}, nil
	}
	unauthorized(w, "Authentication required. Provide a Bearer token or X-API-Key header.")
	return nil, errNone
}

// requireAuthUser: admin_api_key/disabled 落首用户, 否则 401。
func (s *Server) requireAuthUser(w http.ResponseWriter, r *http.Request, ac *auth.Context) (*store.User, bool) {
	if ac.UserID != "" && ac.UserID != auth.BootstrapAdminID {
		user, err := s.store.GetUser(r.Context(), ac.UserID)
		if err == nil && user != nil {
			return user, true
		}
	}
	if ac.IsAdminLike() {
		first, err := s.store.FirstUser(r.Context())
		if err == nil && first != nil {
			return first, true
		}
	}
	unauthorized(w, "Authentication required.")
	return nil, false
}

// requireAdminUser: 三分支回退 (doc-03 §3.3 require_admin)。
func (s *Server) requireAdminUser(w http.ResponseWriter, r *http.Request, ac *auth.Context) (*store.User, bool) {
	if ac.UserID != "" && ac.UserID != auth.BootstrapAdminID {
		user, err := s.store.GetUser(r.Context(), ac.UserID)
		if err == nil && user != nil {
			if user.Role != "admin" {
				forbidden(w, "Admin role required.")
				return nil, false
			}
			return user, true
		}
	}
	if ac.IsAdminLike() {
		first, err := s.store.FirstUser(r.Context())
		if err == nil && first != nil {
			if first.Role != "admin" {
				forbidden(w, "Admin role required.")
				return nil, false
			}
			return first, true
		}
		return sentinelUser(), true
	}
	unauthorized(w, "Authentication required.")
	return nil, false
}
