# MemGo 需求与验收

本文档是 MemGo 当前需求真源。最多使用三级标题；历史实现说明仅在能解释当前可观察行为或已确认缺口时保留。

## 服务兼容性

### HTTP 合同

- 自托管 REST API 的路径、鉴权优先级、状态码、JSON 形状、错误信封、`X-Request-ID`、`WWW-Authenticate` 与 OpenAPI 必须与 `tests/contract/` 的 Python 基线及 `tests/contract/goldens/` 一致。
- `GET /memories` 两种模式的不同响应形状是合同；PUT 必须区分字段缺失、显式 `null` 与有值。仅 `metadata` 不改内容；`expiration_date: null` 清除字段；`text: null` 与空更新返回 400。
- `/configure` 必须递归 deep-merge 且递归脱敏敏感字段。内置 provider 范围固定为 LLM `openai`、`anthropic`、`gemini`；embedder `openai`、`gemini`；vector store `pgvector`。

### 数据与安全

- `memgo_app` 只存用户、API key、refresh JTI、请求日志和 settings；pgvector 库只存记忆向量。两库不得混存。
- 密钥、JWT secret、数据库密码和 provider 凭据只经环境变量或未跟踪 `.env` 提供，日志与响应必须脱敏。

## 客户端与界面

### Go CLI

- Go CLI 是 Python 与 Node CLI 外的第三实现。Python/Node CLI 按下节明确授权独立改造。Go CLI 新增命令或选项必须先进入 parity golden。
- 已确认缺口：Go CLI 未迁移 `agent-rush`、`agent-mode`、`init` 的邮箱验证与 `--agent` bootstrap、`plugin_sync`，以及 rich 色彩面板/Spinner。用户不可将这些流程视为 Go CLI 已支持功能。
- `init --email`、`--code`、`--agent` 或 `--agent-caller` 在 Go CLI 必须明确报未迁移；支持路径是 `--api-key` 与可选 `--user-id`。

### Node CLI 重写

- 大号终端 `LOGO` 展示用户指定的 `MEMGO CLI` 字样，保持品牌面板对齐。终端品牌采用青春明亮配色：亮粉主色、浅青强调色，成功、错误、警告和弱化提示分别使用薄荷绿、珊瑚红、柠檬黄和蓝灰色；色值统一由品牌模块维护。

- 用户明确要求重写 `cli/node/`：项目维护的源码、测试、脚本及配置中的自然语言注释全部使用中文；保留协议字段、代码标识符、工具指令及法律声明原文，不修改第三方依赖或生成产物。
- 整理文件布局，分离命令解析、业务用例、外部系统访问与输出；使用简洁函数和明确类型，新增功能无需修改无关命令，不引入通用插件框架或多包拆分。
- 用户说明和开发文档使用清晰中文；README 面向使用者，development.md 面向维护者，代码规则写入对应模块的中文 JSDoc，避免重复文档。
- ESLint 必须接入现有 `pnpm lint` 和 CI，使用 JS/TypeScript 推荐规则及 JSDoc 校验；函数声明、连接器方法与构造函数补全中文摘要、参数及有意义的返回值说明。TypeScript 类型保留在签名，JavaScript 脚本使用 JSDoc 类型；既有测试文档同样校验。此项仅调整开发依赖、检查入口和注释，不改变 CLI 合同；相对导入保留 `.js` 的 ESM 约定。验收包括规则正反例、完整测试与安装包冒烟；回滚只撤回本次规则、开发依赖及对应文档改动。
- Node.js 24 必须通过安装、类型检查、检查风格、完整测试、构建和安装包运行验证。“支持 Node 24”不等于仅支持 Node 24；现有 `engines.node >=18.0.0` 的最低版本调整须另行明确，不能随重构静默删除兼容性。
- 此重写是仓库“Node CLI 保持上游不变”规则的明确例外；已实施并完成 Node CLI 本地验收。既有命令、选项、默认值、配置路径和格式、环境变量优先级、stdout/stderr、退出码、Platform/OSS 协议及遥测语义作为兼容基线，不新增产品功能、不修改 Python/Go CLI 或服务端合同。
- 错误行为采用执行计划中的明确失败策略：业务鉴权预检失败终止命令；GET 网络错误或 502/503/504 最多两次、间隔 100ms，写请求不自动重试。探活每次 2400ms；普通 Platform 请求 30s，OSS 业务请求 600s。HTTP 错误携带方法、路径、状态与脱敏响应，畸形 JSON/列表/对象和无效配置字段类型明确失败。
- 密钥复用保留特殊兼容规则：仅 401/403 判定现有密钥无效，网络故障不创建替代账号；网络失败告警。插件同步和遥测启动失败只告警，不撤销已保存的主配置；独立遥测发送器失败以 stderr/非零码报告，但正常分离启动保持既有忽略子进程输出语义。批量导入保留 added/failed 聚合统计，调用方仍须检查 failed。
- 验收修正的用户可见差异：初始化、identify、whoami、AGENTRUSH 的全局 JSON 成功路径补齐信封；导入 JSON 不混入进度；未知配置字段退出码改为 1；项目级 --dry-run 在任何删除前明确拒绝。其他命令、选项、默认值、帮助描述及已有 -o json 形状保留。
- 配置保存中的插件同步和 JSON notice 使用有效 ESM 导入；初始化响应缺少有效密钥时禁止持久化。交互密钥输入必须先关闭回显、再显示提示，结束和取消时清理监听并恢复终端状态。
- Node CI 和 Makefile 测试入口统一为冻结安装与本地 pnpm 工具；pnpm 固定为 11.21.0，Node 类型使用 24，开发依赖只承诺 Node 24。增加安装包跨版本及真实伪终端验收；不新增发布流程。
- 动机是降低维护成本并验证 Node 24；兼容性目标是不改变上述用户合同。验证结果见 [实施计划](node-cli-rewrite-plan.md)，Node 18/20/22/24 安装包均需通过本地验收，服务端和真实平台验收单独标记。按阶段恢复到实施前基线可回滚，不迁移用户配置、不重采现有 golden；规划本身可独立撤回。

### Python CLI 改造

- 大号终端 `LOGO` 按最新用户反馈显示 `MEMGO LI`，使用原有块状字符风格；此项是下述终端文案保持约束的明确例外。

- 用户明确授权改造 `cli/python/`，覆盖原 Python 上游冻结约束；沿用 Node 改造的中文注释、合理布局、可扩展架构、简洁代码和清晰文档要求。Python 使用标准 docstring 与类型标注，Ruff 校验规范，不使用 JSDoc 或 ESLint。
- 保留 `memgo` 与 `python -m memgo_cli` 入口、命令/选项/默认值、英文帮助与终端文案、配置路径及环境变量优先级、Platform 请求协议。Python 3.10+ 的最低版本声明不提高；现有工厂只提供 Platform，不能把可选 SDK 依赖宣传为已经实现的 OSS CLI。
- 命令注册、用例、配置解析/存储、Backend 类型/工厂/通信、输出、终端状态与集成分别归位。业务规则优先使用函数；配置 dataclass 与外部连接器保留必要类。运行状态按调用隔离，外部输入先校验，错误明确且脱敏。
- 首先建立测试与命令树基线，再迁移和验收。增加 Ruff 格式/规范/docstring 检查与类型检查、实际 CI 和安装包冒烟。真实平台账户、邮件与代理客户端验收单独记录；不重采 frozen golden、不迁移用户配置、不自动提交或发布。
- 初始化及命令预检遇到网络、鉴权或响应校验失败时返回非零退出码；初始化失败不得保存新密钥。畸形配置、消息、元数据和导入文件明确拒绝；导入先验证全部记录再发送写请求，网络导致部分导入失败时保留成功计数、报告失败原因并返回 1。
- `config get/set` 的未知键或无效值返回 1。`delete --all --project --dry-run` 必须拒绝执行，不能忽略预览选项而删除。初始化成功的代理输出必须是 JSON 信封，不泄露密钥。
- 密钥输入先关闭终端回显再显示提示，支持编辑、中断和结束输入后的终端恢复。遥测子进程只从 stdin 接收凭据；可选集成失败记录告警，不再静默吞错。账号复用仍保留既有策略：无法确认失效时不自动创建新账号。
- Python 欢迎面板同步当前 Node 的 MEMGO CLI 标志和青春明亮配色。除上述明确修复外，命令选项、默认值、英文帮助和正常 Platform 请求协议保持基线一致；内部 Python 模块路径按新职责布局调整，不作为兼容导入 API。
- 开发和 CI 使用冻结 uv 锁文件、Ruff、中文注释检查、严格 mypy、完整 pytest 与 sdist/wheel 构建；CI 在 Python 3.10–3.14 隔离安装实际 wheel 后运行完整测试。`make cli-py-test` 使用项目虚拟环境；移除无法解析且未使用的 `[oss]` extra，不宣传不存在的 OSS 连接器；移除版本控制中的 Python 缓存产物。
- 动机为降低维护成本并修复明确失败和预览语义。验收结果见当日计划；本轮回滚只撤回 Python 产品及配套治理、检查入口改动。用户实施中已提交 `717e166` 阶段成果，后续修复保持未提交；回滚须区分两个阶段，不能撤回 Node 工作、覆盖用户配置或其他既有改动。

### Dashboard

- `dashboard/` 为已迁入的 Next.js 资产，通过 HTTP 合同使用服务端；不可将 Go 领域逻辑迁入前端。
- Dashboard favicon、PWA 图标、登录页 Logo 和公共主题 Logo 使用统一的透明底 MEMGO 品牌图标：中心记忆环连接三处神经元突触节点，单色浅灰蓝 `#94A3B8`，无附加文字、外层方框、矩形边缘或包围框。
- 已确认质量缺口：当前没有 Dashboard 单元测试或 Playwright 配置/命令。首次修改 Dashboard 产品代码时，必须在同一变更补齐两层测试与稳定选择器。
- Dashboard 全流程冒烟（setup、登录、memories、配置保存、API key 创建/撤销、requests）尚无当前轮实测证据，状态为待验证，不得标为通过。

## 已确认历史缺口与验收

### 范围裁剪

- AsyncMemory、reranker、spaCy NER/词形还原不在当前 Go 迁移范围；实体抽取使用无 spaCy 回退语义。若恢复任一能力，必须先补充合同影响、测试和回滚方案。
- 平台 backend 的真实 `whoami` 与 `search` 尚需用户提供可用平台 API key 执行；本地/契约桩验证不等同于该实测。

### 文档治理

- 历史 `docs/go-refactor-plan.md` 曾在提交 `bf93f99` 被删除，且此前没有 `docs/REQUIREMENTS.md` 与日计划。本文档、恢复后的重构计划和当日计划建立治理基线；不追溯宣称历史任务均已按该流程验收。
- 后续行为、合同、数据格式或上述缺口状态变化，必须同步更新本文档、当日计划与对应测试。

### 当前验收标准

- Go 代码改动：`go test ./...`、`go vet ./...`、`go build ./...`、`golangci-lint run`；涉及 HTTP 合同的改动还须运行 Go 契约套件。
- Dashboard 改动：补齐单元测试与 Playwright 后运行它们，并运行 `pnpm lint`、`pnpm typecheck`、`pnpm build`。
- Node CLI：完整类型检查、ESLint/JSDoc、Biome、Vitest、构建、真实终端及 Node 18/20/22/24 安装包冒烟；结果与未验证项见 Node CLI 重写计划。
