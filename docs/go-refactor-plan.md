# MemGo — mem0 Python → Go 重构计划

> 文档版本：v1.0 | 生成日期：2026-09-09
> 事实来源：`/Users/johnson/work/github/mem0/architecture/`（doc-01~05）
> 本文档回答：**迁移什么、用什么映射、按什么顺序做、每期怎么验收**。所有 API 合同红线直接继承 doc-02，不在此重复展开。

---

## 0. 目标与原则

**目标**：将 mem0 的主体实现从 Python 重写为 Go 项目（落地于本仓库 `MemGo`），CLI 部分尽量迁移。

**继承的硬约束**（来自 doc-01 头部约束，Go 重构同样适用）：

1. **自托管 REST API 合同不变**——doc-02 是逐端点对照基准，路径/鉴权/响应形状/错误信封/状态码全部保持
2. **Dashboard（Next.js）零改动**——它只依赖 HTTP 合同，Go server 满足合同即天然兼容
3. **CLI：上游双 CLI 保留不动，只新增 Go CLI**——python/node 两个 CLI 是存量资产，零改动；Go CLI 是新增的第三个实现，命令、选项、输出行为以双 CLI 为对齐目标

**总原则**：

- **合同快照先行**：先用黑盒契约测试锁死 Python server 行为（golden 基线），Go 实现对着同一套测试收敛
- **行为优先于结构**：响应 JSON 形状、状态码、错误文案（英文字符串原样保留）以 Python 实测为准，不自作主张"改进"
- **每期独立可验证**：每期以契约测试全绿为验收，不以"感觉写完了"为验收
- **Python SDK / TS SDK / 平台 API 不动**：Go 是并行实现，不是替换上游

---

## 1. 范围决策：迁移 / 不迁移

| 上游资产 | 处置 | 理由 |
|----------|------|------|
| `mem0/memory/`（Memory 核心引擎，3868 行） | **Go 重写** | 重构主体 |
| `mem0/configs/prompts.py`（1062 行） | **Go 重写，prompt 文本逐字搬运** | prompt 决定 LLM 抽取/决策行为，改一字即行为漂移 |
| `mem0/llms`、`embeddings`、`vector_stores`、`reranker` | **Go 重写、按需裁剪**（见 §2 决策） | server 合同只暴露内置 provider 集 |
| `mem0/memory/storage.py`（SQLite 历史库） | **Go 重写** | history 端点数据源，schema 兼容 |
| `mem0/utils/entity_extraction.py` + main.py 内 entity 方法 | **Go 重写** | entity 端点与 search 实体加成依赖 |
| `server/`（FastAPI，1660 行） | **Go 重写** | 重构重点 |
| `server/alembic/versions/001-006` | **翻译为 goose SQL 迁移** | 表结构保持一致 |
| `server/dashboard/`（Next.js） | **不动** | 只依赖 HTTP 合同 |
| `mem0-ts/`（TypeScript SDK） | **不动** | 本就是 TS，与 py→go 无关 |
| `mem0/client/`（平台 MemoryClient）、平台 API 本身 | **不迁移** | SaaS 侧不可移植，且 Go CLI 需要的是"平台 HTTP 客户端"而非平台服务端 |
| `cli/python`（Typer）+ `cli/node`（Commander） | **保留不动；另增 Go CLI（第三实现）** | 双 CLI 是存量资产，option-parity 测试继续守护它们；Go CLI 走并行实现，parity golden 以两边规格为准 |
| Python SDK 本体（PyPI mem0ai） | **保留不动** | 存量用户继续可用；Go 版以独立包交付 |

---

## 2. 关键范围决策（默认取值，可推翻）

| # | 决策 | 默认选择 | 理由 |
|---|------|----------|------|
| D1 | Go 版 provider 覆盖面 | **内置集对齐 server**：LLM = openai / anthropic / gemini；Embedder = openai / gemini；VectorStore = pgvector；Reranker 暂不做（server 合同未暴露 rerank 参数） | doc-02 §3 `GET /configure/providers` 只暴露这五个；server 合同是硬边界，其余 20+ provider 属 SDK 广度，按需后补 |
| D2 | HTTP 框架 | `chi` v5 + 标准库 `net/http` | 贴近 stdlib、无反射魔法、422/错误路径完全可控；FastAPI 自动行为（422、openapi）需手写复刻，框架越薄越好 |
| D3 | 仓库布局 | **单 Go module**：根 `go.mod`（module 名待定，暂用 `memgo`），`cmd/server` + `cmd/memgo`（CLI）+ `server/`、`cli/`、`core/` 包 | 单模块最简单；`server/`、`cli/` 空目录已建，沿用 |
| D4 | SQLite 驱动 | `modernc.org/sqlite`（纯 Go，无 CGO） | `CGO_ENABLED=0` 交叉编译 + 极小 Docker 镜像 |
| D5 | 应用库迁移工具 | goose（SQL 迁移文件），alembic 001-006 逐条翻译为 SQL，DDL 逐字一致 | 新部署全新建库即可；存量库迁移见 §8 风险 R6 |
| D6 | LLM/Embedder 客户端 | 直接 `net/http` 写小型类型化客户端，**不引 langchaingo 等重框架** | 只需 chat + structured JSON 两个能力；依赖最小化，错误分类可控 |
| D7 | 契约测试形态 | **黑盒 HTTP 套件**（bash+jq 或 pytest，与被测实现无进程内耦合），golden 快照采自 Python server | 跨语言重写的唯一可靠安全网；套件同时打 Python（采基线）与 Go（验目标） |
| D8 | CLI 合并策略 | **上游 python/node 双 CLI 不动；新增 Go CLI 作为第三实现**（cobra），二进制名 `memgo`（避开存量 `mem0` 命令），platform backend 先行、OSS backend 随 server 完成接入；Backend 接口面沿用 cli/python backend/base.py | 三实现并行：parity golden（命令×选项矩阵）成为三向对齐的唯一规范，Go CLI 不得有双 CLI 不存在的选项或命令 |

---

## 3. 技术栈映射

| 关注点 | Python 现状 | Go 选型 |
|--------|------------|---------|
| HTTP 框架 | FastAPI 0.115 + uvicorn | net/http + chi v5 |
| 请求/响应模型 | Pydantic v2（`model_fields_set` 驱动部分更新） | Go struct + 自写 presence 追踪（见 §4 难点 T1） |
| ORM/迁移 | SQLAlchemy 2 + alembic + psycopg | pgx/v5 pool + goose SQL 迁移 |
| 应用库 | PostgreSQL `mem0_app`（users/api_keys/request_logs/refresh_token_jtis/settings） | 同库同表，pgx 直连 |
| 向量库 | SDK pgvector provider | pgx + pgvector-go |
| 历史库 | SQLAlchemy + SQLite（storage.py） | modernc.org/sqlite，schema 与 storage.py 一致 |
| JWT | python-jose（HS256） | golang-jwt/jwt/v5 |
| 密码/API Key 哈希 | passlib bcrypt | golang.org/x/crypto/bcrypt（rounds 与 Python 对齐，移植时核对实际值） |
| 限流 | slowapi（按远端 IP） | golang.org/x/time/rate，按远端 IP 建桶 |
| 请求日志落库 | run_in_executor + 独立 session | 有界 channel + 后台批量 writer（goroutine） |
| 遥测 | PostHog（telemetry.py） | posthog-go 或最小 HTTP capture；事件名/频控语义照搬 |
| 遥测/历史状态文件 | `/app/history/telemetry.json`、`history.db` | 同路径同格式 |
| dotenv | python-dotenv | joho/godotenv |
| LLM providers | 20 个 provider 文件 | 3 个（openai/anthropic/gemini）小型 REST client |
| Embedders | 14 个 | 2 个（openai/gemini） |
| CLI | Typer + rich | cobra + fatih/color |
| OpenAPI 文档 | FastAPI 自动生成 /openapi.json /docs /redoc | 静态 golden openapi.json 入库直接 serve；swagger-ui 静态资源嵌入（见 §8 风险 R3） |
| 测试 | pytest | go test + 黑盒契约套件（bash/pytest 共用） |
| Lint | ruff | golangci-lint |
| 构建 | hatch + Dockerfile(python:3.12-slim) | 多阶段 Docker build，`CGO_ENABLED=0`，alpine/distroless |

---

## 4. 核心难点清单（移植时逐条对照，快照测试兜底）

按破坏风险排序。**每条都必须有契约测试用例**，全部来自 doc-02/doc-03：

| # | 难点 | 细节 | Go 侧对策 |
|---|------|------|-----------|
| T1 | **PUT /memories/{id} 部分更新语义** | pydantic `model_fields_set`：区分"未传"与"显式 null"——`{"text": null}` 会进 fields_set（显式置空），`{}` 则什么也不动 | ⚠️ Go 指针字段**无法**区分 absent 与 explicit null。必须自写 `UnmarshalJSON`：先解 `map[string]json.RawMessage` 记录出现的键集合，再解目标 struct。键集合即 `fields_set` 等价物。专测：只传 metadata 不重写内容、`{"expiration_date": null}` 清除、`{"text": null}` 置空 |
| T2 | **422 校验失败响应格式** | FastAPI/pydantic 默认格式：`{"detail":[{loc,msg,type,...}]}`（数组、含 loc 路径） | 手写校验层输出同构数组；黑盒快照锁定。若决定偏离（只保 400/401/403/404/409/502 形状），必须在此文档记录决策 + 更新 openapi golden |
| T3 | **时间戳格式** | Python `datetime.isoformat()`：无时区、零微秒时省略小数（`2026-01-01T00:00:00`），有微秒时 6 位 | Go time.Time 默认 marshal 是 RFC3339 带纳秒+时区，**必然不匹配**。统一封装 `FormatISO()`；快照测试逐字段比对 |
| T4 | **错误信封与 code 分类** | 502 `{"detail","code","request_id"}`，8 种 code 沿异常链归因（doc-02 §1.3）；`ValueError+"not found"`→404；401 带 `WWW-Authenticate: Bearer` | Go 定义等价错误分类树（上游超时/限流/鉴权/坏请求/存储不可用…），映射逻辑单独包 + 表驱动测试 |
| T5 | **鉴权矩阵全量语义** | 三种凭据优先级、bootstrap 哨兵（空库 UUID(int=0)）、require_auth 落到首个用户、admin 判定看 auth_type（doc-03 §3.3 全部回退链） | `AuthContext{User, Role, AuthType}` 枚举实现回退链；矩阵测试逐格断言（doc-02 §8 表） |
| T6 | **GET /memories 双模式形状不一致是现状** | 模式 A（带 id，SDK get_all 序列化，含 actor_id/role/attributed_to 提升键）vs 模式 B（admin 全量，`_serialize_memory` 保留键集不同）——**两者不一致本身是合同**，分别快照 | 端口两个序列化器原样；快照分别锁定 |
| T7 | **/configure 深合并 + redact** | POST 是增量 deep-merge 非替换；GET 敏感键（8 个键名，大小写不敏感，递归 dict/list）→ `"[redacted]"`；非内置 provider 400 | deepmerge 单测覆盖嵌套 dict/标量覆盖/数组整体替换语义（对齐 Python 实现行为）；redact 键集抄常量 |
| T8 | **/search 顶层 id deprecated 但必须工作** | user_id/agent_id/run_id 自动并入 filters + 服务端 warning 日志 | 照搬合并逻辑 + 日志；测试断言响应与旧实现一致 |
| T9 | **Memory.add 流水线** | LLM 抽取事实 → embed → 向量检索 → LLM 判定 ADD/UPDATE/DELETE/NONE → 写入 + SQLite 历史 + entity store；procedural 分支单独 prompt | prompts 文本**逐字节**从 prompts.py 搬（含 f-string 拼出的变量段）；LLM JSON 解析容错逻辑照搬（含解析失败回退 "I like to hike on weekends."） |
| T10 | **实体聚合** | `/entities` 内存扫 ≤10k 行按 (type,id) 聚合排序；entity store 有独立 collection（`_entity_collection_name` 后缀规则） | Go v1 保持同算法（含排序与 created_at/updated_at 语义）；性能优化属后续项不进本期 |
| T11 | **刷新 token 一次性** | jti 条件 UPDATE（`used_at IS NULL AND expires_at > now`）+ rowcount 判定防并发重放 | `UPDATE ... RETURNING` 单条 SQL 实现等价 CAS；并发重放测试（同 token 并发刷 → 恰一个成功） |
| T12 | **请求日志旁路** | 跳过清单（OPTIONS、/api/health、/docs、/redoc、/openapi.json、/requests 前缀）、失败不影响响应、只记 api_key 类 | middleware 尽力写库、错误只记日志；latency_ms 为 float |
| T13 | **限流 429** | slowapi 默认体（纯文本）+ 仅 auth 三端点（5/10/20 per min，按远端 IP） | rate.Limiter per-IP + 同响应体格式；快照锁定 429 体 |
| T14 | **hash 与 id 生成** | memory hash 算法、UUID 版本、`m0sk_` key（32 字节 urlsafe → 43 字符）、request_id（8 hex）逐一对齐 | 移植时对照 Python 实现逐行核对；快照断言长度/字符集 |

---

## 5. 目标仓库布局（MemGo）

```
MemGo/
├── go.mod / go.sum
├── cmd/
│   ├── server/main.go            # server 入口: 配置加载/启动校验/迁移/http server
│   └── memgo/main.go             # CLI 入口（cobra root）
├── core/                         # 记忆引擎（无 HTTP 依赖的 Go 库）
│   ├── memory/                   # add/search/get/get_all/update/delete/history/reset 流水线
│   ├── llm/                      # LLM 接口 + openai/anthropic/gemini 实现
│   ├── embedder/                 # Embedder 接口 + openai/gemini 实现
│   ├── vectorstore/              # VectorStore 接口 + pgvector 实现
│   ├── history/                  # SQLite 历史库（对齐 storage.py schema）
│   ├── entity/                   # entity store + 实体加成
│   ├── prompts/                  # prompt 常量（原文搬运）+ 消息拼装
│   └── config/                   # MemoryConfig + v1.1 校验 + deepmerge + redact
├── server/                       # HTTP 层（合同实现）
│   ├── api/                      # handler: memories / search / configure / entities / requests / health
│   ├── auth/                     # JWT / API key / bootstrap / AuthContext 依赖链
│   ├── middleware/               # request-id / request-log / cors / rate-limit
│   ├── store/                    # 应用库 pgx 访问 + goose 迁移（users/api_keys/logs/jtis/settings）
│   ├── errpkg/                   # 上游错误分类 + 错误信封
│   ├── telemetry/                # PostHog 两事件 + 状态文件
│   └── openapi/openapi.json      # 合同快照（serve 于 /openapi.json）
├── cli/                          # Go CLI（cobra，新增第三实现，不替代上游）
│   ├── commands/                 # memory / config / entities / events / init / identify / whoami / status
│   ├── backend/                  # Backend 接口 + platform.go + oss.go（对齐 cli/python backend/base.py 面）
│   ├── config/                   # CLI 配置与状态文件（对齐 cli/python state.py/config.py 位置与 schema）
│   └── output/                   # 表格/JSON 输出
├── tests/
│   └── contract/                 # 黑盒契约套件（bash/pytest，可打任意实现）
├── deploy/
│   ├── Dockerfile.server         # 多阶段，CGO_ENABLED=0
│   └── docker-compose.yaml       # memgo-server + pgvector + dashboard（沿用上游端口表）
├── Makefile                      # build/test/contract/up/bootstrap
└── docs/
```

**双库拓扑照搬**（doc-01 §5.5）：pgvector 记忆库（`POSTGRES_*`，postgres 库）与应用库（`APP_DB_NAME=mem0_app`）物理分离；server 表永不碰向量，记忆数据永不进 mem0_app。

---

## 6. 分期执行计划

> 执行顺序强依赖：P0 是其余各期的安全网；P1 与 P3(platform) 可并行；P2 依赖 P1；P3(oss) 依赖 P2。
> 每期验收 = 前期全部测试仍绿 + 本期清单。

### P0 — 契约基线 + 仓库骨架（硬前置）✅ 已完成（2026-09-10）

> 完成记录：251 项断言两轮独立栈全绿（capture + verify）；51 个 golden 入库 `tests/contract/goldens/`；
> 套件 = 11 个 bash 套件脚本 + OpenAI 打桩 server（Go，`/_control` 故障注入）+ Podman 契约栈（:18888 主 / :18889 AUTH_DISABLED / :8432 pg）。
> 实测合同发现（doc-02 与源码不符处）记录于 `tests/contract/README.md`：PUT text null 实为 400、user-key 403 格单 admin 拓扑不可达、bootstrap 哨兵仅覆盖 require_admin、infer=false 无去重、threshold>1 → 400。
> 容器运行时：Docker 不可用，改用 Podman（podman compose 委托 docker-compose）。

| # | 任务 | 产出 |
|---|------|------|
| 1 | Go 模块骨架：go.mod（版本与 module 名落定）、目录、Makefile、golangci-lint、CI | 可构建空项目 |
| 2 | **黑盒契约套件**（tests/contract）：鉴权矩阵逐格、错误信封（8 code + 404 映射 + X-Request-ID）、响应形状快照、PUT 部分更新三用例、GET /memories 双模式分别快照、configure 深合并/redact、refresh 一次性、search 顶层 id、429 体、openapi.json 快照 | 套件可跑 |
| 3 | 用套件打 **Python server**（compose 起真 Postgres；LLM/embedder 用 openai 兼容 stub server 打桩，`infer=false` 用例全确定性）→ 全绿 | Python 基线绿 |
| 4 | golden 快照入库：响应 JSON、openapi.json、422 格式、429 体、时间戳样例 | goldens/ |

**验收**：契约套件在未改动的 Python 主干全绿；golden 文件入库。

### P1 — 记忆引擎 core/（Go 库，无 HTTP）✅ 代码完成（2026-09-10）

> 完成记录：8 个子包全部落地并编译（38 文件 / ~5600 行）：config（解析/默认值/deepmerge/redact）、
> prompts（生成器机械搬运 tools/gen_prompts.py，**parity test 与 Python diff=0 实测 PASS**）、
> llm（openai/anthropic/gemini REST 客户端）、embedder（openai/gemini）、vectorstore（pgx+pgvector-go，
> 含 $or/$not/操作符过滤翻译）、history（modernc sqlite，history+messages 双表含驱逐）、
> entity（无 spaCy 回退语义=恒空，与契约基线一致 + upsert/remove/boosts）、memory（add 双分支/
> search 混合评分/get/get_all/update/delete/delete_all/history/reset）。
> 测试：core 六包 go test 全绿（LLM/embedder/向量库桩注入 + pgvector 真实集成测试打契约栈 :8432）。
> 已知裁剪（计划内）：AsyncMemory 不移植、reranker 不做、spaCy NER/词形还原取无 spaCy 回退路径（identity/恒空），
> NER 接入属后续增强且须连带更新契约基线。infer=true 全链路对拍在 P2 由契约套件打 Go server 完成。

| # | 任务 | 对应上游 |
|---|------|----------|
| 1 | config：MemoryConfig v1.1 解析/校验、deepmerge、redact、敏感键集 | mem0/configs/base.py、server/server_state.py 合并逻辑 |
| 2 | llm：接口 + openai/anthropic/gemini（chat + structured JSON 输出、重试+告警后上抛） | mem0/llms/{openai,anthropic,gemini}.py |
| 3 | embedder：接口 + openai/gemini | mem0/embeddings/{openai,gemini}.py |
| 4 | vectorstore：接口 + pgvector（建表/collection 语义、top_k、阈值、filters、list、delete、reset；entity collection 命名规则） | mem0/vector_stores/pgvector.py |
| 5 | history：SQLite schema 逐列对齐 storage.py，读写 API | mem0/memory/storage.py |
| 6 | prompts：常量原文搬运 + 消息拼装函数（FACT_RETRIEVAL、UPDATE_MEMORY、PROCEDURAL、MEMORY_ANSWER 等） | mem0/configs/prompts.py |
| 7 | memory：add（infer 双分支 + procedural）、search（阈值/实体加成/explain）、get/get_all/update/delete/delete_all/history/reset | mem0/memory/main.py 同步版（AsyncMemory 不移植） |
| 8 | entity：抽取 + upsert/link/remove + 查询加成 | main.py entity 方法 + utils/entity_extraction.py |

**验收**：

- [ ] `go test ./core/...` 全绿（LLM/embedder 以接口桩注入，行为断言对齐 Python 单测可移植部分）
- [ ] **跨语言对拍**：`infer=false` 全链路与 Python SDK 输出逐字段一致（同输入同输出，含 hash/时间戳格式）；`infer=true` 用录制 stub（固定 LLM 响应）对齐事件序列
- [ ] prompts 与 Python 版 diff = 0（脚本机械校验）

### P2 — server/ HTTP 层（完整 doc-02 合同）

| # | 任务 | 对应上游 |
|---|------|----------|
| 1 | 应用库 store + goose 迁移（alembic 001-006 翻译，DDL 逐字一致） | server/models.py、alembic/versions/ |
| 2 | auth：bcrypt（rounds 对齐）、m0sk_ key 生成/校验（前缀索引 + 全量比对 + last_used_at）、JWT 签发/校验、jti 一次性、三层依赖 + bootstrap 哨兵 | server/auth.py、routers/auth.py |
| 3 | middleware：X-Request-ID（8 hex）、请求日志（跳过清单 + 有界后台批量 writer）、CORS（仅 DASHBOARD_URL）、限流 | main.py:262-319、rate_limit.py |
| 4 | handlers：/memories*（双模式 GET、部分更新 PUT、批量 DELETE admin）、/search（顶层 id 兼容）、/configure*（深合并 + 重建实例 + redact）、/generate-instructions、/entities（同算法聚合）、/requests、/auth/*、/api-keys、`/`→/docs 重定向、/api/health | server/main.py 560 行 + 4 routers |
| 5 | errpkg：上游错误分类（8 code）、`not found`→404、502 信封、422/429 形状 | server/errors.py + pydantic 校验面 |
| 6 | 启动序：env 校验（无 JWT_SECRET 拒启）、DEFAULT_CONFIG 拼装、settings overrides 载入、实例构建 | main.py:66-143 |
| 7 | /openapi.json + /docs + /redoc：静态 golden + 内嵌 swagger-ui | FastAPI 自动产物 |
| 8 | 遥测两事件 + nudge、限流、Dockerfile.server + compose（memgo-server 与 python server 同 compose 可切换） | telemetry.py、部署文件 |

**验收**：

- [ ] P0 契约套件全绿打 **Go server**（同一套测试，Python 基线零改动）
- [ ] Dashboard 手工冒烟全流程：setup 向导 → 登录 → memories → 配置页保存 → api-keys 创建/撤销 → requests 页
- [ ] 并发用例：refresh 并发重放恰一成功；并发 POST /configure 无撕裂
- [ ] 镜像内无热重载；`make health` 三探全通

### P3 — Go CLI（新增第三实现，上游双 CLI 零改动）

> 边界：`cli/python`、`cli/node` 不进本计划任何任务，mem0 仓库不得出现 cli/ 变更。Go CLI 的 OSS backend 打 MemGo server（P2 产物），与上游 CLI（默认打平台 API）同型定位。

| # | 任务 | 对应上游 |
|---|------|----------|
| 1 | Backend 接口 + **platform backend**（api.mem0.ai，Token 鉴权，timeout 300s，X-Mem0-Source/Caller-Type/Client-Version 头） | cli/python backend/{base,platform}.py + cli/node platform.ts 头部约定 |
| 2 | 命令面新增（cobra）：`add/search/get/list/update/delete`、`config show/get/set`、`entity list/delete`、`event list/status`、`init`、`identify`、`whoami`、`status` | cli/python commands/* 与 cli/node 对等命令 |
| 3 | config/state：配置文件路径与 schema、state、识别逻辑 | cli/python config.py、state.py、identify_cmd.py |
| 4 | output：表格/JSON 双模式、错误退出码、branding | output.py、branding.py |
| 5 | telemetry：匿名事件 | telemetry.py、telemetry_sender.py |
| 6 | **三向 parity golden**：把 python/node 两个 option-parity 测试的命令×选项矩阵导出为 golden 规范文件，Go CLI 对照断言——Go 侧不得有矩阵外的命令/选项；缺席项必须显式列出 | cli 两语言的 parity 测试 |
| 7 | OSS backend：实现 Backend 接口打 MemGo server（X-API-Key），工厂按 base-url/backend 配置选择 | doc-01 §4.3 纯增量路径 |

**验收**：

- [ ] platform backend：对 parity golden 全命令选项比对通过；`whoami` + 一次 search 实测（打平台 API）
- [ ] OSS backend：对 MemGo server 跑通 add/search/get/list/update/delete/entity 往返
- [ ] mem0 仓库 diff 核查：cli/ 目录零改动
- [ ] 未迁移项（如 agent-mode/agent-rush 交互流程）逐条列出并给理由，不静默缺失

### P4 — 切换与收尾

| # | 任务 |
|---|------|
| 1 | compose 默认指向 memgo-server；Makefile（up/down/bootstrap/health）适配；保留 python server 镜像作对照开关 |
| 2 | 单容器生产路径 `run_local` 等价物（连外部 Postgres）；多副本部署说明 |
| 3 | 压测基线：add/search P95 对比 Python 版（真实 LLM 路径 vs stub 两套数据） |
| 4 | 文档：README、env 变量表（doc-01 §8.1 全集）、迁移指引（Python server 存量部署 → Go server：DDL 兼容性说明、alembic_version 处理） |
| 5 | 收尾核对：doc-02 合同逐条与 Go 实现交叉核对一遍；docs 集内行号引用若已失配仅记录不修 |

**验收**：

- [ ] 契约套件对 Go 实现全绿（含 422/429/openapi 决策已落档）
- [ ] dashboard 全流程冒烟 + CLI OSS backend 冒烟
- [ ] `docker compose up` 一键可用，seed 流程走通（setup-status → register → 建 key → add/search 往返）

---

## 7. 契约测试策略（P0 建立，全程复用）

```
tests/contract/
├── conftest.sh            # SERVER_URL 注入；支持 --impl python|go
├── auth_matrix.sh         # doc-02 §8 逐格
├── memories.sh            # §2 全端点形状（add/get/list 双模式/update 部分更新/delete/history/search）
├── configure.sh           # §3 深合并 + redact + provider 白名单
├── auth_flow.sh           # §4 register→login→refresh 一次性→me→change-password
├── api_keys.sh entities.sh requests.sh
├── error_envelope.sh      # §1.3：stub 触发 502 各 code；404 映射；X-Request-ID
├── goldens/               # 从 Python server 采集的响应快照
└── openapi/golden.json    # openapi 快照（diff 规则：仅允许新增）
```

要点：

- **打桩方式**：起一个 openai 兼容 mock server（固定返回 FACT_RETRIEVAL / UPDATE_DECISION 的 JSON），两个实现都通过环境变量指向它 → 同一用例两端可复现
- **确定性用例优先**：`infer=false`、CRUD、auth、configure 全确定性；`infer=true` 只断言结构（results 数组、event 枚举、键集）
- 双库真实 Postgres（pgvector + mem0_app）必起；SQLite 历史库在容器内 `/app/history` 路径不变
- Python 基线采集后**冻结**；Go 侧每次 CI 重跑同一套件比对

---

## 8. 风险登记

| # | 风险 | 缓解 |
|---|------|------|
| R1 | 部分更新语义（T1）在 Go 实现中退化为全量覆盖 | RawMessage presence 方案 + 专项测试 + review checklist 一条 |
| R2 | 时间戳/浮点/空值序列化差异导致快照漂移（T3） | 统一时间格式化工具 + 快照测试逐字段断言（不做字节级） |
| R3 | openapi.json 无法自动生成 pydantic 同构 schema | 采 Python golden 静态维护；新增端点时手动更新 + diff 测试（只允许新增） |
| R4 | 422 pydantic 格式复刻成本高且收益存疑（Dashboard 不依赖 detail 数组结构） | **默认复刻**；若成本失控，走 T2 记录偏差决策，更新契约文档 |
| R5 | prompts/LLM 解析行为差异导致 add 流水线结果不同 | prompt 文本 diff=0 机器校验 + stub LLM 录制回放对齐 |
| R6 | 存量 Python 部署数据迁移：alembic_version 表与 goose 版本表不互通 | 方案 A（默认）：Go 部署目标为**全新部署**，DDL 幂等；存量迁移单独立项（必要时保留 alembic 兼容读）。在 docs 记录 |
| R7 | 两库拓扑搞混（server 表进 pgvector 库） | store 包禁止引 pgvector 连接串；migration review 一条 |
| R8 | sqlite 驱动差异（modernc vs python sqlite3）行为差异（如时间戳存储格式） | history schema 与写入格式逐列核对 + 跨实现读取同一 db 文件的兼容性测试 |
| R9 | CLI 与平台 API 的隐藏约定（X-Mem0-* 头、snake/camel 转换、错误分类）对齐不全 | 以 cli/node platform.ts + cli/python platform.py 为对照源，逐方法移植；parity golden 锁定 |
| R10 | agent-mode/agent-rush 交互功能在 Go 侧工作量大 | P3 允许列为"尽力"项，缺席必须显式列出，不得静默 |
| R11 | **三实现并行后行为分叉**：Go CLI 与 python/node CLI 漂移，或上游双 CLI 演进后 Go 侧失配 | parity golden 规范文件为三向对齐的唯一基准；上游双 CLI 升级时先更新 golden 再对齐 Go；Go 缺席项显式列明（P3 验收） |

---

## 9. 未决问题（已全部落定 2026-09-10）

1. **module 路径**：`github.com/zhao-core/memgo` ✅
2. **D2 框架**：chi v5 ✅（备选 echo/gin 弃）
3. **T2 决策**：422 pydantic 格式**严格复刻** ✅（golden 快照锁定）
4. **R6 决策**：存量迁移**立项**（推翻默认"不立项"）——alembic_version 与 goose 版本表互通的迁移工具纳入范围，排期在 P4 收尾前独立小项目
5. **容器运行时**：本机 Docker 不可用，**改用 Podman**（podman compose 委托，OPENAI_BASE_URL 指向宿主 stub `host.containers.internal`）✅

> 已决：CLI 二进制名 `memgo`（新增第三实现，避开存量 `mem0` 命令；上游双 CLI 不动）。

---

## 10. 一页验收清单（合并前逐项勾选）

- [ ] `go build ./...` && `golangci-lint run` 通过
- [ ] 黑盒契约套件对 Go 实现全绿（goldens 与 Python 基线一致或差异均有记录）
- [ ] 鉴权矩阵 doc-02 §8 逐格有断言（含空库 bootstrap 哨兵）
- [ ] PUT 部分更新三用例通过（只传 metadata / null 清除 / text null）
- [ ] GET /memories 双模式快照分别通过
- [ ] /configure 深合并 + redact + provider 白名单通过
- [ ] refresh 一次性并发重放测试通过
- [ ] /search 顶层 deprecated id 仍工作
- [ ] prompts 与上游 diff = 0
- [ ] dashboard 全流程冒烟通过
- [ ] CLI parity golden 通过；未迁移命令逐条列明；mem0 仓库 cli/ 目录零改动
- [ ] 双库拓扑核查：server 表仅在 mem0_app；向量仅经 SDK pgvector 连接
- [ ] 无密钥/POSTGRES_PASSWORD 真值入库；改动保持未提交可 review
