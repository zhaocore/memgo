#!/usr/bin/env python3
"""MemGo 插件本地遥测(落 spool 文件, 不对外发送)。

MemGo server 已决定"网络发送省略"(仅本地状态文件), 接入端同样只落本地
spool, 删掉 PostHog 发送与 /v1/ping/ 邮箱解析。纯 stdlib。

CLI usage (由 hooks 后台子进程调用):
  python3 telemetry.py <event_type> [--memory_count=N] [--error_detected]
                                    [--file_paths_detected] [--source=<src>]
                                    [--tool=<name>] [--files_count=N]

Opt-out: 设 MEMGO_TELEMETRY=false(或 0/no/off)关闭。

本地 spool: ~/.memgo/telemetry/events.jsonl(逐行追加 JSON)。
永不记录: 用户内容、记忆内容、API key、原始 user/agent id。
"""

from __future__ import annotations

import json
import os
import platform
import sys
from datetime import datetime

SPOOL_DIR = os.path.expanduser("~/.memgo/telemetry")
SPOOL_FILE = os.path.join(SPOOL_DIR, "events.jsonl")


def detect_platform() -> str:
    explicit = os.environ.get("MEMGO_PLATFORM")
    if explicit:
        return explicit
    if os.environ.get("PLUGIN_ROOT"):
        return "codex"
    return "plugin"


def is_enabled() -> bool:
    return os.environ.get("MEMGO_TELEMETRY", "true").lower() not in ("false", "0", "no", "off")


def _record(event_type: str, properties: dict) -> None:
    """把一条事件追加到本地 spool 文件。"""
    row = {
        "ts": datetime.now().isoformat(),
        "event": event_type,
        "platform": detect_platform(),
        "os": sys.platform,
        "os_version": platform.version(),
        **properties,
    }
    try:
        os.makedirs(SPOOL_DIR, exist_ok=True)
        with open(SPOOL_FILE, "a") as f:
            f.write(json.dumps(row, ensure_ascii=False) + "\n")
    except OSError:
        pass


def emit(event_type: str, properties: dict | None = None) -> None:
    if not is_enabled():
        return
    _record(f"plugin.{event_type}", properties or {})


def main() -> int:
    if not is_enabled():
        return 0
    if len(sys.argv) < 2:
        return 1

    event_type = sys.argv[1]
    properties: dict = {}

    for arg in sys.argv[2:]:
        if arg.startswith("--memory_count="):
            try:
                properties["memory_count"] = int(arg.split("=", 1)[1])
            except ValueError:
                pass
        elif arg == "--error_detected":
            properties["error_detected"] = True
        elif arg == "--file_paths_detected":
            properties["file_paths_detected"] = True
        elif arg.startswith("--source="):
            properties["source_detail"] = arg.split("=", 1)[1]
        elif arg.startswith("--tool="):
            properties["tool"] = arg.split("=", 1)[1]
        elif arg.startswith("--files_count="):
            try:
                properties["files_count"] = int(arg.split("=", 1)[1])
            except ValueError:
                pass

    emit(event_type, properties)
    return 0


if __name__ == "__main__":
    sys.exit(main())
