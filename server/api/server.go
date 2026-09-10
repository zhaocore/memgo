package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/zhao-core/memgo/server/auth"
	"github.com/zhao-core/memgo/server/middleware"
	"github.com/zhao-core/memgo/server/store"
)

// Config server 启动配置。
type Config struct {
	JWTSecret     string
	AdminAPIKey   string
	AuthDisabled  bool
	DashboardURL  string
	DefaultConfig map[string]any
}

// Server 聚合依赖。
type Server struct {
	cfg      Config
	store    *store.Store
	state    *AppState
	tokens   *auth.TokenManager
	logs     *middleware.LogWriter
	limReg   *middleware.RateLimiter
	limLogin *middleware.RateLimiter
	limRef   *middleware.RateLimiter
}

// NewServer 构造。
func NewServer(cfg Config, st *store.Store, state *AppState) *Server {
	return &Server{
		cfg: cfg, store: st, state: state,
		tokens:   auth.NewTokenManager(cfg.JWTSecret),
		logs:     middleware.NewLogWriter(st),
		limReg:   middleware.NewRateLimiter(5),
		limLogin: middleware.NewRateLimiter(10),
		limRef:   middleware.NewRateLimiter(20),
	}
}

// Close 释放后台资源。
func (s *Server) Close() { s.logs.Close() }

// Routes 装配 chi 路由 (含鉴权中间件执行顺序: log 最外层计时)。
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(func(next http.Handler) http.Handler {
		return middleware.LogRequest(s.logs, next)
	})
	r.Use(func(next http.Handler) http.Handler {
		return middleware.CORS(s.cfg.DashboardURL, next)
	})

	// 公开
	r.Get("/", s.redirectDocs)
	r.Get("/docs", s.serveDocs)
	r.Get("/redoc", s.serveRedoc)
	r.Get("/openapi.json", s.serveOpenAPI)
	r.Get("/api/health", s.health)
	r.Get("/auth/setup-status", s.setupStatus)

	// 限流三端点
	r.Group(func(rg chi.Router) {
		rg.Post("/auth/register", s.wrapLimit(s.limReg, "5 per 1 minute", s.register))
		rg.Post("/auth/login", s.wrapLimit(s.limLogin, "10 per 1 minute", s.login))
		rg.Post("/auth/refresh", s.wrapLimit(s.limRef, "20 per 1 minute", s.refresh))
	})

	// 业务端点 (鉴权在 handler 内执行, 便于 401 信封与日志 auth_type 一致)
	r.Post("/memories", s.handler(s.addMemory, authRequireNone))
	r.Get("/memories", s.handler(s.listMemories, authRequireNone))
	r.Delete("/memories", s.handler(s.deleteAllMemories, authAdmin))
	r.Get("/memories/{memory_id}", s.handler(s.getMemory, authRequireNone))
	r.Put("/memories/{memory_id}", s.handler(s.updateMemory, authRequireNone))
	r.Delete("/memories/{memory_id}", s.handler(s.deleteMemory, authRequireNone))
	r.Get("/memories/{memory_id}/history", s.handler(s.memoryHistory, authRequireNone))
	r.Post("/search", s.handler(s.search, authRequireNone))
	r.Post("/reset", s.handler(s.reset, authAdmin))
	r.Get("/configure", s.handler(s.getConfig, authVerify))
	r.Post("/configure", s.handler(s.setConfig, authAdmin))
	r.Get("/configure/providers", s.handler(s.listProviders, authVerify))
	r.Post("/generate-instructions", s.handler(s.generateInstructions, authVerify))
	r.Get("/entities", s.handler(s.listEntities, authVerify))
	r.Delete("/entities/{entity_type}/{entity_id}", s.handler(s.deleteEntity, authAdmin))
	r.Get("/requests", s.handler(s.listRequests, authAdmin))
	r.Post("/auth/onboarding-complete", s.handler(s.onboardingComplete, authRequireAuth))
	r.Get("/auth/me", s.handler(s.me, authRequireAuth))
	r.Patch("/auth/me", s.handler(s.updateMe, authRequireAuth))
	r.Post("/auth/change-password", s.handler(s.changePassword, authRequireAuth))
	r.Get("/api-keys", s.handler(s.listKeys, authRequireAuth))
	r.Post("/api-keys", s.handler(s.createKey, authRequireAuth))
	r.Delete("/api-keys/{key_id}", s.handler(s.revokeKey, authRequireAuth))
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeDetail(w, http.StatusNotFound, "Not Found")
	})
	return r
}

// handler 组合: 鉴权 → handler。鉴权失败已写响应。
func (s *Server) handler(fn func(w http.ResponseWriter, r *http.Request, ac *auth.Context, user *store.User), level int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ac, user, ok := s.resolve(w, r, level)
		if !ok {
			return
		}
		fn(w, r, ac, user)
	}
}

func (s *Server) wrapLimit(rl *middleware.RateLimiter, label string, next http.HandlerFunc) http.HandlerFunc {
	return middleware.LimitMiddleware(rl, label, next).ServeHTTP
}

func (s *Server) redirectDocs(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/docs", http.StatusTemporaryRedirect)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
