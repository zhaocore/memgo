"""声明当前代理模式密钥的调用方名称。"""

from __future__ import annotations

import typer
from rich.console import Console

console = Console()
err_console = Console(stderr=True)


def identify(
    name: str = typer.Argument(..., help="Agent identity (e.g. claude-code, cursor, my-bot)."),
) -> None:
    """声明当前代理模式密钥的调用方名称。"""
    from memgo_cli.cli.commands.identify import run_identify

    run_identify(name)


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        rich_help_panel="Setup",
        help="Tag your active Agent Mode key with the AI agent that's using it.\n\nRun this once after `memgo init --agent` if you didn't pass --agent-caller.\nIdempotent — re-running just overwrites the value.\n\nExample:\n  memgo identify claude-code",
    )(identify)
