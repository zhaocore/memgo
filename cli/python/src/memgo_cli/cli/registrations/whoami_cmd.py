"""命令 whoami_cmd 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

console = Console()
err_console = Console(stderr=True)


def whoami_cmd() -> None:
    """显示当前代理默认用户标识。"""
    from memgo_cli.cli.commands.whoami import run_whoami

    run_whoami()


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        name="whoami",
        rich_help_panel="Setup",
        help="Print your AGENTRUSH identifier (default_user_id).\n\nExample:\n  memgo whoami",
    )(whoami_cmd)
