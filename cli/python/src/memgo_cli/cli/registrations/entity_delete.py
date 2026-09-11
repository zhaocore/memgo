"""命令 entity_delete 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.cli.context import _get_backend

console = Console()
err_console = Console(stderr=True)


def entity_delete(
    user_id: str | None = typer.Option(
        None, "--user-id", "-u", help="User ID.", rich_help_panel="Scope"
    ),
    agent_id: str | None = typer.Option(
        None, "--agent-id", help="Agent ID.", rich_help_panel="Scope"
    ),
    app_id: str | None = typer.Option(None, "--app-id", help="App ID.", rich_help_panel="Scope"),
    run_id: str | None = typer.Option(None, "--run-id", help="Run ID.", rich_help_panel="Scope"),
    force: bool = typer.Option(False, "--force", help="Skip confirmation."),
    dry_run: bool = typer.Option(
        False, "--dry-run", help="Show what would be deleted without deleting."
    ),
    output: str = typer.Option(
        "text", "--output", "-o", help="Output: text, json, quiet.", rich_help_panel="Output"
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
    """解析范围和确认选项并删除实体。"""
    from memgo_cli.cli.commands.entities import cmd_entities_delete

    backend = _get_backend(api_key, base_url)
    cmd_entities_delete(
        backend,
        user_id=user_id,
        agent_id=agent_id,
        app_id=app_id,
        run_id=run_id,
        force=force,
        dry_run=dry_run,
        output=output,
    )


def register(entity_app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    entity_app.command(
        "delete",
        help="Delete an entity and ALL its memories (cascade).\n\nExamples:\n  memgo entity delete --user-id alice --force\n  memgo entity delete -u alice --dry-run",
    )(entity_delete)
