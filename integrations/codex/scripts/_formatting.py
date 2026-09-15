"""MemGo 插件 hooks 共用的格式化工具。

file_context.py / session_timeline.py 等展示记忆的脚本共用。
"""

from __future__ import annotations

TYPE_ICONS = {
    "decision": "⚖️",
    "anti_pattern": "\U0001f534",
    "bug_fix": "\U0001f534",
    "convention": "\U0001f504",
    "task_learning": "\U0001f535",
    "user_preference": "\U0001f7e3",
    "session_summary": "\U0001f4cb",
    "session_state": "\U0001f4cb",
    "project_profile": "\U0001f4d6",
    "compact_summary": "\U0001f4cb",
    "auto_capture": "✅",
    "environmental": "🌐",
    "health_check": "🩺",
}


def format_age(memory: dict) -> str:
    """格式化记忆创建时间距今多久, 如 '2h ago'、'3d ago'。"""
    created = memory.get("created_at", "")
    if not created:
        return ""
    try:
        from datetime import datetime, timezone

        if created.endswith("Z"):
            created = created[:-1] + "+00:00"
        dt = datetime.fromisoformat(created)
        now = datetime.now(timezone.utc)
        delta = now - dt
        seconds = int(delta.total_seconds())
        if seconds < 3600:
            return f"{seconds // 60}m ago"
        if seconds < 86400:
            return f"{seconds // 3600}h ago"
        days = seconds // 86400
        if days == 1:
            return "1d ago"
        if days < 30:
            return f"{days}d ago"
        return f"{days // 30}mo ago"
    except Exception:
        return ""
