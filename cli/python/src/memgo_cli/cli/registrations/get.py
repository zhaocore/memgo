"""命令 get 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.cli.context import _get_backend

console = Console()
err_console = Console(stderr=True)


def get(
    memory_id: str = typer.Argument(..., help="Memory ID to retrieve."),
    output: str = typer.Option(
        "text", "--output", "-o", help="Output: text, json.", rich_help_panel="Output"
    ),
    api_key: str | None = typer.Option(
        None,
        "--api-key",
        help="Override API key.",
        envvar="MEMGO_API_KEY",
        rich_help_panel="Connection",
    ),
    base_url: str | None = typer.Option(
        None, "--base-url", help="Override API base URL.", rich_help_panel="Connection"
    ),
) -> None:
    """按 ID 获取单条记忆。"""
    from memgo_cli.cli.commands.memory import cmd_get

    backend = _get_backend(api_key, base_url)
    cmd_get(backend, memory_id, output=output)


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        rich_help_panel="Memory",
        help="Get a specific memory by ID.\n\nExamples:\n  memgo get abc-123-def-456\n  memgo get abc-123-def-456 -o json",
    )(get)
