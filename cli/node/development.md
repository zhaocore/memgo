# Node CLI 开发

## 环境与常用命令

使用 Node.js 24、pnpm 11.21.0。`packageManager` 固定 pnpm 版本，`pnpm-lock.yaml` 固定依赖。不要混用系统中其他 pnpm，也不要更新全局工具来修复本包。

在 `cli/node/` 执行：

```bash
pnpm install --frozen-lockfile
pnpm typecheck
pnpm lint
pnpm test
pnpm build
pnpm test:terminal
pnpm pack
```

真实终端验收另需 Python 3（仅使用标准库），运行 `pnpm test:terminal`。

`pnpm test` 先构建，再运行完整 Vitest；类型检查同时覆盖源码、测试和构建配置。`pnpm lint` 先运行 ESLint（零警告），再运行 Biome；检查源码、测试、构建配置和维护脚本。开发调试用 `pnpm dev --help`；不要额外插入 `--`。

## 文件职责

| 路径 | 职责 |
| --- | --- |
| `src/index.ts`、`src/cli/run.ts` | 进程装配、单次调用上下文和退出码 |
| `src/cli/program.ts`、`src/cli/registrations/` | 创建命令树、注册选项和参数映射 |
| `src/cli/commands/` | 命令执行及结果展示；memory 按操作拆分 |
| `src/application/` | 实体选择、输入校验和初始化用例 |
| `src/backend/` | Backend 端口、Platform/OSS/认证连接器、HTTP 错误与 JSON 校验 |
| `src/config/` | 内存模型、默认值、纯解析/更新与文件持久化 |
| `src/output/` | 品牌、文本、表格和 JSON 输出 |
| `src/runtime/` | 终端交互、单次调用状态、结束信号和资源定位 |
| `src/integrations/` | 代理环境检测、插件同步、遥测发送装配 |
| `telemetry-sender.cjs` | 独立的 CommonJS 遥测进程，不依赖 CLI 状态 |
| `tests/unit/`、`tests/integration/` | 单元测试、真实本地 HTTP 与 CLI 集成测试 |
| `tests/package/` | 安装包跨版本冒烟及真实伪终端验收 |

依赖方向：入口装配具体依赖，注册层调用命令/用例，用例依赖窄 backend 接口。Backend 不导入 CLI 状态或输出模块；平台 caller 类型和 notice 接收器由调用层注入。简单查询直接使用接口，不添加纯转发层。

终端、文件和网络是副作用边界；业务计算返回新值，不修改输入。调用状态通过 AsyncLocalStorage 按执行隔离，入口用 `withInvocation` 装配；不可直接在缺少上下文时调用终端格式化器。

## 添加功能

1. 在需求真源记录可观察行为、兼容性及验收标准。
2. 在对应领域增加必要的请求/结果类型及窄接口；校验外部必需字段，开放 metadata 使用递归 JSON 类型。
3. 在对应目录增加单一职责命令或用例；在 `registrations/` 声明参数，再在 `program.ts` 注册。不得复制其他命令的网络实现。
4. 增加最少的行为测试；协议变更优先使用 `tests/helpers/http-server.ts` 的真实本地服务。通过安装包入口检查输出、退出码和配置隔离。
5. 执行完整检查，更新当日计划。未经用户要求不创建提交或发布。

注释与 JSDoc 使用中文；保留代码标识符、协议字段和工具指令。参数显式提供，不设默认参数。错误通过明确异常或 `CliExit` 传回入口，命令内部不得调用 `process.exit()`。

## ESM 导入与注释规范

相对导入保留 `.js`，例如 `import { resolveIds } from "../application/entity-ids.js"`。TypeScript 在类型检查时将该路径解析到对应 `.ts` 源码；tsup 负责构建 ESM 产物，Node 运行 JavaScript 文件。当前 `moduleResolution: "bundler"` 也允许省略扩展名，因此 `.js` 是本包统一约定，并非该配置的硬性要求。Node 原生 ESM 的相对路径需要明确扩展名；包名和 `node:` 内置模块不添加 `.js`。

ESLint 使用 `eslint.config.mjs` 的 flat config，启用 JS、TypeScript 推荐规则与 `eslint-plugin-jsdoc`。Biome 保留现有格式、导入排序和基础检查。CI 已执行 `pnpm lint`，因此自动包含 ESLint；开发依赖仅在 Node 24 安装运行，发布包仍验证 Node 18/20/22/24。

- 函数声明、连接器方法及构造函数必须有中文 JSDoc；测试不强制每个辅助函数新增文档，但已有 JSDoc 同样校验。
- 使用 `/** ... */`，多行内容带 `*`；每个标签独占一行。摘要与参数、返回值说明必须包含中文，并人工检查语义准确性。
- `@param` 名称与函数参数一致，使用 `-` 分隔中文说明；需要记录对象参数的字段。返回数据的函数使用 `@returns` 描述结果，`void` 函数不堆砌无意义的返回说明。
- TypeScript 的参数及返回类型以签名为准，JSDoc 不重复声明类型；独立 JavaScript 脚本通过 JSDoc 提供类型。必要时使用 `@throws` 说明明确的异常条件。
- 保留接口签名但确实不用的参数以 `_` 开头；ESLint 会拒绝仍被使用的下划线参数。未使用变量和导入必须清除。
- `pnpm lint:fix` 修复可自动处理的问题；业务说明需人工补全，不能依靠自动修复生成空标签后交付。

函数声明中的注释示例：

```ts
/**
 * 隐藏密钥内容，仅保留识别所需片段。
 * @param key - 需要展示的 API 密钥。
 * @returns 脱敏后的密钥文本。
 */
export declare function redactKey(key: string): string;
```


## 验收边界

CI 在 Node 24 安装、检查和构建，将 tarball 安装到临时项目，再分别用 Node 18、20、22、24 执行包级冒烟。开发依赖不承诺在 Node 18 运行。

本地安装包验收：

```bash
node tests/package/smoke.mjs /absolute/path/to/installed/package/dist/index.js
```

脚本在临时目录启动本地 HTTP 服务，验证命令、配置权限、读写、插件同步和遥测发送器，不使用真实凭据。普通 Vitest 中的 stdin 用例使用真实 POSIX 管道；Node 的 `spawn` pipe 在 macOS 可能表现为 socket，不能用它假冒 FIFO。

真实平台账户、邮箱送达和真实代理客户端需要单独验收。本地服务及接口桩结果不代表这些外部系统已通过。服务端及 Go 验证遵守根目录规范，详见 [需求](../../docs/REQUIREMENTS.md) 与 [重写计划](../../docs/node-cli-rewrite-plan.md)。
