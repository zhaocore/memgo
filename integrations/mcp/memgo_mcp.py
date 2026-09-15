#!/usr/bin/env python3
"""MemGo 共享 stdio MCP server (OSS 面直连)。

暴露记忆管理工具, 后端为 MemGo 自托管 server 的 OSS API。
供 Codex / Cursor 等客户端复用; Claude Code 插件自带精简 search 工具。

纯 stdlib (urllib), 无第三方依赖。

配置 (env):
    MEMGO_BASE_URL    server 根地址, 默认 https://memgo.wxget.com
    MEMGO_API_KEY     鉴权 key (必填)
"""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request
from typing import Any

PROTOCOL_VERSION = "2024-11-05"
SERVER_NAME = "memgo"
SERVER_VERSION = "0.1.0"


def base_url() -> str:
    """server 根地址, 去除尾部斜杠。"""
    return os.environ.get("MEMGO_BASE_URL", "https://memgo.wxget.com").rstrip("/")


def api_key() -> str:
    """鉴权 key, 缺失时抛错。"""
    key = os.environ.get("MEMGO_API_KEY", "")
    if not key:
        raise RuntimeError("MEMGO_API_KEY 未设置")
    return key


def _request_json(
    method: str, path: str, *, payload: Any | None = None
) -> Any:
    """发起 OSS API 请求并解析 JSON 响应, 失败抛带状态的 RuntimeError。"""
    url = base_url() + path
    headers = {"X-API-Key": api_key(), "Content-Type": "application/json"}
    data = None if payload is None else json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(request, timeout=15) as response:
            raw = response.read()
    except urllib.error.HTTPError as error:
        body = error.read().decode("utf-8", "replace")
        raise RuntimeError(f"{method} {path}: HTTP {error.code}, {body[:500]}") from error
    except urllib.error.URLError as error:
        raise RuntimeError(f"{method} {path}: 网络错误 {error.reason}") from error
    return json.loads(raw or b"{}")


# ---- OSS API 封装 (契约见 integrations/README.md) ----

def add_memory(
    messages: list[dict[str, str]],
    *,
    user_id: str | None = None,
    agent_id: str | None = None,
    run_id: str | None = None,
    metadata: dict[str, Any] | None = None,
    infer: bool = True,
) -> list[dict[str, Any]]:
    """同步写入一条/多条记忆, 返回提取后的 results。"""
    payload: dict[str, Any] = {"messages": messages, "infer": infer}
    if user_id:
        payload["user_id"] = user_id
    if agent_id:
        payload["agent_id"] = agent_id
    if run_id:
        payload["run_id"] = run_id
    if metadata:
        payload["metadata"] = metadata
    response = _request_json("POST", "/memories", payload=payload)
    results = response.get("results") if isinstance(response, dict) else None
    return results if isinstance(results, list) else []


def search_memories(
    query: str,
    *,
    filters: dict[str, str] | None = None,
    top_k: int | None = None,
) -> list[dict[str, Any]]:
    """语义搜索, 返回 results 列表。"""
    payload: dict[str, Any] = {"query": query}
    if filters:
        payload["filters"] = filters
    if top_k:
        payload["top_k"] = top_k
    response = _request_json("POST", "/search", payload=payload)
    results = response.get("results") if isinstance(response, dict) else None
    return results if isinstance(results, list) else []


def list_memories(
    *,
    user_id: str | None = None,
    agent_id: str | None = None,
    run_id: str | None = None,
) -> list[dict[str, Any]]:
    """列出范围记忆 (按实体过滤)。"""
    query: list[str] = []
    for name, value in (("user_id", user_id), ("agent_id", agent_id), ("run_id", run_id)):
        if value:
            query.append(f"{name}={urllib.parse.quote(value)}")
    suffix = f"?{'&'.join(query)}" if query else ""
    response = _request_json("GET", f"/memories{suffix}")
    results = response.get("results") if isinstance(response, dict) else None
    return results if isinstance(results, list) else []


def get_memory(memory_id: str) -> dict[str, Any] | None:
    """取单条记忆, 不存在返回 None。"""
    response = _request_json("GET", f"/memories/{urllib.parse.quote(memory_id)}")
    return response if isinstance(response, dict) and response.get("id") else None


def update_memory(
    memory_id: str, *, text: str | None = None, metadata: dict[str, Any] | None = None
) -> dict[str, Any]:
    """更新记忆文本/元数据。"""
    payload: dict[str, Any] = {}
    if text is not None:
        payload["text"] = text
    if metadata is not None:
        payload["metadata"] = metadata
    return _request_json("PUT", f"/memories/{urllib.parse.quote(memory_id)}", payload=payload)


def delete_memory(memory_id: str) -> dict[str, Any]:
    """删除单条记忆。"""
    return _request_json("DELETE", f"/memories/{urllib.parse.quote(memory_id)}")


def delete_all_memories(
    *, user_id: str | None = None, agent_id: str | None = None, run_id: str | None = None
) -> dict[str, Any]:
    """批量删除范围记忆。"""
    query: list[str] = []
    for name, value in (("user_id", user_id), ("agent_id", agent_id), ("run_id", run_id)):
        if value:
            query.append(f"{name}={urllib.parse.quote(value)}")
    suffix = f"?{'&'.join(query)}" if query else ""
    return _request_json("DELETE", f"/memories{suffix}")


# ---- MCP 工具 ----

TOOLS: list[dict[str, Any]] = [
    {
        "name": "add_memory",
        "description": "保存文本或对话历史到 MemGo, 供后续会话检索。写入为同步提取, 返回每条记忆。",
        "inputSchema": {
            "type": "object",
            "properties": {
                "messages": {
                    "type": "array",
                    "items": {
                        "type": "object",
                        "properties": {
                            "role": {"type": "string"},
                            "content": {"type": "string"},
                        },
                        "required": ["role", "content"],
                    },
                    "description": "消息列表, 通常 [{role:'user', content:...}]",
                },
                "user_id": {"type": "string"},
                "agent_id": {"type": "string"},
                "run_id": {"type": "string"},
                "metadata": {"type": "object"},
                "infer": {"type": "boolean", "default": True},
            },
            "required": ["messages"],
        },
    },
    {
        "name": "search_memories",
        "description": "语义搜索记忆。回答可能依赖先前上下文的问题前务必先调用。filters 至少含一个 user_id/agent_id/run_id。",
        "inputSchema": {
            "type": "object",
            "properties": {
                "query": {"type": "string", "description": "要回忆什么"},
                "filters": {
                    "type": "object",
                    "properties": {
                        "user_id": {"type": "string"},
                        "agent_id": {"type": "string"},
                        "run_id": {"type": "string"},
                    },
                },
                "top_k": {"type": "integer", "minimum": 1, "maximum": 100},
            },
            "required": ["query"],
        },
    },
    {
        "name": "get_memories",
        "description": "按实体范围列出记忆 (过滤分页)。",
        "inputSchema": {
            "type": "object",
            "properties": {
                "user_id": {"type": "string"},
                "agent_id": {"type": "string"},
                "run_id": {"type": "string"},
                "limit": {"type": "integer"},
            },
        },
    },
    {
        "name": "get_memory",
        "description": "按 ID 取单条记忆。",
        "inputSchema": {
            "type": "object",
            "properties": {"memory_id": {"type": "string"}},
            "required": ["memory_id"],
        },
    },
    {
        "name": "update_memory",
        "description": "按 ID 覆盖记忆文本或元数据。",
        "inputSchema": {
            "type": "object",
            "properties": {
                "memory_id": {"type": "string"},
                "text": {"type": "string"},
                "metadata": {"type": "object"},
            },
            "required": ["memory_id"],
        },
    },
    {
        "name": "delete_memory",
        "description": "按 ID 删除单条记忆。",
        "inputSchema": {
            "type": "object",
            "properties": {"memory_id": {"type": "string"}},
            "required": ["memory_id"],
        },
    },
    {
        "name": "delete_all_memories",
        "description": "批量删除范围记忆, 至少提供一个实体过滤。",
        "inputSchema": {
            "type": "object",
            "properties": {
                "user_id": {"type": "string"},
                "agent_id": {"type": "string"},
                "run_id": {"type": "string"},
            },
        },
    },
]


def _call_tool(name: str, arguments: dict[str, Any]) -> Any:
    """分派工具调用到 OSS 封装。"""
    if name == "add_memory":
        return add_memory(
            arguments.get("messages") or [],
            user_id=arguments.get("user_id"),
            agent_id=arguments.get("agent_id"),
            run_id=arguments.get("run_id"),
            metadata=arguments.get("metadata"),
            infer=arguments.get("infer", True),
        )
    if name == "search_memories":
        return search_memories(
            arguments.get("query") or "",
            filters=arguments.get("filters"),
            top_k=arguments.get("top_k"),
        )
    if name == "get_memories":
        return list_memories(
            user_id=arguments.get("user_id"),
            agent_id=arguments.get("agent_id"),
            run_id=arguments.get("run_id"),
        )
    if name == "get_memory":
        return get_memory(arguments.get("memory_id") or "")
    if name == "update_memory":
        return update_memory(
            arguments.get("memory_id") or "",
            text=arguments.get("text"),
            metadata=arguments.get("metadata"),
        )
    if name == "delete_memory":
        return delete_memory(arguments.get("memory_id") or "")
    if name == "delete_all_memories":
        return delete_all_memories(
            user_id=arguments.get("user_id"),
            agent_id=arguments.get("agent_id"),
            run_id=arguments.get("run_id"),
        )
    raise ValueError(f"未知工具: {name}")


def _tool_response(text: str, *, is_error: bool = False) -> dict[str, Any]:
    return {"content": [{"type": "text", "text": text}], "isError": is_error}


def handle_request(message: Any) -> dict[str, Any] | None:
    """处理单条 JSON-RPC 消息。"""
    if not isinstance(message, dict):
        return None
    request_id = message.get("id")
    method = message.get("method")

    if method == "notifications/initialized":
        return None
    if method == "initialize":
        requested = (message.get("params") or {}).get("protocolVersion")
        return {
            "jsonrpc": "2.0",
            "id": request_id,
            "result": {
                "protocolVersion": requested or PROTOCOL_VERSION,
                "capabilities": {"tools": {"listChanged": False}},
                "serverInfo": {"name": SERVER_NAME, "version": SERVER_VERSION},
            },
        }
    if method == "ping":
        return {"jsonrpc": "2.0", "id": request_id, "result": {}}
    if method == "tools/list":
        return {"jsonrpc": "2.0", "id": request_id, "result": {"tools": TOOLS}}
    if method == "tools/call":
        params = message.get("params") or {}
        tool_name = params.get("name", "")
        arguments = params.get("arguments") or {}
        try:
            result = _tool_response(
                json.dumps(_call_tool(tool_name, arguments), ensure_ascii=False, indent=2)
            )
        except Exception as error:  # 记忆失败不得阻断 agent
            result = _tool_response(f"{tool_name} 失败: {error}", is_error=True)
        return {"jsonrpc": "2.0", "id": request_id, "result": result}
    if request_id is None:
        return None
    return {
        "jsonrpc": "2.0",
        "id": request_id,
        "error": {"code": -32601, "message": "Method not found"},
    }


def main() -> int:
    """stdin 逐行读取 JSON-RPC 请求, stdout 写响应。"""
    for raw_line in sys.stdin:
        try:
            message = json.loads(raw_line)
            response = handle_request(message)
        except json.JSONDecodeError:
            response = {
                "jsonrpc": "2.0",
                "id": None,
                "error": {"code": -32700, "message": "Parse error"},
            }
        except Exception:
            response = {
                "jsonrpc": "2.0",
                "id": None,
                "error": {"code": -32603, "message": "Internal error"},
            }
        if response is not None:
            sys.stdout.write(json.dumps(response, separators=(",", ":")) + "\n")
            sys.stdout.flush()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
