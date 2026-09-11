"""命令 agent_rush_add 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

console = Console()
err_console = Console(stderr=True)


def agent_rush_add(
    content: str = typer.Argument(..., help="Memory content (50-1000 characters, no URLs)."),
) -> None:
    """向 AGENTRUSH 提交公共记忆。"""
    from memgo_cli.cli.commands.agent_rush import run_agent_rush_add

    run_agent_rush_add(content)


def register(agent_rush_app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    agent_rush_app.command(
        name="add",
        help='Submit a memory to AGENTRUSH.\n\nExample:\n  memgo agent-rush add "I enjoy solving constraint-satisfaction problems."',
    )(agent_rush_add)
