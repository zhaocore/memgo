#!/usr/bin/env python3
"""Capture the post-compaction summary into MemGo.

PreCompact hooks fire BEFORE the summary is generated, so they can't
store the actual compact-summary text. This script runs at
SessionStart with source=compact, reads the transcript, finds the
most recent entry flagged isCompactSummary=true, and stores it via
MemGo OSS 面(metadata.type=compact_summary)。

Input:  JSON on stdin with transcript_path, session_id, source
Output: stderr logs only (exit 0 always -- must not block)

由 on_session_start.sh 在后台启动; 用户可见的 bootstrap 文本无需等待网络。
"""

from __future__ import annotations

import json
import logging
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from _api import add_memory
from _identity import resolve_api_key, resolve_user_id
from _project import resolve_agent_id, resolve_branch

log = logging.getLogger("memgo-compact-summary")
log.setLevel(logging.DEBUG)
_handler = logging.StreamHandler(sys.stderr)
_handler.setFormatter(logging.Formatter("[memgo-compact-summary] %(message)s"))
log.addHandler(_handler)

if os.environ.get("MEMGO_DEBUG"):
    _log_dir = os.path.expanduser("~/.memgo")
    try:
        os.makedirs(_log_dir, exist_ok=True)
        _file_handler = logging.FileHandler(os.path.join(_log_dir, "hooks.log"))
        _file_handler.setFormatter(logging.Formatter("[memgo-compact-summary] %(asctime)s %(message)s"))
        log.addHandler(_file_handler)
    except OSError:
        pass

MAX_TAIL_LINES = 2000
MAX_SUMMARY_CHARS = 50000


def tail_lines(filepath: str, n: int) -> list[str]:
    try:
        with open(filepath, "rb") as f:
            f.seek(0, 2)
            file_size = f.tell()
            if file_size == 0:
                return []
            chunk_size = min(file_size, n * 4096)
            f.seek(max(0, file_size - chunk_size))
            data = f.read().decode("utf-8", errors="replace")
            return data.splitlines()[-n:]
    except OSError:
        return []


def find_compact_summary(lines: list[str]) -> str:
    """反向遍历 transcript, 返回最近一条 isCompactSummary=true 的文本内容。"""
    for line in reversed(lines):
        line = line.strip()
        if not line:
            continue
        try:
            entry = json.loads(line)
        except json.JSONDecodeError:
            continue
        if not entry.get("isCompactSummary"):
            continue

        message = entry.get("message", {})
        content = message.get("content", [])
        if isinstance(content, str):
            return content[:MAX_SUMMARY_CHARS]
        if isinstance(content, list):
            parts = []
            for block in content:
                if isinstance(block, str):
                    parts.append(block)
                elif isinstance(block, dict) and block.get("type") == "text":
                    parts.append(block.get("text", ""))
            return "\n".join(parts).strip()[:MAX_SUMMARY_CHARS]
    return ""


def store_summary(summary: str, user_id: str, session_id: str, agent_id: str, branch: str) -> bool:
    metadata = {
        "type": "compact_summary",
        "source": "session-start-compact",
        "session_id": session_id,
    }
    if branch:
        metadata["branch"] = branch
    # 紧凑摘要是模型写的第一人称散文, 没有标识其身份的框架。MemGo server 会从
    # 消息提取事实, role 是判断说话者的信号, 故这里用 role="assistant"。
    try:
        results = add_memory(
            [{"role": "assistant", "content": summary}],
            user_id=user_id,
            agent_id=agent_id,
            metadata=metadata,
            infer=True,
        )
        log.info("Compact summary stored (%d memory(-ies))", len(results))
        return True
    except Exception as e:
        log.warning("API call failed: %s", e)
        return False


def main():
    if not resolve_api_key():
        log.debug("MEMGO_API_KEY not set, skipping capture")
        return

    try:
        hook_input = json.loads(sys.stdin.read())
    except (json.JSONDecodeError, OSError):
        log.debug("No valid JSON on stdin")
        return

    transcript_path = hook_input.get("transcript_path", "")
    if not transcript_path:
        log.debug("No transcript_path provided")
        return

    session_id = hook_input.get("session_id", "")
    cwd = hook_input.get("cwd") or None
    user_id = resolve_user_id()
    agent_id = resolve_agent_id(cwd)
    branch = resolve_branch(cwd)

    lines = tail_lines(transcript_path, MAX_TAIL_LINES)
    if not lines:
        log.debug("Transcript empty or unreadable: %s", transcript_path)
        return

    summary = find_compact_summary(lines)
    if not summary:
        log.debug("No isCompactSummary entry found")
        return

    if len(summary.strip()) < 100:
        log.debug("Compact summary too short (%d chars) — skipping", len(summary.strip()))
        return

    marker_dir = os.path.expanduser("~/.memgo")
    marker_file = os.path.join(marker_dir, f"compact_captured_{session_id}")
    if session_id and os.path.isfile(marker_file):
        log.info("Compact summary already captured for session %s — skipping", session_id)
        return

    log.info("Capturing compact summary (%d chars)", len(summary))
    if store_summary(summary, user_id, session_id, agent_id, branch):
        if session_id:
            try:
                os.makedirs(marker_dir, exist_ok=True)
                with open(marker_file, "w") as f:
                    f.write("")
            except OSError:
                pass


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        log.error("Unexpected error: %s", e)
    sys.exit(0)
