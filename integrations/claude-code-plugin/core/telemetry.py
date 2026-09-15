#!/usr/bin/env python3
"""MemGo Claude Code 插件的匿名使用遥测。

遥测只保留本地部分:
`record` 把事件追加到本地 JSONL spool(append),另维护一个本机匿名 id
状态文件。MemGo server 已决定"网络发送省略",这里同样不对外发送任何数据。

Hooks 运行在 3-6 秒预算内、每次工具调用都会触发,所以 record 从不触网:
写一行 JSON 立即返回。纯 stdlib,与插件其余部分一致。
用 MEMGO_TELEMETRY=false 关闭。

绝不发送 prompt、记忆文本、查询、文件路径、仓库名或 API key:
只记录事件名、耗时、数量、粗略结果和加盐哈希。
"""

from __future__ import annotations

import hashlib
import json
import os
import platform
import sys
import uuid
from pathlib import Path
from typing import Any

import memory_core

# 事件名前缀沿用 "code",保持本机事件历史一致;
# 若要区分 MemGo,可改为 "memgo"。
EVENT_PREFIX = "code"
SPOOL_LIMIT_BYTES = 256 * 1024


def is_enabled() -> bool:
    """Whether telemetry is switched on for this process."""
    return os.environ.get("MEMGO_TELEMETRY", "true").strip().lower() not in {
        "false",
        "0",
        "no",
        "off",
    }


def _digest(value: str, length: int = 16) -> str:
    return hashlib.sha256(value.encode("utf-8")).hexdigest()[:length]


def _spool_path() -> Path:
    return memory_core.data_dir() / "telemetry.jsonl"


def _identity_path() -> Path:
    return memory_core.data_dir() / "telemetry-identity.json"


def _read_identity() -> dict[str, str]:
    try:
        value = json.loads(_identity_path().read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return {}
    return value if isinstance(value, dict) else {}


def _write_identity(identity: dict[str, str]) -> None:
    path = _identity_path()
    temporary = path.with_suffix(f".{os.getpid()}.tmp")
    try:
        path.parent.mkdir(parents=True, exist_ok=True)
        temporary.write_text(json.dumps(identity), encoding="utf-8")
        temporary.replace(path)
    except OSError:
        try:
            temporary.unlink()
        except OSError:
            pass


def anonymous_id(identity: dict[str, str] | None = None) -> str:
    """Per-machine anonymous identifier, created and persisted on first use."""
    identity = _read_identity() if identity is None else identity
    existing = identity.get("anonymous_id")
    if existing:
        return existing
    created = f"code-anon-{uuid.uuid4().hex}"
    identity["anonymous_id"] = created
    _write_identity(identity)
    return created


def is_first_run() -> bool:
    """Whether this machine has never recorded a plugin event before."""
    return not _identity_path().exists()


def record(
    event: str,
    *,
    repo: Any = None,
    session_id: str | None = None,
    **properties: Any,
) -> None:
    """Append one event to the local spool. Never blocks and never raises."""
    if not is_enabled():
        return
    try:
        spool = _spool_path()
        try:
            if spool.stat().st_size > SPOOL_LIMIT_BYTES:
                return
        except OSError:
            pass
        properties.update(
            harness="claude-code",
            plugin_version=memory_core.PLUGIN_VERSION,
            os=sys.platform,
            python_version=platform.python_version(),
        )
        if repo is not None:
            properties["repo_hash"] = _digest(getattr(repo, "identity", ""))
        if session_id:
            properties["session_hash"] = _digest(session_id)
        line = json.dumps(
            {
                "event": f"{EVENT_PREFIX}.{event}",
                "timestamp": memory_core.utc_now(),
                "properties": {
                    key: value for key, value in properties.items() if value is not None
                },
            },
            separators=(",", ":"),
            default=str,
        )
        spool.parent.mkdir(parents=True, exist_ok=True)
        with spool.open("a", encoding="utf-8") as handle:
            handle.write(line + "\n")
    except Exception:
        pass


def error_kind(exc: BaseException | str) -> str:
    """Coarse, content-free label for a failure, safe to keep locally."""
    text = exc if isinstance(exc, str) else f"{type(exc).__name__}: {exc}"
    lowered = text.lower()
    if "timed out" in lowered or "timeout" in lowered:
        return "timeout"
    if "401" in lowered or "403" in lowered or "unauthor" in lowered or "forbidden" in lowered:
        return "auth"
    if "429" in lowered or "rate limit" in lowered:
        return "rate-limited"
    if any(code in lowered for code in ("500", "502", "503", "504")):
        return "server-error"
    if "400" in lowered or "422" in lowered:
        return "bad-request"
    if isinstance(exc, str):
        return "other"
    return type(exc).__name__
