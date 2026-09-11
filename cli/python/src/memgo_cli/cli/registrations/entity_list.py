"""命令 entity_list 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.cli.context import _get_backend

console = Console()
err_console = Console(stderr=True)


def entity_list(
    entity_type: str = typer.Argument(..., help="Entity type: users, agents, apps, runs."),
    output: str = typer.Option(
        "table", "--output", "-o", help="Output: table, json.", rich_help_panel="Output"
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
    """按实体类型列出实体。"""
    from memgo_cli.cli.commands.entities import cmd_entities_list

    backend = _get_backend(api_key, base_url)
    cmd_entities_list(backend, entity_type, output=output)


def register(entity_app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    entity_app.command(
        "list",
        help="List all entities of a given type.\n\nExamples:\n  memgo entity list users\n  memgo entity list agents -o json",
    )(entity_list)
