# MemGo for Claude Code

面向 Claude Code 的持久跨会话记忆,外加一个用于委派工作的 Sonnet sidekick agent。

Claude Code 在会话之间会遗忘一切。本插件解决这个问题:hooks 在本地记录会话
细节,后台 worker 把会话内容同步写入 MemGo server 的 OSS 面,由 server 抽取生成
记忆;后续会话开始时,Claude 自动取回相关记忆。

API 层指向 MemGo 自托管
server(OSS 契约):同步写入、`X-API-Key` 鉴权、无事件轮询、无 category。

## 前置条件

- Python 3.10+ 与 Git。
- 支持 plugin agents、worktree 隔离以及 `SubagentStart`、`SubagentStop`、
  `PostToolUseFailure` hook 事件的 Claude Code 版本。
- MemGo server 的 API key(前缀 `mgsk_`),以及 server 地址
  `https://memgo.wxget.com`(可用 `MEMGO_BASE_URL` 覆盖)。

## 安装

```bash
export MEMGO_API_KEY='your-memgo-api-key'
claude plugin install memgo@memgo-plugins --scope user --config MEMGO_API_KEY="$MEMGO_API_KEY"
unset MEMGO_API_KEY
```

重启 Claude Code(或运行 `/reload-plugins`),打开一个 Git 仓库正常工作即可。

本地开发时,直接加载当前目录:

```bash
claude --plugin-dir .
```

## 工作原理

### 记忆

1. **捕获(Capture)。** Hooks 在本地保存主 agent 的活动:用户消息、Claude 的
   回答、修改过的文件路径、简短的测试/构建结果。不调用模型、不阻塞。
   Sidekick 输出不计入主会话。

2. **刷新(Flush)。** 每完成五轮交流,一个分离的后台 worker 就把该批内容同步
   写入 MemGo server(OSS `POST /memories`)。大段交流更早刷新。结束或压缩
   会话时刷掉剩余内容;空闲五分钟自动刷新(可用 `MEMGO_CODE_IDLE_FLUSH_SECONDS`
   调整)。worker 在 Claude Code 退出后仍然存活。

3. **抽取(Extract)。** 每次 flush 用 `{messages, user_id, agent_id, run_id,
   metadata, infer}` 调用 OSS `POST /memories`,由 server 同步抽取并返回
   results。OSS 面只有三条范围 lane:

   - **项目共享记忆** = `agent_id`(仓库身份,无 `user_id`);
   - **个人记忆** = `user_id` + `agent_id`;
   - **会话记录** = 在个人 lane 上再加 `run_id`。

   插件写入时同时携带三条身份,因此 repo 范围搜索(`agent_id`)能命中仓库下
   全部记忆,mine 范围搜索(`user_id` + `agent_id`)只命中个人记忆。metadata
   保留 `{source, branch, git_sha, author, dirs}`。

4. **召回(Recall)。** 下一次会话的第一个 prompt 时,插件自动搜索并注入至多
   五条相关记忆。写入查询不调用模型。

首次搜索之后,Claude 可用 `search_memories` 工具提出具体问题,你也可以自己
运行 `/memgo:search`。显式搜索默认返回至多 3 条(可配到 20),上限 4,000 字符。

### Sonnet sidekick agent

`memgo:sidekick` 是一个 Sonnet 编码 agent,运行在独立 Git worktree 中。它可以用
来调查、实现、测试、调试或评审,替代主(Opus/Fable)会话做同样的工作,降低成本。

主 agent 评审其结果。修改留在 sidekick 的 worktree 中,直到主 agent 评审并
复制过去。hook 失败时 MemGo 从不阻塞 Claude Code 的正常工作。

## 使用

在 Claude Code 中正常工作即可,记忆自动捕获与召回。

```text
/memgo:search 为什么 ODS 序列化器让日期丢失时区信息?
/memgo:search 之前修过哪些解析失败? --top-k 5
/memgo:search 我更喜欢 pnpm 还是 npm? --scope mine
```

使用 sidekick:

```text
让 MemGo 的 sidekick 在它的独立 worktree 里调查并实现这个。
评审它的结果,并把修正发回同一个 sidekick。
```

默认情况下 worktree 从仓库默认分支切出。在 Claude 设置里把 `worktree.baseRef`
设为 `"head"` 可以从当前提交切出。未提交的改动不会被复制进 sidekick 的 worktree。

## 命令

| 命令 | 作用 |
| --- | --- |
| `/memgo:search` | 搜索之前会话的记忆。支持 `--top-k <n>` 与 `--scope <repo\|dir\|mine>`。 |
| `/memgo:status` | 检查配置、捕获状态、待发 flush 与 API key 有效性。 |
| `/memgo:forget` | 删除你在这个仓库的记忆(项目共享记忆保留,除非加 `--include-project-memory`)。 |
| `/memgo:pause` | 暂停记忆捕获。 |
| `/memgo:resume` | 暂停后恢复捕获。 |
| `/memgo:remember` | 让 Claude 在回复中明确复述某条要记住的事实。 |

## 搜索范围

| 范围 | 结果 |
| --- | --- |
| `repo`(默认) | 该仓库的共享记忆加你的偏好(按 `agent_id` 过滤) |
| `dir` | 与 `repo` 相同:OSS 无目录过滤,目录维度记录在 `metadata.dirs` |
| `mine` | 仅你的个人偏好(按 `user_id` + `agent_id` 过滤) |

用 `search_scope` 设置或 `MEMGO_CODE_SEARCH_SCOPE` 修改默认范围。传
`--run-id <session-id>` 只看某个会话记录的内容。

## 设置与环境变量

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `MEMGO_BASE_URL` | `https://memgo.wxget.com` | server 根地址 |
| `MEMGO_API_KEY` | 必填 | OSS 面鉴权 key |
| `user_id`(设置) | 本机账户名 | 记忆的用户 ID。解析顺序:设置、`MEMGO_CODE_USER_ID`、`MEMGO_USER_ID`、`$USER`、`%USERNAME%`,最后 `default`。跨机器共享时显式设置。 |
| `MEMGO_PROJECT_ID` | — | 覆盖仓库身份(agent_id) |
| `top_k`(设置) | `3` | 每次显式搜索的记忆条数上限(1 到 20) |
| `max_context_chars`(设置) | `4000` | 每次搜索返回的字符数上限(1,000 到 10,000) |
| `search_scope`(设置) | `repo` | 默认范围:`repo`、`dir` 或 `mine` |
| `MEMGO_TELEMETRY` | `true` | 遥测开关,`false` 关闭 |

## 本地存储与发送内容

本地数据位于 `${CLAUDE_PLUGIN_DATA}`:

- `api-key`:配置的 MemGo key(仅本机用户可读)
- `evidence.sqlite3`:会话细节与记忆创建/搜索记录
- `pending/`:等待发送到 server 的会话(被打断后重试)
- `flush-worker.log`:记忆创建是否成功
- `plugin-errors.log`:hook 错误(不含凭据)
- `telemetry.jsonl` / `telemetry-identity.json`:本地匿名使用事件

发送到 server 的内容:每段用户消息、Claude 的回答、sidekick 的回答和修改过的
文件路径。完整文件与一般工具输出留在本机。形似凭据的值在发送前会被脱敏。

## 遥测

使用事件(哪个 hook 跑了、耗时、结果条数、失败类型)只落在本地 JSONL spool,
**不对外发送**。仓库与会话 ID 在落盘前哈希。prompt、记忆文本、文件路径、
工具输出与 API key 永不写入遥测。

关闭:

```bash
export MEMGO_TELEMETRY=false
```

## 五分钟记忆测试

安装后在一个 Git 仓库中运行:

1. 告诉 Claude:

   ```text
   记住,后续工作要用到本仓库的验收标记 cobalt-orchid-731。
   ```

2. 结束会话。在同一仓库开启新会话,运行:

   ```text
   /memgo:search 验收标记是什么?
   ```

3. 确认结果包含 `cobalt-orchid-731`。

记忆创建在后台同步完成。若首次搜索为空,稍等片刻再试。

## 故障排查

| 问题 | 解决 |
| --- | --- |
| 缺少 key | 在变量已设置时重新安装:`claude plugin install ... --config MEMGO_API_KEY="$MEMGO_API_KEY"`。 |
| `401 Unauthorized` | API key 无效或过期。运行 `/memgo:status`。 |
| 结束会话后没有记忆 | 抽取同步完成;若仍无结果,运行 `/memgo:status` 查看 doctor 检查。 |
| Sidekick 无法启动 | 需在 Git 仓库中,且 Claude Code 版本支持 plugin agents 与 worktrees。 |
| 卸载插件 | `claude plugin uninstall memgo@memgo-plugins` |

## 开发检查

从 `integrations/claude-code-plugin/` 运行:

```bash
python3 -m py_compile core/*.py adapters/claude/hook.py core/mcp_server.py
claude plugin validate --strict .
```
