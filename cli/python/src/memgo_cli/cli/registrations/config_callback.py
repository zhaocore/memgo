"""命令 config_callback 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.cli.context import _fire_telemetry

console = Console()
err_console = Console(stderr=True)


def _config_callback(ctx: typer.Context) -> None:
    """处理配置命令组入口。"""
    if ctx.invoked_subcommand:
        _fire_telemetry(f"config.{ctx.invoked_subcommand}", extra=None)


def register(config_app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    config_app.callback(invoke_without_command=True)(_config_callback)
