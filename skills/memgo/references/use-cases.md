# MemGo 用例

基于 MemGo 真实能力的 Go / HTTP 示例,每个用例可运行。统一记忆 loop:retrieve →
generate → store。

## 目录

- [1. 个性化助手](#1-个性化助手)
- [2. 会话记忆(短生命周期)](#2-会话记忆短生命周期)
- [3. 多项目隔离(agent_id)](#3-多项目隔离agent_id)
- [4. 编码 agent 跨会话记忆](#4-编码-agent-跨会话记忆)
- [5. 过期记忆与审计](#5-过期记忆与审计)
- [通用模式](#通用模式)

---

## 1. 个性化助手

健身教练助手,跨会话记住目标与偏好。MemGo 持久化上下文,无需自管会话状态。

### HTTP 版

```bash
# 存目标
curl -X POST $MEMGO_BASE_URL/memories -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"我想 4 小时内跑完马拉松"}],"user_id":"max"}'

# 次日/重开后检索
curl -X POST $MEMGO_BASE_URL/search -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query":"训练重点","filters":{"user_id":"max"}}'
```

### Go 版(core/memory 内嵌)

```go
// 教练助手:检索记忆 → 带记忆生成 → 存回
func chat(m *memory.Memory, lang llm.LLM, userInput, userID string) (string, error) {
	res, err := m.Search(memory.SearchParams{Query: userInput, Filters: map[string]any{"user_id": userID}})
	if err != nil {
		return "", err
	}
	context := ""
	for _, r := range res["results"].([]map[string]any) {
		context += "- " + fmt.Sprint(r["memory"]) + "\n"
	}
	if context == "" {
		context = "暂无先前的上下文。"
	}
	reply, err := lang.GenerateResponse([]llm.Message{
		{Role: "system", Content: "你是健身教练 Ray,请基于以下已知事实个性化回复:\n" + context},
		{Role: "user", Content: userInput},
	}, llm.GenerateOptions{})
	if err != nil {
		return "", err
	}
	_, err = m.Add([]map[string]any{
		{"role": "user", "content": userInput},
		{"role": "assistant", "content": reply},
	}, memory.AddParams{UserID: userID})
	return reply, err
}
```

**适用**:健身教练、导师、客服 —— 任何需要跨会话记目标与偏好的助手。

---

## 2. 会话记忆(短生命周期)

用 `run_id` 圈出会话级上下文,结束后批量清理。

```bash
# 写入会话级记忆
curl -X POST $MEMGO_BASE_URL/memories -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"登录时一直报 504"}],"user_id":"alice","run_id":"ticket-9241"}'

# 会话内检索
curl -X POST $MEMGO_BASE_URL/search -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query":"故障现象","filters":{"AND":[{"user_id":"alice"},{"run_id":"ticket-9241"}]}}'

# 会话结束,清理
curl -X DELETE "$MEMGO_BASE_URL/memories?user_id=alice&run_id=ticket-9241" \
  -H "X-API-Key: $MEMGO_API_KEY"
```

Go:

```go
// 会话范围写入
m.Add(messages, memory.AddParams{UserID: "alice", RunID: "ticket-9241"})
// 会话范围检索
m.Search(memory.SearchParams{
	Query:   "故障现象",
	Filters: map[string]any{"AND": []any{
		map[string]any{"user_id": "alice"},
		map[string]any{"run_id": "ticket-9241"},
	}},
})
// 会话结束清理
m.DeleteAll(memory.DeleteAllParams{UserID: "alice", RunID: "ticket-9241"})
```

**适用**:onboarding、故障排查、支持工单 —— 短任务上下文,按需清理。

---

## 3. 多项目隔离(agent_id)

OSS 面没有 `app_id`,仓库/项目范围用 `agent_id` 表达。多个项目互不串味。

```bash
# 项目 A 的共享记忆
curl -X POST $MEMGO_BASE_URL/memories -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"该仓库约定用 buf 生成 proto 代码"}],"agent_id":"repo-alpha"}'

# 项目 A 内检索(不涉及具体用户)
curl -X POST $MEMGO_BASE_URL/search -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query":"proto 生成","filters":{"agent_id":"repo-alpha"}}'
```

Go:

```go
// 项目共享记忆(仅 agent_id,无 user_id)
m.Add(messages, memory.AddParams{AgentID: "repo-alpha"})
// 项目内检索
m.Search(memory.SearchParams{Query: "proto 生成", Filters: map[string]any{"agent_id": "repo-alpha"}})
```

组合维度:`user_id` + `agent_id` 表示"本机用户在某个仓库里的个人偏好";
`user_id` + `agent_id` + `run_id` 表示"该用户在仓库里的某次会话"。

**适用**:多仓库开发、多 agent 服务 —— 维度组合做完全隔离。

---

## 4. 编码 agent 跨会话记忆

编码 agent 的会话摘要写进 `agent_id`(仓库)范围,跨会话检索项目上下文。
仓库内已有完整接入(Claude Code 插件 / Codex hooks / MCP),见
[integration-patterns.md](integration-patterns.md);这里是手写等价:

```bash
# 会话结束时存摘要(agent 视角)
curl -X POST $MEMGO_BASE_URL/memories -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"本次会话结论:把排序逻辑抽到 core/sort,并补了基准测试"}],"agent_id":"repo-alpha","user_id":"dev1","run_id":"sess-7"}'

# 新会话启动时,预取仓库相关记忆
curl -X POST $MEMGO_BASE_URL/search -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query":"排序逻辑","filters":{"agent_id":"repo-alpha"}}'
```

**适用**:编码 agent、CI 上下文、项目知识沉淀 —— 仓库级记忆让每次会话有上下文。

---

## 5. 过期记忆与审计

敏感/时效性记忆设置过期时间;所有写操作留历史可回溯。

```bash
# 带过期时间写入(例如 24 小时后不再检索到)
curl -X POST $MEMGO_BASE_URL/memories -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"今天临时用 /tmp/token 认证"}],"user_id":"alice","expiration_date":"2025-03-13T12:00:00Z"}'

# 更新时清除过期
curl -X PUT "$MEMGO_BASE_URL/memories/<id>" -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" -d '{"expiration_date":null}'

# 审计:查看该记忆的 ADD/UPDATE/DELETE 历史
curl "$MEMGO_BASE_URL/memories/<id>/history" -H "X-API-Key: $MEMGO_API_KEY"
```

Go:

```go
m.Add(messages, memory.AddParams{UserID: "alice", ExpirationDate: "2025-03-13T12:00:00Z"})
m.Update(memory.UpdateParams{MemoryID: id, ExpirationSet: true}) // expiration_date=null 清除
rows, _ := m.History(id) // []history.HistoryRow{OldMemory, NewMemory, Event, ...}
```

**适用**:临时凭证、合规审计、带 TTL 的上下文 —— 过期自动收敛,历史可回溯。

---

## 通用模式

### 模式 1:retrieve → generate → store

```bash
# 1. 检索
curl -X POST $MEMGO_BASE_URL/search -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query":"<输入>","filters":{"user_id":"<uid>"}}'
# 2. 用结果生成响应(你的 LLM 调用)
# 3. 存回
curl -X POST $MEMGO_BASE_URL/memories -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"<输入>"},{"role":"assistant","content":"<响应>"}],"user_id":"<uid>"}'
```

### 模式 2:实体维度隔离

| 场景 | 维度 |
|------|------|
| 个人长期偏好 | `user_id` |
| 会话内上下文 | `user_id` + `run_id` |
| 项目共享知识 | `agent_id` |
| 个人在某项目的偏好 | `user_id` + `agent_id` |
| 个人在某项目的一次会话 | `user_id` + `agent_id` + `run_id` |

### 模式 3:metadata 多维过滤

```bash
# 存带 metadata 的记忆
curl -X POST $MEMGO_BASE_URL/memories -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"Q3 预算复盘"}],"user_id":"alice","metadata":{"type":"billing","priority":"high"}}'

# 按 metadata 过滤检索
curl -X POST $MEMGO_BASE_URL/search -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query":"预算","filters":{"AND":[{"user_id":"alice"},{"metadata":{"contains":"high"}}]}}'
```
