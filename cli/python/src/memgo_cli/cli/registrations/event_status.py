"""命令 event_status 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.cli.context import _get_backend

console = Console()
err_console = Console(stderr=True)


def event_status(
    event_id: str = typer.Argument(..., help="Event ID to inspect."),
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
    """解析事件 ID 并查询处理状态。"""
    from memgo_cli.cli.commands.events import cmd_event_status

    backend = _get_backend(api_key, base_url)
    cmd_event_status(backend, event_id, output=output)


def register(event_app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    event_app.command(
        "status",
        help="Check the status of a specific background event.\n\nExamples:\n  memgo event status <event-id>\n  memgo event status <event-id> -o json",
    )(event_status)
