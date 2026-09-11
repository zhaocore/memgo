# MemGo Node CLI

`@zhaots/memgo-cli` 提供 `memgo` 命令，用于管理平台或自托管 MemGo 的记忆。发布产物支持 Node.js 18+，包含 Node.js 24；开发环境使用 Node.js 24 和 pnpm 11.21.0。

## 安装与连接

在使用方项目安装后运行，不需要全局安装：

```bash
pnpm add @zhaots/memgo-cli
pnpm exec memgo --help
pnpm exec memgo init
```

已有凭据时，通过 `MEMGO_API_KEY` 提供密钥。`MEMGO_BASE_URL` 指向 `https://api.memgo.ai` 时使用平台协议；指向其他 HTTP(S) 服务地址时使用 OSS 协议。不要把真实密钥写进脚本或提交到仓库。

```bash
pnpm exec memgo add "偏好简洁的代码" --user-id alice
pnpm exec memgo search "代码偏好" --user-id alice
pnpm exec memgo list --user-id alice
pnpm exec memgo get <memory-id>
pnpm exec memgo update <memory-id> "偏好明确的模块边界"
pnpm exec memgo delete <memory-id> --force
```

配置保存到 `~/.memgo/config.json`，权限为 `0600`；优先级为命令参数、环境变量、配置文件、默认值。`config show` 和 `config get` 对密钥脱敏。保存配置时只更新已有 Claude `MEMGO_API_KEY` 和 shell 导出，不创建新条目。

## 命令

| 命令 | 用途 |
| --- | --- |
| `init` | 交互配置、API key 配置、邮箱验证码登录或 Agent Mode 初始化 |
| `add` / `search` / `get` / `list` / `update` | 记忆读写与查询 |
| `delete` | 单条、指定范围或实体级删除；用 `--help` 查看互斥规则 |
| `import <filePath>` | 从 JSON 文件批量导入 |
| `config show/get/set` | 查看、读取和修改配置 |
| `entity list/delete` | 实体查询和删除 |
| `event list/status` | 平台后台任务状态 |
| `identify` / `whoami` | 声明及查询代理身份 |
| `agent-rush add/search` | 公开活动记忆的提交与查询 |
| `status` / `version` / `help` | 连接状态、版本和帮助 |

参数、默认值和示例以 `memgo <command> --help` 为准。OSS 不支持平台专属选项，例如 `--immutable`、`--keyword`、app 范围和 event 命令；这些请求明确报错，不静默忽略。项目级 `--dry-run` 不受支持，会在删除前报错。

## 自动化与输入

全局 `--json` 或 `--agent` 选择机器输出，成功和错误均使用 JSON。`init --agent` 是创建 Agent Mode 账号的选项，与放在子命令前的全局 `--agent` 含义不同。

```bash
pnpm exec memgo --json search "偏好" --user-id alice
pnpm exec memgo init --agent --json
pnpm exec memgo init --email alice@example.com --code 123456
pnpm exec memgo add --file messages.json --user-id alice -o json
```

`messages.json` 是含 `role` 和 `content` 的消息对象数组。普通模式下，add、search 和 update 可读取管道或文件重定向；全局代理模式不自动读取 stdin，避免无人值守流程挂起。使用子进程 API 时直接传参数数组，不拼 shell 字符串。

`--json` 使用命令信封；各命令的 `-o json` 保留既有形状，不能假定二者完全一致。诊断和进度使用 stderr。业务请求、无效输入及配置错误返回非零退出码；GET 瞬时网络错误或 502/503/504 最多尝试两次，写请求不自动重试。批量导入保留逐项统计合同，自动化调用还须检查结果中的 `failed`。

## 遥测与开发

`MEMGO_TELEMETRY=false` 禁用遥测。开启时沿用设备匿名标识、已有身份关联和独立发送子进程；凭据通过 stdin 传递。发送器打包在包内，可选同步或遥测故障不撤销已完成的业务操作。

开发步骤、目录职责和验收命令见 [development.md](development.md)。此次重写尚未发布；本地安装包的验证结果见仓库 [重写计划](../../docs/node-cli-rewrite-plan.md)。
