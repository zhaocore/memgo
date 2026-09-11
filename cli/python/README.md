# MemGo Python CLI

面向终端和 AI 代理的 MemGo Platform 客户端。支持 Python 3.10+，提供 `memgo` 和 `python -m memgo_cli` 两个入口。

## 安装与开始使用

在虚拟环境安装已发布版本：

```bash
python3 -m venv .venv
.venv/bin/python -m pip install memgo-cli
.venv/bin/memgo --help
```

源码开发及安装本地构建包见 [开发指南](development.md)。当前源码改造尚未发布，PyPI 安装得到的是已发布版本。

```bash
# 交互输入密钥或通过邮箱登录
memgo init
memgo init --email alice@example.com

# 添加、检索、更新记忆
memgo add "我偏好中文回答" --user-id alice
memgo search "回答偏好" --user-id alice
memgo list --user-id alice
memgo get <memory-id>
memgo update <memory-id> "我偏好简洁的中文回答"
memgo delete <memory-id>
```

当前只实现 Platform 协议。`MEMGO_BASE_URL` 可指定兼容 Platform 的地址；指向自托管 Go REST 服务不会自动切换协议。

## 命令与帮助

| 命令 | 用途 |
| --- | --- |
| `init` | 初始化、邮箱验证码登录、代理账号创建或认领 |
| `add` / `search` / `get` / `list` / `update` / `delete` | 记忆读写、检索和范围删除 |
| `entity list` / `entity delete` | 实体查询和级联删除 |
| `event list` / `event status` | 后台处理事件查询 |
| `config show` / `config get` / `config set` | 脱敏查看和更新配置 |
| `identify` / `whoami` | 声明代理名称、查询默认用户 ID |
| `agent-rush add` / `agent-rush search` | 提交和检索公开游戏记忆 |
| `import` | 从 JSON 文件导入记忆 |
| `status` / `version` / `help` | 连接状态、版本和命令说明 |

完整选项以 `memgo <command> --help` 为准；帮助和终端文案保留英文。`memgo help --json` 输出机器可读命令描述。

## 输入与输出

```bash
# 管道文本与结构化消息
printf '我使用 vim\n' | memgo add --user-id alice
memgo add --messages '[{"role":"user","content":"我使用 vim"}]' -u alice
memgo add --file messages.json -u alice

# 元数据必须为 JSON 对象
memgo update <memory-id> --metadata '{"source":"terminal"}'

# 普通 JSON 输出与代理信封
memgo search "编辑器" -u alice --output json
memgo --json search "编辑器" -u alice
```

`--output` 支持 `text`、`json`、`quiet`。全局 `--json` 或 `--agent` 使用包含 `status`、`command`、`data` 的代理信封；错误包含 `error`，失败返回非零退出码。代理模式去掉品牌和进度输出，平台通知放入信封。`init --agent` 专门表示创建或复用代理账号；需要 JSON 输出时另加 `--json`。

导入前先校验全部文件记录；请求失败时汇总成功/失败数量并返回 1，已成功的导入不会自动撤回。未知配置键和无效配置值同样返回 1。

异步添加可能返回 `PENDING` 和事件 ID；用 `memgo event status <event-id>` 查询结果。同一事件的重复待处理项合并展示。

## 配置与身份

配置位于 `~/.memgo/config.json`，目录权限为 `0700`，文件为 `0600`。配置读取顺序：命令显式选项 > 环境变量 > 配置文件 > 内置默认值。

| 环境变量 | 用途 |
| --- | --- |
| `MEMGO_API_KEY` | Platform 密钥 |
| `MEMGO_BASE_URL` | Platform API 地址 |
| `MEMGO_USER_ID` / `MEMGO_AGENT_ID` / `MEMGO_APP_ID` / `MEMGO_RUN_ID` | 默认实体范围 |
| `MEMGO_TELEMETRY=false` | 关闭可选遥测 |
| `NO_COLOR` | 关闭终端彩色符号 |

提供任意显式实体 ID 时，只使用显式范围，不混入其他实体默认值。`config show/get` 对密钥脱敏。保存配置时，只同步已经存在的 Claude `env.MEMGO_API_KEY` 和 shell 导出条目，不创建新的集成配置。

```bash
memgo config set defaults.user_id alice
memgo config get defaults.user_id
memgo config show
```

初始化失败不保存新密钥；已有配置需要确认或 `--force` 才覆盖。代理初始化优先复用有效环境变量密钥，其次复用配置密钥；瞬时网络故障不会触发新账号创建。

```bash
memgo init --agent --agent-caller my-agent
memgo identify my-agent
memgo whoami
memgo init --email alice@example.com --code <verification-code>
```

## 删除与公开记忆

```bash
memgo delete <memory-id> --dry-run
memgo delete --all --user-id alice --force
memgo entity delete --user-id alice --force
```

代理模式删除必须显式提供 `--force`。`delete --all --project --dry-run` 明确失败且不删除，因为平台没有项目级预览接口。

AGENTRUSH 记忆对其他玩家公开。首次交互添加需要确认公开提示；非交互调用向标准错误提示后继续，平台负责内容限制和配额。
