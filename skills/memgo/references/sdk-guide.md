# MemGo Go 客户端指南

两条路接入:直连 OSS HTTP(任意语言通用),或内嵌 `core/memory` Go 库(进程内记忆引擎)。
MemGo 无官方 Python/TS SDK —— 它们用 HTTP 直连,见 [quickstart](quickstart.md)。

## 方式 A:直连 OSS HTTP

完整端点与响应见 [api-reference](api-reference.md)。核心四点:

1. 请求头 `X-API-Key: <key>`,Base URL 默认 `https://memgo.wxget.com`。
2. 至少一个实体维度(`user_id` / `agent_id` / `run_id`)圈定范围。
3. 写入与搜索都是**同步**的,响应直接返回结果。
4. 记忆对象字段:`id` / `memory` / `user_id` / `agent_id` / `run_id` / `metadata` /
   `created_at` / `updated_at` / `hash` / `expiration_date`(搜索再带 `score`)。

```bash
# 写入
curl -X POST $MEMGO_BASE_URL/memories -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"用户喜欢暗色模式"}],"user_id":"alice"}'

# 搜索(带过滤与阈值)
curl -X POST $MEMGO_BASE_URL/search -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query":"界面偏好","filters":{"AND":[{"user_id":"alice"},{"run_id":"*"}]},"top_k":5,"threshold":0.3}'

# 更新 / 删除
curl -X PUT $MEMGO_BASE_URL/memories/<id> -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" -d '{"text":"新文本","metadata":{"verified":true}}'
curl -X DELETE $MEMGO_BASE_URL/memories/<id> -H "X-API-Key: $MEMGO_API_KEY"
curl -X DELETE "$MEMGO_BASE_URL/memories?user_id=alice&run_id=session_1" \
  -H "X-API-Key: $MEMGO_API_KEY"
```

## 方式 B:内嵌 core/memory Go 库

模块路径 `github.com/zhao-core/memgo`。记忆引擎在 `core/memory` 包,无 HTTP 依赖;
LLM / embedder / 向量库 / 历史库都是端口,由调用方注入。

### 依赖注入

```go
m := memory.New(cfg, vecStore, embedder, llm, historyDB, entityExtractor)
```

| 参数 | 类型 | 说明 |
|------|------|------|
| `cfg` | `*config.MemoryConfig` | 记忆配置(含 vector store / LLM / embedder provider 与 prompt) |
| `vecStore` | `vectorstore.VectorStore` | pgvector 等向量库实现 |
| `embedder` | `embedder.Embedder` | OpenAI / Gemini 嵌入 |
| `llm` | `llm.LLM` | OpenAI / Anthropic / Gemini |
| `historyDB` | `*history.Manager` | SQLite 历史库 |
| `entityExtractor` | `memory.EntityExtractor` | 实体抽取(可传 nil) |

server 侧的装配参考 `server/api/factory.go` 的 `buildVectorStore` / `buildLLM` /
`buildEmbedder`(内置集:pgvector、LLM openai/anthropic/gemini、embedder openai/gemini)。

### add() —— 写入记忆

```go
results, err := m.Add(
	[]map[string]any{
		{"role": "user", "content": "我吃素,对坚果过敏"},
		{"role": "assistant", "content": "记住了。"},
	},
	memory.AddParams{UserID: "alice"},
)
```

`AddParams`:

| 字段 | 类型 | 说明 |
|------|------|------|
| `UserID` / `AgentID` / `RunID` | string | 实体范围,至少一个非空 |
| `Metadata` | `map[string]any` | 自定义键值 |
| `ExpirationDate` | `any` | 过期时间字符串或 nil |
| `Infer` | bool | `false` 原样存储不做提取(默认 true 由调用方填) |
| `MemoryType` | string | 仅 `"procedural_memory"` 特判 |
| `Prompt` | string | 覆盖提取提示词 |

返回 `[]AddResultItem`(即 `{id, memory, event}`),失败返回明确错误。

### search() —— 检索记忆

```go
threshold := 0.3
res, err := m.Search(memory.SearchParams{
	Query:     "饮食偏好",
	TopK:      10,
	Filters:   map[string]any{"user_id": "alice"},
	Threshold: &threshold, // nil 用默认 0.1
	Explain:   false,
})
// res["results"] 为 []map[string]any,每项含 memory / score / 元数据
```

`SearchParams` 字段:`Query`、`TopK`、`Filters`、`Threshold *float64`、`Explain`、`ShowExpired`。

`Filters` 语义(与 HTTP 一致):
- 至少一个实体维度,否则报错。
- 支持 `AND` / `OR` / `NOT` 逻辑与 `eq`/`ne`/`gt`/`gte`/`lt`/`lte`/`in`/`nin`/
  `contains`/`icontains`/`*` 操作符。

### 其余方法

| 方法 | 签名要点 | 说明 |
|------|----------|------|
| `Get(memoryID)` | `(map[string]any, error)` | 不存在返回 `(nil, nil)` |
| `GetAll(GetAllParams)` | `(map[string]any, error)` | `GetAllParams{Filters, TopK, ShowExpired}`;filters 必须含实体维度 |
| `Update(UpdateParams)` | `(map[string]any, error)` | `UpdateParams{MemoryID, Text *string, Metadata, ExpirationDate, ExpirationSet}`;至少一个字段 |
| `Delete(memoryID)` | `(map[string]any, error)` | 不存在报明确错误 |
| `DeleteAll(DeleteAllParams)` | `(map[string]any, error)` | `DeleteAllParams{UserID, AgentID, RunID}`;至少一个维度 |
| `History(memoryID)` | `([]history.HistoryRow, error)` | ADD/UPDATE/DELETE 审计 |
| `Reset()` | `error` | 清空历史库与向量库 |

```go
// 更新 + 历史
newText := "新文本"
m.Update(memory.UpdateParams{MemoryID: id, Text: &newText, Metadata: map[string]any{"verified": true}})
rows, _ := m.History(id) // []history.HistoryRow{OldMemory, NewMemory, Event, ...}

// 批量删除(会话级)
m.DeleteAll(memory.DeleteAllParams{UserID: "alice", RunID: "session_1"})
```

## 常见坑

1. **实体维度必填** —— `search` / `get_all` / `delete_all` 的 filters 至少一个实体维度。
2. **同步语义** —— 写入即落库,无需轮询,`add` 后可直接 `search`。
3. **`*` 通配不含 null** —— 只匹配非 null 值。
4. **metadata 过滤只限顶层键** —— `contains` 作用于顶层键值。
5. **默认 `threshold=0.1`** —— 需要更严格匹配时调高。

## 命名对照

Go 用 PascalCase:`UserID` / `AgentID` / `RunID` / `GetAll` / `DeleteAll`(上游 Python
`user_id` / `get_all` 的 Go 化);HTTP JSON 一律 snake_case(`user_id` / `agent_id` /
`run_id` / `top_k`)。
