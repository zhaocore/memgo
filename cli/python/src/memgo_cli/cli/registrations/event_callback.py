"""命令 event_callback 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.cli.context import _fire_telemetry

console = Console()
err_console = Console(stderr=True)


def _event_callback(ctx: typer.Context) -> None:
    """处理事件命令组入口。"""
    if ctx.invoked_subcommand:
        _fire_telemetry(f"event.{ctx.invoked_subcommand}", extra=None)


def register(event_app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    event_app.callback(invoke_without_command=True)(_event_callback)
