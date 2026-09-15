# memgo-deepseek-plugin

[MemGo](https://memgo.wxget.com) 长期记忆, 作为 [DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness) (Cordis) 原生插件。

它给 Harness agent 两个记忆工具, 直连 MemGo 自托管 server 的 OSS 面(契约见 [integrations/README.md](../README.md)), 召回与写入跨会话持久:

| Tool | Does |
|---|---|
| `search_memory` | 从 MemGo 召回与查询相关的事实 |
| `add_memory` | 为后续会话存储一条事实 |

与托管平台版不同, 这里不使用托管 SDK: 插件内置一个基于 Node 原生 fetch 的 HTTP 客户端(`src/client.ts`), 指向 MemGo server 的 OSS 端点(`/memories` 与 `/search`), 同步写入、`X-API-Key` 鉴权, 无平台事件轮询。

## 工作原理

Cordis 插件是导出 `apply(ctx, config)` 的模块。它声明 `inject = ['tools']` 等待 harness 工具注册表就绪, 然后通过 `ctx.tools.register(defineTool(...))` 注册两个工具。插件卸载时工具自动移除(Cordis 可逆副作用)。

```
[ MemoryClient (src/client.ts) ]  <-- 直连 MemGo OSS server
      |
[ memgo-deepseek-plugin: apply(ctx) -> ctx.tools.register(...) ]  <-- 本包
      |
[ DeepSeek Harness ]  <-- 通过 cordis.yml 加载的 agent
```

## 本地试用

1. 构建插件:
   ```sh
   cd integrations/deepseek-plugin
   pnpm install
   pnpm build
   ```
2. 设置 MemGo key 与(可选)server 地址:
   ```sh
   export MEMGO_API_KEY=...
   export MEMGO_BASE_URL=https://memgo.wxget.com
   ```
3. 让 Harness 加载它。复制 `cordis.example.yml`, 设置到 `dist/index.js` 的绝对路径与你的 `userId`, 然后:
   ```sh
   pnpm dsh web --patch ./integrations/deepseek-plugin/cordis.example.yml
   ```
4. 打开 http://127.0.0.1:3080, 让 agent 记住一条事实, 再在后续对话中召回它。

## 配置

| Field | Required | Default | Notes |
|---|---|---|---|
| `apiKey` | no | `$MEMGO_API_KEY` | MemGo server 的鉴权 key |
| `userId` | yes | | 拥有这些记忆的实体 |
| `host` | no | `$MEMGO_BASE_URL` 或 `https://memgo.wxget.com` | MemGo OSS server 根地址 |

## 与 OSS server 的关系

- 写入 `POST {host}/memories`, body `{messages:[{role,content}], user_id, agent_id?, run_id?}`, 响应 `{results:[...]}` — 同步, 无事件轮询。
- 搜索 `POST {host}/search`, body `{query, filters:{user_id, agent_id?, run_id?}, top_k}`, 响应 `{results:[...]}`。
- 实体范围只有 user_id(人)/ agent_id(项目/仓库)/ run_id(会话) 三个维度, 无 app_id、无 custom_categories。
- 遥测本地化: 事件只追加到 `~/.memgo/deepseek-plugin-telemetry.jsonl`, 不对外发送; `MEMGO_TELEMETRY=false` 关闭。

## 状态

开发预览。跟随 DeepSeek Harness v0.1 插件 API(`@deepseek-ai/cordis`、`@deepseek-ai/dsh-tools`), 该 API 年轻且仍在演进, 稳定后锁定版本。计划中的 auto-capture(存储对话轮次而无需显式工具调用)与 auto-recall(组装 prompt 时注入记忆)待 harness session/assembly 事件 API 确认后实现。
