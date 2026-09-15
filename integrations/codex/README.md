# MemGo for Codex

MemGo 长期记忆接入 Codex(OpenAI CLI / Codex app)。MCP server + 生命周期 hooks,
记忆数据存 MemGo 自托管 server(OSS 面)。

## 安装

```bash
export MEMGO_API_KEY=...            # 必填
# 可选: export MEMGO_BASE_URL=https://memgo.wxget.com  (默认已是)
python3 scripts/install_codex_hooks.py
```

1. hooks 安装器把生命周期事件合并进 `~/.codex/hooks.json`(幂等, `--uninstall` 移除)。
2. MCP server 通过 `.codex-mcp.json` 注册: 把 `${CODEX_PLUGIN_ROOT}` 换成
   本目录绝对路径, 或复制该文件内容进 `~/.codex/mcp.json`。server 为共享的
   [../mcp/memgo_mcp.py](../mcp/memgo_mcp.py) (本地 stdio, 直连 OSS API)。
3. 在 `~/.codex/config.toml` 开启 hooks feature flag:

   ```toml
   [features]
   codex_hooks = true
   ```

4. 重启 Codex。

## Hooks

| 事件 | 作用 |
|------|------|
| `SessionStart` | 启动时加载仓库记忆, 展示记忆数与最近时间线 |
| `UserPromptSubmit` | 预取与当前提示相关的记忆注入上下文 |
| `PreToolUse` ×3 | 拦截写 MEMORY.md; 给 memgo MCP 工具注入 user_id/agent_id 默认值; 读文件前注入相关记忆 |
| `PostToolUse` ×2 | 统计 memgo 工具使用; 扫描 Bash 输出错误提示查记忆 |
| `Stop` | 结束会话时持久化会话摘要 (infer 提取) |
| `PreCompact` | 压缩前保存上下文摘要 |

## 数据读写

hooks 脚本通过 `scripts/_api.py` 直连 OSS 端点(`POST /memories` 写入、
`POST /search` 搜索、`GET /memories` 列表), 鉴权头 `X-API-Key`, 同步返回, 无平台事件轮询。
仓库范围用 `agent_id` 表达(替代平台 app_id)。

## 环境变量

| 变量 | 默认 | 说明 |
|------|------|------|
| `MEMGO_API_KEY` | — | 鉴权 key(必填) |
| `MEMGO_BASE_URL` | `https://memgo.wxget.com` | server 根地址 |
| `MEMGO_USER_ID` | `$USER` | 默认个人范围 |
| `MEMGO_TELEMETRY` | true | 本地 spool, 不对外发送 |
| `MEMGO_PLATFORM` | — | hooks 内部标注 (codex) |
