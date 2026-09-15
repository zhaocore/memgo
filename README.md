# MemGo

AI Agent 记忆层 —— 为你的 AI 应用提供持久化、统一可检索的长期记忆。Go 实现，单二进制部署。
为你的Claude、Codex、DeepSeek 等 AI 工具接入统一记忆层,跨会话记忆贯穿工作流。

## 快速开始（Docker）

```bash
export POSTGRES_PASSWORD=... JWT_SECRET=$(openssl rand -base64 48) OPENAI_API_KEY=sk-...
make up            # memgo-server(:8000) + pgvector(:8432)
make bootstrap     # 注册 → 建 API key
make down          # 停栈清卷
```

## REST API

```bash
KEY=mgsk_xxx     # bootstrap 输出的 key

# 添加记忆
curl -H "X-API-Key: $KEY" -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"I live in Berlin and love hiking."}],"user_id":"alice"}' \
  http://localhost:8000/memories

# 搜索记忆
curl -H "X-API-Key: $KEY" \
  "http://localhost:8000/search" -H "Content-Type: application/json" \
  -d '{"query":"where do I live","filters":{"user_id":"alice"}}'

# 列出记忆
curl -H "X-API-Key: $KEY" "http://localhost:8000/memories?user_id=alice"

# 查看 Swagger → http://localhost:8000/docs
# 关闭文档: DISABLE_DOCS=true make up
```

## CLI

```bash
go build -o bin/memgo ./cli/go/cmd/memgo
bin/memgo init --api-key <key> --user-id alice
bin/memgo add "I like hiking" -u alice
bin/memgo search "hiking" -u alice
```

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `JWT_SECRET` | — | JWT 签名（auth 必填） |
| `ADMIN_API_KEY` | — | 管理密钥 |
| `AUTH_DISABLED` | false | 跳过鉴权（仅本地开发） |
| `POSTGRES_HOST/PORT/DB/USER/PASSWORD` | postgres/5432/postgres/postgres/postgres | pgvector 记忆库 |
| `APP_DB_NAME` | memgo_app | 应用库（用户/密钥/日志） |
| `OPENAI_API_KEY` | — | LLM + Embedder |
| `OPENAI_BASE_URL` | — | 自定义 API 端点 |
| `MEMGO_DEFAULT_LLM_MODEL` | gpt-5-mini | — |
| `MEMGO_DEFAULT_EMBEDDER_MODEL` | text-embedding-3-small | — |
| `HISTORY_DB_PATH` | /app/history/history.db | SQLite 历史 |
| `DASHBOARD_URL` | http://localhost:3000 | CORS 允许源 |
| `DISABLE_DOCS` | false | 关闭 /docs /redoc /openapi.json |
| `PORT` | 8000 | 监听端口 |

> ⚠️ 记忆向量进 pgvector 库，用户/密钥/日志进 `memgo_app`。两库物理分离，不可混存。
