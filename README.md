# MemGo — memgo Python 主体的 Go 重写

把 [memgo](https://github.com/memgoai/memgo) 的自托管主体（记忆引擎 + FastAPI server + CLI）重写为 Go。
合同基准：memgo 仓库 `architecture/doc-01~05`；本仓库验收 = 同一套黑盒契约测试对 Go 全绿（251/251）。

```
core/       记忆引擎（无 HTTP 依赖）: config / prompts / llm / embedder / vectorstore / history / entity / memory
server/     HTTP 层（doc-02 合同）: store+goose迁移 / auth / middleware / api / errpkg
cli/        memgo CLI（第三实现; 上游 python/node CLI 零改动）
cmd/        server / memgo / migrate 入口
tests/      contract/（黑盒契约套件+goldens） bench/（P95 基线）
deploy/     Dockerfile.server / docker-compose.yaml / seed.sh
```

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
# 打自托管 server: MEMGO_BASE_URL=http://localhost:8888 (域名非 api.memgo.ai 即走 OSS backend)
```

命令面与上游 python/node CLI 三向 parity（golden：tests/contract/cli_parity_golden.json）。
未迁移项（显式）：agent-rush / agent-mode / init 邮箱验证流程 / plugin_sync —— 详见计划文档 P3 节。

## 环境变量表（Go server，对齐 doc-01 §8.1）

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

## 存量 Python 部署迁移（alembic → goose）

Python server 的 memgo_app 由 alembic 管版本；Go server 用 goose。首次切换：

```bash
# 1. 停 python server（向量库与 memgo_app 数据不动）
# 2. 补录 goose 版本表（alembic head=006 校验 + DDL 探针 + 版本补录, 幂等可重跑）
APP_DB_DSN=postgres://user:pw@host:5432/memgo_app make migrate
# 3. 起 Go server（goose.Up 成为无操作）
```

注意事项：
- 向量库（postgres 库）零改动 —— Go 用同一套 pgvector 表结构（id UUID / vector / payload JSONB）
- memgo_app 表结构逐字对齐 alembic 001-006（DDL 探针校验）
- **不要混跑**：goose → alembic 反向不支持；python server 迁移后新加的 alembic 版本不会同步到 goose
- request_logs BRIN 索引、partial unique admin 索引均在 001-006 翻译范围内

## 契约测试（P0 安全网）

```bash
make contract                 # 全新栈起 python server → 251 断言回归
CONTRACT_IMPL=go make contract  # 同一套测试打 Go server（goldens 不变）
./tests/contract/cli_oss_smoke.sh  # CLI OSS backend 往返（HOME 隔离）
```

实现细节与实测合同发现（doc-02 与源码不符处）见 [tests/contract/README.md](tests/contract/README.md)。

## 压测基线（stub 路径, N=50, 宿主裸跑）

| 实现 | add P50/P95 | search P50/P95 |
|------|-------------|----------------|
| Go server | 73ms / 75ms | 72ms / 75ms |
| Python server | 267ms / 281ms | 265ms / 270ms |

复跑：`tests/bench/bench.sh [N]`（TARGET/KEY 环境变量）。真实 LLM 路径延迟由 LLM 主导，未纳入对比。

## 已知裁剪（显式）

- AsyncMemory / reranker / spaCy NER（实体抽取取无 spaCy 回退 = 恒空，与契约基线一致）——接入会改变行为，须连带更新契约基线
- CLI：agent-rush / agent-mode / init 邮箱验证流程 / plugin_sync
- Dashboard：已逐字迁入本仓库 `dashboard/`（2026-09-10），`make up-dashboard` 本地构建；UI 全流程冒烟进行中（合同兼容由契约套件保证）
