"""命令 version 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

console = Console()
err_console = Console(stderr=True)


def version() -> None:
    """显示版本并结束命令。"""
    from memgo_cli.cli.commands.utils import cmd_version

    cmd_version()


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        rich_help_panel="Management", help="Show version and exit.\n\nExample:\n  memgo version"
    )(version)
