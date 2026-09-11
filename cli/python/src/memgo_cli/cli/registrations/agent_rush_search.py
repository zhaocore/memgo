"""命令 agent_rush_search 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

console = Console()
err_console = Console(stderr=True)


def agent_rush_search(
    query: str = typer.Argument(..., help="Search query."),
) -> None:
    """查询 AGENTRUSH 公共记忆。"""
    from memgo_cli.cli.commands.agent_rush import run_agent_rush_search

    run_agent_rush_search(query)


def register(agent_rush_app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    agent_rush_app.command(
        name="search",
        help='Search AGENTRUSH memories.\n\nExample:\n  memgo agent-rush search "constraint satisfaction"',
    )(agent_rush_search)
