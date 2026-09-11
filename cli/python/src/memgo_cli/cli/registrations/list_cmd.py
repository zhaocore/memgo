"""命令 list_cmd 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.application.entity_ids import _resolve_ids
from memgo_cli.cli.context import _get_backend_and_config

console = Console()
err_console = Console(stderr=True)


def list_cmd(
    user_id: str | None = typer.Option(
        None, "--user-id", "-u", help="Filter by user.", rich_help_panel="Scope"
    ),
    agent_id: str | None = typer.Option(
        None, "--agent-id", help="Filter by agent.", rich_help_panel="Scope"
    ),
    app_id: str | None = typer.Option(
        None, "--app-id", help="Filter by app.", rich_help_panel="Scope"
    ),
    run_id: str | None = typer.Option(
        None, "--run-id", help="Filter by run.", rich_help_panel="Scope"
    ),
    page: int = typer.Option(1, "--page", help="Page number.", rich_help_panel="Pagination"),
    page_size: int = typer.Option(
        100, "--page-size", help="Results per page.", rich_help_panel="Pagination"
    ),
    category: str | None = typer.Option(
        None, "--category", help="Filter by category.", rich_help_panel="Filters"
    ),
    after: str | None = typer.Option(
        None, "--after", help="Created after (YYYY-MM-DD).", rich_help_panel="Filters"
    ),
    before: str | None = typer.Option(
        None, "--before", help="Created before (YYYY-MM-DD).", rich_help_panel="Filters"
    ),
    show_expired: bool = typer.Option(
        False, "--show-expired", help="Include expired memories.", rich_help_panel="Filters"
    ),
    latest_only: bool = typer.Option(
        False,
        "--latest-only",
        help="Only return the latest version of each memory.",
        rich_help_panel="Filters",
    ),
    output: str = typer.Option(
        "table", "--output", "-o", help="Output: text, json, table.", rich_help_panel="Output"
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
    """解析筛选条件并列出记忆。"""
    from memgo_cli.cli.commands.memory import cmd_list

    backend, config = _get_backend_and_config(api_key, base_url)
    ids = _resolve_ids(config, user_id=user_id, agent_id=agent_id, app_id=app_id, run_id=run_id)

    cmd_list(
        backend,
        **ids,
        page=page,
        page_size=page_size,
        category=category,
        after=after,
        before=before,
        show_expired=show_expired,
        latest_only=latest_only,
        output=output,
    )


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        name="list",
        rich_help_panel="Memory",
        help="List memories with optional filters.\n\nExamples:\n  memgo list -u alice\n  memgo list --category prefs --after 2024-01-01 -o json",
    )(list_cmd)
