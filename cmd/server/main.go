// memgo-server 入口: 启动校验 → DEFAULT_CONFIG → store/迁移 → AppState → HTTP (对齐 main.py:66-143)。
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zhao-core/memgo/internal/dotenv"
	"github.com/zhao-core/memgo/server/api"
	"github.com/zhao-core/memgo/server/store"
	"github.com/zhao-core/memgo/server/webhook"
)

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envTrue(k string) bool {
	switch strings.ToLower(os.Getenv(k)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func main() {
	if err := dotenv.Load(); err != nil {
		log.Fatal(err)
	}

	authDisabled := envTrue("AUTH_DISABLED")
	jwtSecret := os.Getenv("JWT_SECRET")
	adminKey := os.Getenv("ADMIN_API_KEY")

	if !authDisabled && jwtSecret == "" {
		log.Fatal("JWT_SECRET is required. Set it in .env (generate with `openssl rand -base64 48`) or set AUTH_DISABLED=true for local development only.")
	}
	if authDisabled {
		log.Print("AUTH_DISABLED is enabled. Protected endpoints are open for local development only.")
	}

	// DEFAULT_CONFIG (env 拼装, 对齐 main.py:108-139)
	defaultConfig := map[string]any{
		"version": "v1.1",
		"vector_store": map[string]any{
			"provider": "pgvector",
			"config": map[string]any{
				"host":                 envOr("POSTGRES_HOST", "postgres"),
				"port":                 jsonNumber(envOr("POSTGRES_PORT", "5432")),
				"dbname":               envOr("POSTGRES_DB", "postgres"),
				"user":                 envOr("POSTGRES_USER", "postgres"),
				"password":             envOr("POSTGRES_PASSWORD", "postgres"),
				"collection_name":      envOr("POSTGRES_COLLECTION_NAME", "memories"),
				"embedding_model_dims": jsonNumber(envOr("MEMGO_EMBEDDING_DIMS", "1536")),
			},
		},
		"llm": map[string]any{
			"provider": "openai",
			"config": map[string]any{
				"api_key":     os.Getenv("OPENAI_API_KEY"),
				"temperature": 0.2,
				"model":       envOr("MEMGO_DEFAULT_LLM_MODEL", "gpt-5-mini"),
			},
		},
		"embedder": map[string]any{
			"provider": "openai",
			"config": map[string]any{
				"api_key":        os.Getenv("OPENAI_API_KEY"),
				"model":          envOr("MEMGO_DEFAULT_EMBEDDER_MODEL", "text-embedding-3-small"),
				"embedding_dims": jsonNumber(envOr("MEMGO_EMBEDDING_DIMS", "1536")),
			},
		},
		"history_db_path": envOr("HISTORY_DB_PATH", "/app/history/history.db"),
	}

	st, err := store.Open(store.DSN())
	if err != nil {
		log.Fatalf("应用库连接失败: %v", err)
	}
	defer st.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	if err := st.Migrate(ctx); err != nil {
		log.Fatalf("迁移失败: %v", err)
	}
	cancel()

	loadOverrides := func() (map[string]any, error) {
		raw, err := st.GetSetting(context.Background(), "config_overrides")
		if err != nil || raw == nil {
			return nil, err
		}
		var m map[string]any
		if err := jsonUnmarshalString(*raw, &m); err != nil {
			return nil, err
		}
		return m, nil
	}
	state, err := api.NewAppState(defaultConfig, loadOverrides, webhook.NewDispatcher(st))
	if err != nil {
		log.Fatalf("Memory 初始化失败: %v", err)
	}

	srv := api.NewServer(api.Config{
		JWTSecret:     jwtSecret,
		AdminAPIKey:   adminKey,
		AuthDisabled:  authDisabled,
		DashboardURL:  envOr("DASHBOARD_URL", "http://localhost:3000"),
		DefaultConfig: defaultConfig,
		DisableDocs:   envTrue("DISABLE_DOCS"),
	}, st, state)
	defer srv.Close()

	addr := ":" + envOr("PORT", "8000")
	log.Printf("memgo-server listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Routes()); err != nil {
		log.Fatalf("server 退出: %v", err)
	}
}

// jsonNumber 端口字符串 → float64 (JSON 数值形态)。
func jsonNumber(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(s, "%g", &f)
	return f
}
