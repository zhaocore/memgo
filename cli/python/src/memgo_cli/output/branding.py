"""终端品牌面板、颜色与状态输出。"""

import os
import sys
import time
from collections.abc import Iterator
from contextlib import contextmanager
from dataclasses import dataclass

from rich.console import Console
from rich.panel import Panel
from rich.status import Status
from rich.text import Text

# 进度、错误和耗时信息写入标准错误
_err = Console(stderr=True)

LOGO = r"""
█   █ █████ █   █  ███   ███     ████ █     █████
██ ██ █     ██ ██ █     █   █   █     █       █
█ █ █ ████  █ █ █ █ ███ █   █   █     █       █
█   █ █     █   █ █   █ █   █   █     █       █
█   █ █████ █   █  ███   ███     ████ █████ █████
"""

LOGO_MINI = "◆ memgo"

TAGLINE = "The Memory Layer for AI Agents"

BRAND_COLOR = "#f472b6"
ACCENT_COLOR = "#67e8f9"
SUCCESS_COLOR = "#34d399"
ERROR_COLOR = "#fb7185"
WARNING_COLOR = "#facc15"
DIM_COLOR = "#94a3b8"


def _sym(fancy: str, plain: str) -> str:
    """彩色终端使用图形符号，其他环境使用纯文本。"""
    if not sys.stdout.isatty() or os.environ.get("NO_COLOR") is not None:
        return plain
    return fancy


def print_banner(console: Console) -> None:
    """显示欢迎面板，代理模式下保持静默。"""
    from memgo_cli.runtime.state import is_agent_mode

    if is_agent_mode():
        return
    logo_text = Text(LOGO, style=f"bold {BRAND_COLOR}")
    tagline = Text(f"  {TAGLINE}\n", style=f"{ACCENT_COLOR}")

    content = Text()
    content.append_text(logo_text)
    content.append_text(tagline)

    panel = Panel(
        content,
        border_style=BRAND_COLOR,
        padding=(0, 2),
        subtitle=f"[{DIM_COLOR}]Python SDK · v{_get_version()}[/]",
        subtitle_align="right",
    )
    console.print(panel)


def print_success(console: Console, message: str) -> None:
    """在人工输出模式显示成功信息。"""
    from memgo_cli.runtime.state import is_agent_mode

    if is_agent_mode():
        return
    sym = _sym("✓", "[ok]")
    console.print(f"[{SUCCESS_COLOR}]{sym}[/] {message}")


def print_error(console: Console, message: str, hint: str | None) -> None:
    """显示错误，代理模式使用标准输出 JSON 信封。"""
    from memgo_cli.runtime.state import get_current_command, is_agent_mode

    if is_agent_mode():
        import json as _json

        envelope = {
            "status": "error",
            "command": get_current_command(),
            "error": message,
            "data": None,
        }
        print(_json.dumps(envelope))
        return
    from rich.markup import escape

    sym = _sym("✗", "[error]")
    console.print(f"[{ERROR_COLOR}]{sym} Error:[/] {escape(str(message))}")
    if hint:
        console.print(f"  [{DIM_COLOR}]{escape(str(hint))}[/]")


def print_warning(console: Console, message: str) -> None:
    """在人工输出模式显示告警。"""
    from memgo_cli.runtime.state import is_agent_mode

    if is_agent_mode():
        return
    sym = _sym("⚠", "[warn]")
    console.print(f"[{WARNING_COLOR}]{sym}[/] {message}")


def print_info(console: Console, message: str) -> None:
    """在人工输出模式显示提示。"""
    from memgo_cli.runtime.state import is_agent_mode

    if is_agent_mode():
        return
    sym = _sym("◆", "*")
    console.print(f"[{BRAND_COLOR}]{sym}[/] {message}")


@dataclass
class TimedStatusContext:
    """异步操作结束时需要显示的状态文本。"""

    success_msg: str = ""
    error_msg: str = ""


@contextmanager
def timed_status(console: Console, message: str) -> Iterator[TimedStatusContext]:
    """显示计时进度并提供最终状态上下文。

    进度和耗时只写入标准错误；代理模式不显示。异常原样抛出。
    """
    from memgo_cli.runtime.state import is_agent_mode

    ctx = TimedStatusContext()
    if is_agent_mode():
        try:
            yield ctx
        except Exception:
            if ctx.error_msg:
                print_error(_err, ctx.error_msg, hint=None)
            raise
        return

    start = time.perf_counter()
    try:
        with Status(f"[{DIM_COLOR}]{message}[/]", console=_err):
            yield ctx
    except Exception:
        elapsed = time.perf_counter() - start
        if ctx.error_msg:
            print_error(_err, f"{ctx.error_msg} ({elapsed:.2f}s)", hint=None)
            if "Authentication failed" in ctx.error_msg:
                _err.print(
                    f"  [{DIM_COLOR}]Run [bold]memgo init[/bold] to reconfigure your API key"
                    f" · [bold]https://app.memgo.ai/dashboard/api-keys?utm_source=oss&utm_medium=cli-python[/bold][/]"
                )
        raise
    else:
        elapsed = time.perf_counter() - start
        if ctx.success_msg:
            print_success(_err, f"{ctx.success_msg} ({elapsed:.2f}s)")


def print_scope(console: Console, **ids: str | None) -> None:
    """显示非空实体 ID 组成的当前操作范围。"""
    from memgo_cli.runtime.state import is_agent_mode

    if is_agent_mode():
        return
    parts = []
    for key, val in ids.items():
        if val:
            parts.append(f"{key}={val}")
    if parts:
        scope_str = ", ".join(parts)
        console.print(f"  [{DIM_COLOR}]Scope: {scope_str}[/]")


def _get_version() -> str:
    """读取包版本。"""
    from memgo_cli import __version__

    return __version__
