#!/usr/bin/env python3
"""File-context injection for PreToolUse/Read hook.

在 agent 读取文件前, 搜索 MemGo 中引用该文件路径的记忆并返回紧凑的
历史工作时间线, 例如 "上次你在这里修过一个空指针"。

Input:  file_path (positional arg), env vars for identity
Output: additionalContext 文本到 stdout(空则无输出)
"""

from __future__ import annotations

import os
import sys
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from _formatting import TYPE_ICONS, format_age
from _identity import resolve_api_key, resolve_user_id
from _project import resolve_agent_id
from _search import search_memories

FILE_READ_GATE_MIN_BYTES = 1500
MAX_RESULTS = 5


def gate_file(file_path: str, cwd: str) -> str | None:
    """通过 gating 则返回解析后的绝对路径, 否则返回 None。"""
    if not file_path:
        return None
    p = Path(file_path)
    if not p.is_absolute():
        p = Path(cwd) / p
    try:
        p = p.resolve()
        if not p.is_file():
            return None
        if p.stat().st_size < FILE_READ_GATE_MIN_BYTES:
            return None
        return str(p)
    except OSError:
        return None


def relative_path(abs_path: str, cwd: str) -> str:
    try:
        return os.path.relpath(abs_path, cwd)
    except ValueError:
        return abs_path


def format_timeline(memories: list[dict], file_path: str) -> str:
    """把记忆格式化为紧凑时间线, 用于上下文注入。"""
    if not memories:
        return ""

    rel = file_path
    lines = [
        f"Prior work on `{rel}` — {len(memories)} memories found.",
        "Need details? Use `search_memories` with the memory ID.",
        "",
    ]

    for m in memories:
        mid = m.get("id", "?")[:8]
        text = (m.get("memory", "") or "")[:150].replace("\n", " ").strip()
        meta = m.get("metadata") or {}
        cat = meta.get("type", "unknown")
        icon = TYPE_ICONS.get(cat, "❓")
        age = format_age(m)
        age_str = f" ({age})" if age else ""
        lines.append(f"- {icon} [{cat}]{age_str} {text} [memgo:{mid}]")

    return "\n".join(lines)


def search_file_context(
    user_id: str, agent_id: str, file_path: str, cwd: str
) -> str:
    """搜索与文件路径相关的记忆。"""
    rel = relative_path(file_path, cwd)
    basename = os.path.basename(file_path)

    query = f"{rel} {basename}" if rel != basename else rel
    results = search_memories(user_id, agent_id, query, top_k=MAX_RESULTS)

    results = results[:MAX_RESULTS]

    return format_timeline(results, rel)


def main():
    if len(sys.argv) < 2:
        sys.exit(0)

    file_path = sys.argv[1]
    cwd = sys.argv[2] if len(sys.argv) > 2 else os.getcwd()

    if not resolve_api_key():
        sys.exit(0)

    resolved = gate_file(file_path, cwd)
    if not resolved:
        sys.exit(0)

    user_id = resolve_user_id()
    agent_id = resolve_agent_id(cwd)

    timeline = search_file_context(user_id, agent_id, resolved, cwd)
    if not timeline:
        sys.exit(0)

    print(timeline, end="")


if __name__ == "__main__":
    try:
        main()
    except Exception:
        pass
    sys.exit(0)
