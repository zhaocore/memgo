#!/usr/bin/env python3
"""Auto-capture recent conversation exchanges into MemGo.

由 UserPromptSubmit hook 在后台运行(每第 3 条消息)。
读取 transcript 末尾几段对话, 通过 OSS 面同步写入, 由 MemGo server
抽取事实(不依赖平台分类)。

Input:  env vars (MEMGO_API_KEY, MEMGO_RESOLVED_USER_ID, MEMGO_AGENT_ID, etc.)
        argv[1] = transcript_path
Output: stderr logs only (exit 0 always — must not block)
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

log = logging.getLogger("memgo-auto-capture")
log.setLevel(logging.DEBUG)
_handler = logging.StreamHandler(sys.stderr)
_handler.setFormatter(logging.Formatter("[memgo-auto-capture] %(message)s"))
log.addHandler(_handler)

if os.environ.get("MEMGO_DEBUG"):
    _log_dir = os.path.expanduser("~/.memgo")
    try:
        os.makedirs(_log_dir, exist_ok=True)
        _fh = logging.FileHandler(os.path.join(_log_dir, "hooks.log"))
        _fh.setFormatter(logging.Formatter("[memgo-auto-capture] %(asctime)s %(message)s"))
        log.addHandler(_fh)
    except OSError:
        pass

TAIL_LINES = 200
MAX_CONTENT_CHARS = 8000
MIN_CONTENT_CHARS = 100


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


def extract_recent_exchanges(lines: list[str], max_exchanges: int = 3) -> list[dict]:
    """从 transcript JSONL 提取最后 N 组 user+assistant 消息对。"""
    messages = []
    for line in lines:
        line = line.strip()
        if not line:
            continue
        try:
            entry = json.loads(line)
        except json.JSONDecodeError:
            continue

        if entry.get("isCompactSummary"):
            continue

        msg = entry.get("message", {})
        role = msg.get("role", "")
        if role not in ("user", "assistant"):
            continue

        content = msg.get("content", "")
        if isinstance(content, list):
            parts = []
            for block in content:
                if isinstance(block, str):
                    parts.append(block)
                elif isinstance(block, dict) and block.get("type") == "text":
                    parts.append(block.get("text", ""))
            content = "\n".join(parts).strip()

        if not content or len(content) < 20:
            continue

        # Skip tool-call-only assistant messages
        if role == "assistant" and content.startswith("{"):
            continue

        messages.append({"role": role, "content": content[:2000]})

    # Take last N exchanges (pairs of user+assistant)
    if not messages:
        return []

    result = messages[-(max_exchanges * 2):]
    return result


def store_exchange(messages: list[dict], user_id: str, agent_id: str, branch: str, session_id: str) -> bool:
    """通过 OSS 面同步写入对话记录。"""
    metadata = {
        "type": "auto_capture",
        "source": "auto_capture",
        "confidence": 0.7,
    }
    if branch:
        metadata["branch"] = branch
    if session_id:
        metadata["session_id"] = session_id

    try:
        results = add_memory(
            messages,
            user_id=user_id,
            agent_id=agent_id,
            run_id=session_id or None,
            metadata=metadata,
            infer=True,
        )
        log.info("Auto-captured %d memory(-ies)", len(results))
        return True
    except Exception as e:
        log.warning("API call failed: %s", e)
        return False


def main():
    if not resolve_api_key():
        log.debug("MEMGO_API_KEY not set, skipping")
        return

    if len(sys.argv) < 2:
        log.debug("No transcript_path argument")
        return

    transcript_path = sys.argv[1]
    if not transcript_path or not os.path.isfile(transcript_path):
        log.debug("Transcript not found: %s", transcript_path)
        return

    user_id = resolve_user_id()
    agent_id = resolve_agent_id()
    branch = resolve_branch()
    session_id = ""
    sid_file = f"/tmp/memgo_session_id_{os.environ.get('USER', 'default')}"
    if os.path.isfile(sid_file):
        try:
            with open(sid_file) as f:
                session_id = f.read().strip()
        except OSError:
            pass

    lines = tail_lines(transcript_path, TAIL_LINES)
    if not lines:
        log.debug("Transcript empty")
        return

    messages = extract_recent_exchanges(lines, max_exchanges=4)
    if not messages:
        log.debug("No substantial exchanges found")
        return

    total_chars = sum(len(m["content"]) for m in messages)
    if total_chars < MIN_CONTENT_CHARS:
        log.debug("Exchanges too short (%d chars), skipping", total_chars)
        return

    log.info("Auto-capturing %d messages (%d chars)", len(messages), total_chars)
    if store_exchange(messages, user_id, agent_id, branch, session_id):
        try:
            import session_stats
            session_stats.record_add("auto_capture")
        except Exception:
            pass


if __name__ == "__main__":
    try:
        main()
    except Exception as e:
        log.error("Unexpected error: %s", e)
    sys.exit(0)
