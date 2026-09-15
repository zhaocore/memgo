# MemGo 集成模式

MemGo 接入 AI 应用的四种形态,按场景选择。全部直连自托管 server 的 OSS API。

## 共同模式

无论哪种接入,记忆 loop 都一样:

1. **Retrieve** —— 生成响应前搜索相关记忆
2. **Generate** —— 把记忆作为上下文放进 LLM prompt
3. **Store** —— 把交互写回记忆,供下次使用

OSS 面三点硬约束:请求头 `X-API-Key`、至少一个实体维度(`user_id` / `agent_id` /
`run_id`)、同步写入。

## MCP server(通用 AI 客户端)

共享 stdio MCP server `integrations/mcp/memgo_mcp.py`,Claude Code / Codex / Cursor /
Windsurf 可复用。工具面:add / search / get / update / delete(详见
[features.md](features.md) 的 MCP 表)。

```bash
export MEMGO_BASE_URL=https://memgo.wxget.com
export MEMGO_API_KEY=...
python3 integrations/mcp/memgo_mcp.py   # stdio MCP
```

客户端注册成 MCP 后,agent 自主决定何时存 / 取记忆,无需手动调 API。

## CLI(memgo)

终端 / 脚本 / CI 操作记忆:

```bash
go build -o bin/memgo ./cli/go/cmd/memgo
bin/memgo init --api-key "$MEMGO_API_KEY" --user-id alice
bin/memgo add "用户偏好暗色模式" -u alice
bin/memgo search "界面偏好" -u alice -o table
bin/memgo list -u alice
bin/memgo update <id> "新文本"
bin/memgo delete <id>
```

命令面与上游 CLI 三向 parity(golden:`tests/contract/cli_parity_golden.json`)。
打自托管 server 时设 `MEMGO_BASE_URL`。

## 编码 agent hooks(Claude Code / Codex / DeepSeek)

跨会话记忆贯穿编码工作流,仓库范围用 `agent_id` 表达:

### Claude Code(integrations/claude-code-plugin)

Python 插件:hooks + skills(remember / search / forget / status / pause / resume)+
后台 flush worker。会话生命周期内自动捕获、压缩前存摘要、`Stop` 时持久化。

### Codex(integrations/codex)

hooks + 本地 MCP + 生命周期脚本。安装:

```bash
export MEMGO_API_KEY=...
python3 integrations/codex/scripts/install_codex_hooks.py
# ~/.codex/config.toml 里开启 codex_hooks = true,重启 Codex
```

关键 hooks:`SessionStart`(加载仓库记忆)、`UserPromptSubmit`(预取相关记忆注入)、
`PreToolUse`(拦截 MEMORY.md 写、给 MCP 工具注入 user_id/agent_id)、`Stop`(会话摘要持久化)、
`PreCompact`(压缩前保存上下文)。hooks 脚本通过 `scripts/_api.py` 直连 OSS 端点。

### DeepSeek(integrations/deepseek-plugin)

TypeScript Cordis 插件,`src/client.ts` 直连 OSS HTTP,带 scoping / 格式化 / 遥测。

## HTTP 直连(任意语言)

最通用的一条路。curl / Python / Node / 任何语言,见
[quickstart.md](quickstart.md) 的 Python / Node 无 SDK 示例与
[api-reference.md](api-reference.md) 的端点表。

```bash
# 写
curl -X POST $MEMGO_BASE_URL/memories -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"我喜欢意大利菜,下个月去罗马"}],"user_id":"alice"}'

# 取
curl -X POST $MEMGO_BASE_URL/search -H "X-API-Key: $MEMGO_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"query":"罗马旅行","filters":{"user_id":"alice"}}'
```

## 通用 retrieve→generate→store(Go 内嵌 core/memory)

```go
// chat 助手:检索 → 生成 → 存储
func chat(m *memory.Memory, llm llm.LLM, userInput, userID string) (string, error) {
	// 1. 检索相关记忆
	res, err := m.Search(memory.SearchParams{
		Query:   userInput,
		Filters: map[string]any{"user_id": userID},
	})
	if err != nil {
		return "", err
	}
	context := ""
	for _, r := range res["results"].([]map[string]any) {
		context += "- " + fmt.Sprint(r["memory"]) + "\n"
	}

	// 2. 带记忆上下文生成
	reply, err := llm.GenerateResponse([]llm.Message{
		{Role: "system", Content: "用户上下文:\n" + context},
		{Role: "user", Content: userInput},
	}, llm.GenerateOptions{})
	if err != nil {
		return "", err
	}

	// 3. 存回交互
	_, err = m.Add([]map[string]any{
		{"role": "user", "content": userInput},
		{"role": "assistant", "content": reply},
	}, memory.AddParams{UserID: userID})
	return reply, err
}
```

HTTP 等价:第 1/3 步换成 `POST /search` 与 `POST /memories`。

## 接入选择建议

| 需求 | 推荐形态 |
|------|----------|
| 任何语言的脚本 / CI | HTTP 直连 |
| Go 服务进程内嵌引擎 | core/memory 库 |
| 终端 / 手动管理记忆 | CLI(memgo) |
| 编码 agent 自动记忆 | Claude Code / Codex / DeepSeek 插件,或共享 MCP |
| 多客户端统一 | 共享 MCP server |
