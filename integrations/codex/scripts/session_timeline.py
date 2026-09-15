#!/usr/bin/env python3
"""Fetch recent memories and format a compact timeline for SessionStart.

搜索 MemGo OSS GET /memories 获取项目最近记忆, 格式化为紧凑活动时间线,
注入到 SessionStart banner 下方。

Input:  env vars (MEMGO_API_KEY, MEMGO_RESOLVED_USER_ID, MEMGO_AGENT_ID, MEMGO_CWD)
Output: Compact timeline text to stdout (empty if nothing found)
"""

from __future__ import annotations

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from _api import list_memories
from _formatting import TYPE_ICONS, format_age
from _identity import resolve_api_key, resolve_user_id
from _project import resolve_agent_id

MAX_RECENT = 10


def fetch_recent_memories(user_id: str, agent_id: str) -> list[dict]:
    """按 user_id + agent_id 获取最近记忆。"""
    try:
        results = list_memories(user_id=user_id, agent_id=agent_id)
        return results[:MAX_RECENT]
    except Exception:
        return []


def format_timeline(memories: list[dict]) -> str:
    """把记忆格式化为紧凑的最近活动时间线。"""
    if not memories:
        return ""

    lines = ["### Recent Activity", ""]

    for m in memories:
        mid = m.get("id", "?")[:8]
        text = (m.get("memory", "") or "")[:120].replace("\n", " ").strip()
        meta = m.get("metadata") or {}
        cat = meta.get("type", "unknown")
        icon = TYPE_ICONS.get(cat, "❓")
        age = format_age(m)
        age_str = f" ({age})" if age else ""
        lines.append(f"- {icon} [{cat}]{age_str} {text} [memgo:{mid}]")

    lines.append("")
    lines.append("Search MemGo for details on any of these, or for past decisions and task learnings relevant to the current task.")

    return "\n".join(lines)


def main():
    if not resolve_api_key():
        return

    user_id = resolve_user_id()
    agent_id = resolve_agent_id(os.environ.get("MEMGO_CWD"))

    memories = fetch_recent_memories(user_id, agent_id)
    if not memories:
        return

    timeline = format_timeline(memories)
    if timeline:
        print(timeline, end="")


if __name__ == "__main__":
    try:
        main()
    except Exception:
        pass
    sys.exit(0)
