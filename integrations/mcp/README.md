# MemGo 共享 MCP server

单 stdio MCP server, 直连 MemGo server 的 OSS API, 供 Codex / Cursor 等客户端复用。
Claude Code 插件自带精简 search 工具, 不依赖本 server。

## 运行

```bash
export MEMGO_BASE_URL=https://memgo.wxget.com
export MEMGO_API_KEY=...
python3 memgo_mcp.py
```

## 工具

| 工具 | 说明 |
|------|------|
| `add_memory` | 同步写入消息列表, 返回提取结果 |
| `search_memories` | 语义搜索 (filters 至少一个实体维度) |
| `get_memories` | 按实体范围列记忆 |
| `get_memory` | 按 ID 取单条 |
| `update_memory` | 覆盖文本/元数据 |
| `delete_memory` | 删除单条 |
| `delete_all_memories` | 批量删除范围记忆 |

契约见 [../README.md](../README.md)。
