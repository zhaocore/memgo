#!/usr/bin/env python3
"""MemGo OSS API 客户端(纯 stdlib, 直连自托管 server)。

所有 hooks 的数据读写统一走这里, 不依赖外部 SDK。鉴权头用 X-API-Key
(不是 Authorization: Token), 写入/搜索均同步返回 results, 无事件轮询。

端点与字段见 integrations/README.md 的 OSS API 契约, 与共享
mcp/memgo_mcp.py 保持同一套调用约定(实体范围: user_id/agent_id/run_id)。
"""

from __future__ import annotations

import json
import os
import urllib.error
import urllib.parse
import urllib.request
from typing import Any

DEFAULT_BASE_URL = "https://memgo.wxget.com"
TIMEOUT = 15


def base_url() -> str:
    """server 根地址, 去除尾部斜杠。"""
    return os.environ.get("MEMGO_BASE_URL", DEFAULT_BASE_URL).rstrip("/")


def api_key() -> str:
    """鉴权 key, 缺失时抛错。"""
    key = os.environ.get("MEMGO_API_KEY", "").strip()
    if not key:
        raise RuntimeError("MEMGO_API_KEY 未设置")
    return key


def _request_json(method: str, path: str, *, payload: Any | None = None) -> Any:
    """发起 OSS API 请求并解析 JSON 响应, 失败抛带状态的 RuntimeError。"""
    url = base_url() + path
    headers = {"X-API-Key": api_key(), "Content-Type": "application/json"}
    data = None if payload is None else json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(request, timeout=TIMEOUT) as response:
            raw = response.read()
    except urllib.error.HTTPError as error:
        body = error.read().decode("utf-8", "replace")
        raise RuntimeError(f"{method} {path}: HTTP {error.code}, {body[:500]}") from error
    except urllib.error.URLError as error:
        raise RuntimeError(f"{method} {path}: 网络错误 {error.reason}") from error
    return json.loads(raw or b"{}")


def _results(response: Any) -> list[dict[str, Any]]:
    results = response.get("results") if isinstance(response, dict) else None
    return results if isinstance(results, list) else []


def add_memory(
    messages: list[dict[str, str]],
    *,
    user_id: str | None = None,
    agent_id: str | None = None,
    run_id: str | None = None,
    metadata: dict[str, Any] | None = None,
    infer: bool = True,
) -> list[dict[str, Any]]:
    """同步写入消息, 返回提取后的 results 列表。"""
    payload: dict[str, Any] = {"messages": messages, "infer": infer}
    if user_id:
        payload["user_id"] = user_id
    if agent_id:
        payload["agent_id"] = agent_id
    if run_id:
        payload["run_id"] = run_id
    if metadata:
        payload["metadata"] = metadata
    return _results(_request_json("POST", "/memories", payload=payload))


def search_memories(
    query: str,
    *,
    filters: dict[str, str] | None = None,
    top_k: int | None = None,
) -> list[dict[str, Any]]:
    """语义搜索, 返回 results 列表。filters 至少含一个 user_id/agent_id/run_id。"""
    payload: dict[str, Any] = {"query": query}
    if filters:
        payload["filters"] = filters
    if top_k:
        payload["top_k"] = top_k
    return _results(_request_json("POST", "/search", payload=payload))


def list_memories(
    *,
    user_id: str | None = None,
    agent_id: str | None = None,
    run_id: str | None = None,
) -> list[dict[str, Any]]:
    """按实体范围列出记忆。"""
    query: list[str] = []
    for name, value in (("user_id", user_id), ("agent_id", agent_id), ("run_id", run_id)):
        if value:
            query.append(f"{name}={urllib.parse.quote(value)}")
    suffix = f"?{'&'.join(query)}" if query else ""
    return _results(_request_json("GET", f"/memories{suffix}"))
