# MemGo

## 快速开始（一键栈）

```bash
export POSTGRES_PASSWORD=... JWT_SECRET=$(openssl rand -base64 48) OPENAI_API_KEY=sk-...
make up            # memgo-server(:8888) + pgvector(:8432); dashboard: make up-dashboard
make bootstrap     # seed: setup-status → register → 建 API key
make health        # 三探
make down          # 停栈清卷
```

单容器连外部 Postgres：`make run-local`（打印 run 命令模板）。

## CLI（memgo）

```bash
go build -o bin/memgo ./cli/go/cmd/memgo
bin/memgo init --api-key <key> --user-id alice
bin/memgo add "I like hiking" -u alice
bin/memgo search "hiking" -u alice -o table
# 打自托管 server: MEMGO_BASE_URL=http://localhost:8888
```

命令面与上游 python/node CLI 三向 parity（golden：tests/contract/cli_parity_golden.json）。
未迁移项（显式）：agent-rush / agent-mode / init 邮箱验证流程 / plugin_sync

## 环境变量表（Go server）

| 变量 | 默认 | 说明 |
|------|------|------|
| `JWT_SECRET` | —（auth 开启时必填，拒启） | JWT 签名（HS256） |
| `ADMIN_API_KEY` | — | 遗留管理密钥（X-API-Key 恒时比较） |
| `AUTH_DISABLED` | false | 仅本地开发 |
| `POSTGRES_HOST/PORT/DB/USER/PASSWORD` | postgres/5432/postgres/postgres/postgres | **pgvector 记忆库**（postgres 库） |
| `POSTGRES_COLLECTION_NAME` | memories | 向量 collection |
| `APP_DB_NAME` | memgo_app | **应用库**（users/api_keys/request_logs/jtis/settings） |
| `OPENAI_API_KEY` | — | 默认 LLM+Embedder（openai） |
| `OPENAI_BASE_URL` | — | 自托管/打桩端点（LLM 与 embedder 共用） |
| `MEMGO_DEFAULT_LLM_MODEL` | gpt-5-mini | LLM 模型 |
| `MEMGO_DEFAULT_EMBEDDER_MODEL` | text-embedding-3-small | Embedder 模型 |
| `HISTORY_DB_PATH` | /app/history/history.db | SQLite 历史库 |
| `DASHBOARD_URL` | http://localhost:3000 | CORS 允许源 |
| `MEMGO_TELEMETRY` | true | 遥测开关 |
| `MEMGO_TELEMETRY_STATE_PATH` | /app/history/telemetry.json | 遥测状态文件 |
| `PORT` | 8000 | Go server 监听端口（新增） |

双库拓扑（红线）：**记忆向量只进 pgvector 库（postgres）；server 表只进 memgo_app**。

## 契约测试（P0 安全网）

```bash
make contract                 # 全新栈起 python server → 251 断言回归
CONTRACT_IMPL=go make contract  # 同一套测试打 Go server（goldens 不变）
./tests/contract/cli_oss_smoke.sh  # CLI OSS backend 往返（HOME 隔离）
```

## 压测基线（stub 路径, N=50, 宿主裸跑）

| 实现 | add P50/P95 | search P50/P95 |
|------|-------------|----------------|
| Go server | 73ms / 75ms | 72ms / 75ms |
| Python server | 267ms / 281ms | 265ms / 270ms |

复跑：`tests/bench/bench.sh [N]`（TARGET/KEY 环境变量）。真实 LLM 路径延迟由 LLM 主导，未纳入对比。
