# MemGo 工程协作规范

## 开始前

每个任务开始前必须依次完成：

1. 阅读本文件、`docs/REQUIREMENTS.md`、相关的当日 `docs/PLAN_md/PLAN_yymmdd.md`、`docs/go-refactor-plan.md`，以及受影响目录的测试。
2. 执行 `git status --short`，保留所有既有未提交改动；不得重置、覆盖或顺手清理无关文件。
3. 将任务拆成按时间顺序的复选项，写入当日 `docs/PLAN_md/PLAN_yymmdd.md`；目录或文件缺失时必须创建。完成后勾选对应项。
4. 对照 `docs/REQUIREMENTS.md` 判断本次改动是否改变既有行为、HTTP 合同、CLI 行为或数据格式。冲突或含义不明确时必须向用户说明，不得自行猜测。

`docs/REQUIREMENTS.md` 是唯一的需求真源，使用最多三级标题维护。它是活文档：行为变更、验收条件澄清和用户反馈都必须同步更新。当前文件尚未建立，首个产品改动必须创建它，不能以缺失为由跳过需求维护。

## 仓库现状与地图

MemGo 是 `mem0` 自托管服务从 Python 迁移到 Go 的单模块项目，模块路径为 `github.com/zhao-core/memgo`。目标和分期以 `docs/go-refactor-plan.md` 为准；该计划不是需求真源。

| 路径 | 当前职责 |
| --- | --- |
| `cmd/server/main.go` | `memgo-server` 入口；当前是 P0 占位，P2 才实现 HTTP 服务。 |
| `cmd/memgo/main.go` | Go CLI 入口；当前是 P0 占位，P3 才实现 cobra 命令面。 |
| `core/config/` | `MemoryConfig` 解析、深合并和敏感配置脱敏。 |
| `core/llm/` | LLM 端口与 OpenAI、Anthropic 等 provider 客户端。 |
| `core/prompts/` | 上游 prompt 常量及消息拼装；`prompts_gen.go` 是生成文件。 |
| `tests/contract/` | Python 基线与 Go 实现共用的黑盒 HTTP 契约套件、golden 和 OpenAI 兼容桩。 |
| `tools/gen_prompts.py` | 从上游 `mem0/configs/prompts.py` 机械生成 Go prompt 常量。 |
| `docs/go-refactor-plan.md` | 迁移范围、目标布局、阶段验收与已知风险。 |
| `Makefile` | `build`、`test`、`vet`、`lint` 与 Podman 契约测试命令。 |

以下路径属于已确认的目标布局，但当前尚未完整落地：`core/memory/`、`core/embedder/`、`core/vectorstore/`、`core/history/`、`core/entity/`、`server/`、`cli/`、`deploy/`。新增实现必须按此职责边界落位，不得将规划路径写成既有实现。

## 后端架构规则

- `cmd/` 只负责进程装配、配置加载和退出码；不得放领域规则、HTTP handler 或 provider 协议细节。
- `core/` 是无 HTTP 依赖的记忆引擎。记忆流程、序列化兼容逻辑、配置规则和业务不变量必须放在对应 `core` 包中。
- 外部系统必须放在明确端口之后：LLM、embedder、向量库、SQLite 历史库、Postgres、遥测和平台 API 都由接口或窄客户端隔离。业务流程依赖端口，不依赖具体 SDK 或 HTTP 请求。
- `server/` 落地后只承载路由、鉴权、校验、middleware、错误映射和存储编排；`server/store` 只访问应用库，不能持有 pgvector 连接串。
- `cli/` 落地后以 Backend 接口隔离平台 API 与 OSS server；命令解析、配置状态、输出格式和后端通信分别归位，命令层不得直接拼 HTTP 请求。
- 维持单 Go module。避免为尚无独立部署、长任务重试或独立伸缩需求的功能拆分服务或 worker。
- 新增后端代码必须同时新增或更新单元测试。每次变更都必须运行完整单元测试；失败原因不明确时必须报告给用户，不能将失败静默归为既有问题。

## 前端规则

本仓库当前没有前端代码，Dashboard 属于上游资产，必须保持零改动。

引入前端时，先在需求和计划中明确入口、可复用状态/领域逻辑、DOM 绑定层和端到端边界。从引入起，每次前端代码变更都必须同时更新前端单元测试和 Playwright 测试；测试使用稳定的 `data-testid`、无障碍 ID 或等价稳定选择器，流程必须幂等或自行清理。

## 领域与兼容性红线

- 自托管 REST API 的路径、鉴权优先级、状态码、JSON 形状、错误信封、`X-Request-ID`、`WWW-Authenticate` 和 OpenAPI 都是兼容合同。以 `tests/contract/` 的 Python 基线和 `tests/contract/goldens/` 为可执行事实，不得按 Go 惯例自行改形状。
- `GET /memories` 的两种模式形状不一致也是合同；PUT 必须保留字段缺失、显式 `null` 和有值之间的语义差异。实测基线：仅 `metadata` 不改内容、`expiration_date: null` 清除字段、`text: null` 返回 400、空更新返回 400。
- `/configure` 是递归 deep-merge，不是整体替换；敏感键递归脱敏；内置 provider 范围固定为 LLM `openai`/`anthropic`/`gemini`、embedder `openai`/`gemini`、vector store `pgvector`，扩大范围必须先记录决策。
- prompt 文本是行为的一部分。修改上游 prompt 对齐时必须通过 `tools/gen_prompts.py` 生成 `core/prompts/prompts_gen.go`，并执行 `MEM0_SOURCE=<mem0仓库根目录> go test ./core/prompts`。不得手改生成文件。
- 应用库 `mem0_app` 与 pgvector 记忆库物理分离：用户、API key、refresh JTI、请求日志和 settings 不得进入向量库；记忆向量不得进入应用库。
- Go CLI 是第三个实现，不能修改上游 Python/Node CLI，也不得新增 parity golden 未定义的命令或选项；缺失功能必须显式列出。
- 外部 provider 的错误必须保留可分类原因和请求上下文；重试时记录告警，耗尽后返回最后一个明确错误，不能吞错或用回退结果掩盖故障。

## 测试与验证

每次后端代码改动必须新增或更新同层单元测试。每次前端代码改动必须新增或更新前端单元测试与 Playwright 测试。每次变更完成前必须运行完整单元测试，并按影响范围执行以下验证：

```bash
go test ./...
go vet ./...
go build ./...
golangci-lint run
```

等价 Make 命令为 `make test`、`make vet`、`make build`、`make lint`。CI 也执行 build、vet、test 和 golangci-lint；本地验证必须与其保持一致。

涉及 HTTP 合同、鉴权、配置、错误信封、OpenAPI、限流或记忆端点时，必须运行黑盒契约测试。契约栈使用 Podman：

```bash
make contract
CONTRACT_BASE_URL=http://localhost:8888 ./tests/contract/run.sh --impl go
```

该套件要求全新数据库，`make contract-up` 会删除契约栈 volume；运行前必须确认没有需要保留的测试数据。仅在已确认 Python 上游合同发生变化、需求已更新且差异已审阅时，才能执行 `make contract-capture` 更新 golden。

外部 LLM、embedder 和平台调用的单元测试使用接口桩或本仓库 contract stub；需要验证真实协议或部署边界时使用明确标注的集成测试。不得用脆弱的网络依赖替代可重复的单元测试。

## 文档、计划与反馈闭环

- 每项任务都必须在 `docs/PLAN_md/PLAN_yymmdd.md` 以时间顺序的复选项记录；同一天追加到同一文件。完成、阻塞和未验证项必须如实标记。
- `docs/REQUIREMENTS.md` 记录用户可观察的行为、接口约束、数据规则和验收标准，最多三级层级。架构说明只在无法写入代码或需求条目时补充到单独文档，不能复制同一事实。
- 每次改动都必须将结果与当前需求对照；任何歧义、合同与需求冲突、或基线和计划不一致都必须先提示用户。
- 用户测试或反馈暴露行为不匹配时，必须在同一次后续修复中同时更新 `docs/REQUIREMENTS.md` 和相关测试，使预期更精确。

## 安全、配置与禁区

- API key、JWT secret、数据库密码和 provider 凭据只能通过环境变量或本地忽略的 `.env` 提供；不得写入 Go 源码、golden、文档示例或日志。
- 配置读取与响应输出必须复用 `core/config` 的脱敏语义；日志不得输出 Authorization、`X-API-Key`、refresh token 或未脱敏配置。
- 不在 `bin/`、`.git/`、`.env`、`tests/contract/.state`、容器 volume、缓存或测试临时文件中放产品逻辑。
- `tests/contract/goldens/` 是冻结合同快照，`tests/contract/stub/` 只服务测试，`core/prompts/prompts_gen.go` 是生成产物；这些路径不能成为业务逻辑归宿。
- 现有大入口和上游兼容层必须保持行为，优先向下抽取小函数或包，不进行整段重写。

## 需显式记录的变更

以下改动必须在 `docs/REQUIREMENTS.md`、当日计划和评审说明中写明动机、兼容性影响、验证结果与回滚方式：

- HTTP 路径、鉴权、状态码、错误信封、OpenAPI 或 golden 的任何变化。
- provider 白名单、prompt 文本、序列化、时间格式、hash 或部分更新语义的变化。
- `core`、`server`、`cli` 边界，双数据库拓扑，数据库 schema 或迁移策略的变化。
- CLI 命令/选项矩阵、外部 API 协议、鉴权凭据处理或遥测行为的变化。
- CI、Makefile、契约栈、测试基线和部署策略的变化。

提交前保持改动未提交供用户审阅；只有用户明确要求时才创建 git commit。
