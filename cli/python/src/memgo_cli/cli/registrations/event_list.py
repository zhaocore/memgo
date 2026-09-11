"""命令 event_list 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.cli.context import _get_backend

console = Console()
err_console = Console(stderr=True)


def event_list(
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
    """解析输出选项并列出后台事件。"""
    from memgo_cli.cli.commands.events import cmd_event_list

    backend = _get_backend(api_key, base_url)
    cmd_event_list(backend, output=output)


def register(event_app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    event_app.command(
        "list",
        help="List recent background processing events.\n\nExamples:\n  memgo event list\n  memgo event list -o json",
    )(event_list)
