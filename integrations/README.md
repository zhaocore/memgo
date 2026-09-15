# MemGo 接入集成

MemGo 各 AI 编码工具接入的实现,直连 MemGo 自托管 server,跨会话记忆贯穿编码工作流。

## 接入清单

| 接入 | 目录 | 形态 |
|------|------|------|
| Claude Code (P0) | [claude-code-plugin/](claude-code-plugin/) | Python 插件( hooks + skills + 后台 worker ) |
| Codex (P1) | [codex/](codex/) | hooks + 本地 MCP + 生命周期脚本 |
| DeepSeek Harness (P2) | [deepseek-plugin/](deepseek-plugin/) | TypeScript Cordis 插件 |
| 共享 MCP server | [mcp/](mcp/) | 单 stdio MCP, 供 Codex/Cursor 等复用 |

## OSS API 契约(接入统一遵守)

所有接入调用 MemGo server(Go 重写版,OSS 面)。三者共享同一套端点、鉴权与范围语义。

### 常量

| 项 | 值 |
|----|----|
| 默认 Base URL | `https://memgo.wxget.com`(env `MEMGO_BASE_URL` 覆盖) |
| API Key env | `MEMGO_API_KEY` |
| 鉴权头 | `X-API-Key: <key>`(不是 `Authorization: Token`) |
| 遥测 | 本地 spool, 不对外发送; `MEMGO_TELEMETRY=false` 关闭 |

### 实体范围

OSS 面只有三个实体维度:**user_id(人)、agent_id(项目/仓库)、run_id(会话)**。无 app_id、
无 custom_categories、无异步事件轮询。

| 记忆类型 | user_id | agent_id | run_id |
|----------|---------|----------|--------|
| 项目共享记忆 | — | 仓库身份 | — |
| 个人偏好 | 本机用户 | 仓库身份 | — |
| 会话记录 | 本机用户 | 仓库身份 | 会话 id |

### 端点

```
POST   {base}/memories                # 写入 (同步)
       body: {messages:[{role,content}], user_id?, agent_id?, run_id?, metadata?, infer?}
       响应: {"results":[{id,memory,user_id,agent_id,run_id,metadata,created_at,updated_at}]}
       memory 字段 = 提取后的记忆文本

POST   {base}/search                  # 搜索 (同步)
       body: {query, filters:{user_id?, agent_id?, run_id?}, top_k?}
       filters 至少含一个 user_id/agent_id/run_id
       响应: {"results":[{...同上}]}

GET    {base}/memories?user_id=&agent_id=&run_id=      # 列表
GET    {base}/memories/{id}                            # 取单个
PUT    {base}/memories/{id}                            # 更新 {text?, metadata?, expiration_date?}
DELETE {base}/memories/{id}                            # 删除单个
DELETE {base}/memories?user_id=&agent_id=&run_id=      # 批量删除
```

### 与托管平台 API 的差异(迁移时注意)

1. **同步写入**: 平台 add 异步返回 event_id 需轮询;OSS 直接返回 `results`。删掉事件轮询与
   `semantic-succeeded`/`explicitly-stored` 状态机,写入即完成。
2. **无 custom_categories**: 平台按后端分类 tag;OSS 不做。删掉 category 过滤与分类提示词。
3. **无 app_id**: 仓库范围改用 `agent_id` 表达(见上表)。
4. **鉴权头**: 平台用 `Authorization: Token`,OSS 用 `X-API-Key`。
5. **遥测**: memgo server 已决定"网络发送省略"(仅本地状态文件),接入端同样只落本地 spool,
   删掉 PostHog 发送与 `/v1/ping/` 邮箱解析。

## 环境变量(统一)

| 变量 | 默认 | 说明 |
|------|------|------|
| `MEMGO_BASE_URL` | `https://memgo.wxget.com` | server 根地址 |
| `MEMGO_API_KEY` | — | 鉴权 key(必填) |
| `MEMGO_USER_ID` | 本机用户名 | 默认个人范围 |
| `MEMGO_TELEMETRY` | true | 遥测开关 |
