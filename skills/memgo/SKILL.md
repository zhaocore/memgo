---
name: memgo
description: >
  MemGo 记忆层接入指南 —— 为 AI 应用添加持久化记忆的自托管 Go 服务(Go server 暴露 OSS HTTP API)。
  TRIGGER when: 用户提到 "memgo", "memory layer", "persistent context",
  "remember user preferences", "长期记忆", 或需要给聊天机器人、编码 agent 或 AI 应用
  增加跨会话长期记忆。
  Covers: 自托管 Go server 的 OSS HTTP API(curl / 任意 HTTP 客户端)、
  Go core 库(github.com/zhao-core/memgo/core/memory)、CLI(memgo)、MCP、编码 agent 集成。
  DO NOT TRIGGER when: 用户只询问 CLI 命令 / 终端脚本的具体用法(用 memgo-cli)或某个接入的
  安装细节(用 memgo-integrate)。
license: Apache-2.0
metadata:
  author: zhao-core
  version: "1.0.0"
  category: ai-memory
  tags: "memory, personalization, ai, go, self-hosted, vector-search, mcp"
compatibility: 自托管 MemGo server + Go 1.26+; curl 或任意 HTTP 客户端直连 OSS API; 无官方 Python/TS SDK。
---

# MemGo 记忆层接入指南

MemGo 是自托管的记忆层,用 Go 编写。它把用户的记忆存储、检索、管理封装成一组简单的
OSS HTTP 端点,供 AI 应用使用 —— 没有托管平台的账号与依赖,自己跑一个 server 即可。

接入方式有三条,任选其一:

| 方式 | 适用场景 |
|------|----------|
| **HTTP**(curl / 任意语言) | 任何语言、脚本、CI,最简单 |
| **Go core 库**(`core/memory`) | 在 Go 进程内嵌记忆引擎,不经过网络 |
| **CLI `memgo` + MCP** | 终端操作与 AI 客户端(Claude Code / Codex / Cursor)自动接管 |

## Step 1: 启动 server

生产实例 `https://memgo.wxget.com`(env `MEMGO_BASE_URL` 可覆盖)。本地起一套:

```bash
export POSTGRES_PASSWORD=... JWT_SECRET=$(openssl rand -base64 48) OPENAI_API_KEY=sk-...
make up            # memgo-server(:8888) + pgvector(:8432)
make bootstrap     # 建 admin 与 API key
make health        # 健康三探
```

## Step 2: 鉴权与范围

所有请求带请求头 `X-API-Key: <key>`(不是 `Authorization: Token`)。

| 项 | 值 |
|----|----|
| Base URL | `https://memgo.wxget.com`(env `MEMGO_BASE_URL`) |
| API Key env | `MEMGO_API_KEY` |
| 鉴权头 | `X-API-Key: <key>` |

OSS 面只有三个实体维度:**user_id(人)、agent_id(项目/仓库)、run_id(会话)**。无 app_id、
无 custom_categories、无异步事件轮询。至少一个维度用于圈定记忆范围。

## Step 3: 核心操作

每个 MemGo 集成都遵循同一个模式:**retrieve → generate → store**(检索 → 生成 → 存储)。

```bash
# 写入(同步,立即完成)
curl -X POST $MEMGO_BASE_URL/memories \
  -H "X-API-Key: $MEMGO_API_KEY" -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"我吃素,对坚果过敏"}],"user_id":"alice"}'

# 检索
curl -X POST $MEMGO_BASE_URL/search \
  -H "X-API-Key: $MEMGO_API_KEY" -H "Content-Type: application/json" \
  -d '{"query":"饮食偏好","filters":{"user_id":"alice"}}'

# 列表 / 取单条 / 更新 / 删除
curl "$MEMGO_BASE_URL/memories?user_id=alice" -H "X-API-Key: $MEMGO_API_KEY"
curl "$MEMGO_BASE_URL/memories/<id>" -H "X-API-Key: $MEMGO_API_KEY"
curl -X PUT "$MEMGO_BASE_URL/memories/<id>" -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" -d '{"text":"更新后的记忆"}'
curl -X DELETE "$MEMGO_BASE_URL/memories/<id>" -H "X-API-Key: $MEMGO_API_KEY"
```

Go 内嵌(直接调用 core 库,不经网络):

```go
m := memory.New(cfg, vecStore, embedder, llm, historyDB, extractor) // 依赖注入

// 写入
_, _ = m.Add([]map[string]any{
    {"role": "user", "content": "我吃素,对坚果过敏"},
}, memory.AddParams{UserID: "alice"})

// 检索
res, _ := m.Search(memory.SearchParams{
    Query:   "饮食偏好",
    Filters: map[string]any{"user_id": "alice"},
})
```

## 常见边界情况

- **filters 必须含实体维度**:`search` / `get_all` 至少给一个 `user_id` / `agent_id` / `run_id`,
  否则报 400。`DELETE /memories` 同理,缺维度时 400。
- **同步写入**:写入即完成,响应直接返回 `results`,无需轮询事件。
- **无 categories / rerank**:OSS 面不做按类目分类与 rerank,删掉相关过滤参数。
- **hash 去重**:重复事实会按内容 hash 去重,不会无限堆积。

## 集成选项

- **MCP**:共享 stdio MCP server(`integrations/mcp/memgo_mcp.py`),Claude Code / Codex / Cursor
  可复用,工具面为 add/search/get/update/delete。
- **CLI**:`bin/memgo`(init / add / search / get / list / update / delete)。
- **编码 agent 插件**:Claude Code 插件、Codex hooks、DeepSeek Cordis 插件,均已内置接入。

## References

按需加载:

| Topic | File |
|-------|------|
| 快速开始(起 server + curl + Go 示例) | [references/quickstart.md](references/quickstart.md) |
| Go 客户端指南(core/memory 库 + HTTP 调用) | [references/sdk-guide.md](references/sdk-guide.md) |
| OSS HTTP API 参考(端点 / filters / 响应) | [references/api-reference.md](references/api-reference.md) |
| 架构(core 分层 / 双库拓扑 / 流水线) | [references/architecture.md](references/architecture.md) |
| 特性(同步提取 / 混合评分 / 实体 / 过滤) | [references/features.md](references/features.md) |
| 集成模式(MCP / CLI / 编码 agent hooks / HTTP 直连) | [references/integration-patterns.md](references/integration-patterns.md) |
| 用例(Go / HTTP 示例) | [references/use-cases.md](references/use-cases.md) |
