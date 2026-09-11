"""标准输入来源检测和管道读取。"""

from __future__ import annotations

import os
import stat as _stat_mod
import sys


def _stdin_is_piped() -> bool:
    """判断标准输入是否来自实际管道或重定向文件。"""
    from memgo_cli.runtime.state import is_agent_mode

    if is_agent_mode():
        return False
    try:
        mode = os.fstat(sys.stdin.fileno()).st_mode
        return _stat_mod.S_ISFIFO(mode) or _stat_mod.S_ISREG(mode)
    except (OSError, ValueError):
        return False


def _read_stdin() -> str | None:
    """读取管道或重定向内容；终端和代理模式不读取。"""
    if _stdin_is_piped():
        return sys.stdin.read().strip() or None
    return None
