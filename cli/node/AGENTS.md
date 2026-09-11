# Node CLI 协作规范

本包为 `@zhaots/memgo-cli`，命令入口 `memgo`。用户已授权重写；Python/Go CLI 与服务端合同不随之修改。

- 开发使用 Node.js 24、pnpm 11.21.0；发布产物保留 Node 18+ 并验证 Node 24。
- 保持 TypeScript strict、Commander、ESM、tsup、Vitest、ESLint 和 Biome；`telemetry-sender.cjs` 是独立 CommonJS 发送器的明确例外。
- 文件职责和新增功能步骤见 [development.md](development.md)，用户操作见 [README.md](README.md)，不要重复维护同一事实。
- 注释全部中文；函数声明、连接器方法及构造函数遵守 ESLint 强制的 JSDoc 摘要、参数与返回值规范。具体规则及示例见 development.md；不修改运行文案来替代注释翻译。
- 网络实现归 `backend/`；纯规则归 `application/` 或 `config/`；进程状态归单次运行上下文，禁止跨调用可变全局业务状态。
- 不修改函数输入；文件、终端和遥测写入集中在明确副作用边界。不设默认参数，避免多模式内部函数。
- 每批改动执行 `pnpm typecheck`、`pnpm lint`、`pnpm test`、`pnpm build`；完成前执行安装包冒烟和根目录要求的完整 Go 测试。
- 测试使用临时配置和本地 HTTP 服务，不读取用户密钥、不访问真实平台或发送真实遥测。
- CI 位于根目录 `.github/workflows/ci.yml`。本次没有新增自动发布；除用户明确要求，不提交、不发布。
