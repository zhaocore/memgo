"""命令 agent_rush_callback 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.cli.context import _fire_telemetry

console = Console()
err_console = Console(stderr=True)


def _agent_rush_callback(ctx: typer.Context) -> None:
    """处理公共记忆命令组入口。"""
    if ctx.invoked_subcommand:
        _fire_telemetry(f"agent-rush.{ctx.invoked_subcommand}", extra=None)


def register(agent_rush_app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    agent_rush_app.callback(invoke_without_command=True)(_agent_rush_callback)
