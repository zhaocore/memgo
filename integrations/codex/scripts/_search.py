#!/usr/bin/env python3
"""MemGo 搜索助手: 统一 /search 调用 + 上下文格式化。

hooks 的预取/错误扫描/文件上下文都走这里, 直连 OSS 面。
OSS 的 filters 只支持 user_id/agent_id/run_id, 不支持按 metadata.type
过滤, 故查询不再带分类过滤(仅展示记忆自带的 metadata.type)。
"""

from __future__ import annotations

import os
import sys

from _api import search_memories as oss_search


def search_memories(
    user_id: str,
    agent_id: str,
    query: str,
    top_k: int = 3,
) -> list[dict]:
    """按 user_id + agent_id 范围搜索, 返回结果列表。无 API key 时返回空。"""
    if not os.environ.get("MEMGO_API_KEY"):
        return []
    try:
        results = oss_search(
            query, filters={"user_id": user_id, "agent_id": agent_id}, top_k=top_k
        )
        return results[:top_k]
    except Exception as e:
        print(f"[memgo] search request failed: {e}", file=sys.stderr)
        return []


def format_results_for_context(
    memories: list[dict],
    heading: str = "Relevant memories",
) -> str:
    if not memories:
        return ""
    lines = [f"### {heading}", ""]
    for m in memories:
        mid = m.get("id", "?")[:8]
        text = m.get("memory", "")[:200]
        cat = (m.get("metadata") or {}).get("type", "unknown")
        lines.append(f"- [{cat}] {text} [memgo:{mid}]")
    lines.append("")
    return "\n".join(lines)
