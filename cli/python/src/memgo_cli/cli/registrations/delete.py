"""命令 delete 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.application.entity_ids import _resolve_ids
from memgo_cli.cli.context import _fire_telemetry, _get_backend, _get_backend_and_config
from memgo_cli.output.branding import print_error

console = Console()
err_console = Console(stderr=True)


def delete(
    memory_id: str | None = typer.Argument(
        None, help="Memory ID to delete (omit when using --all or --entity)."
    ),
    all_: bool = typer.Option(False, "--all", help="Delete all memories matching scope filters."),
    entity: bool = typer.Option(
        False, "--entity", help="Delete the entity itself and all its memories (cascade)."
    ),
    project: bool = typer.Option(
        False, "--project", help="With --all: delete ALL memories project-wide."
    ),
    dry_run: bool = typer.Option(
        False, "--dry-run", help="Show what would be deleted without deleting."
    ),
    force: bool = typer.Option(False, "--force", help="Skip confirmation."),
    delete_linked: bool = typer.Option(
        False, "--delete-linked", help="Also delete memories linked to this memory."
    ),
    user_id: str | None = typer.Option(
        None, "--user-id", "-u", help="Scope to user.", rich_help_panel="Scope"
    ),
    agent_id: str | None = typer.Option(
        None, "--agent-id", help="Scope to agent.", rich_help_panel="Scope"
    ),
    app_id: str | None = typer.Option(
        None, "--app-id", help="Scope to app.", rich_help_panel="Scope"
    ),
    run_id: str | None = typer.Option(
        None, "--run-id", help="Scope to run.", rich_help_panel="Scope"
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
    """校验删除选项互斥关系并分发单条、范围或实体删除。"""
    # 校验选项互斥关系
    modes = sum([memory_id is not None, all_, entity])
    if modes > 1:
        print_error(
            err_console,
            "Only one of memory ID, --all, or --entity may be used at a time.",
            hint=None,
        )
        raise typer.Exit(1)
    if modes == 0:
        print_error(
            err_console,
            "Provide a memory ID, --all, or --entity.",
            hint="Run 'memgo delete --help' for usage.",
        )
        raise typer.Exit(1)

    # 按删除目标分发
    if memory_id is not None:
        _fire_telemetry("delete", {"delete_mode": "single"})
        from memgo_cli.cli.commands.memory import cmd_delete

        backend = _get_backend(api_key, base_url)
        cmd_delete(
            backend,
            memory_id,
            dry_run=dry_run,
            force=force,
            delete_linked=delete_linked,
            output=output,
        )

    elif all_:
        _fire_telemetry("delete", {"delete_mode": "all"})
        from memgo_cli.cli.commands.memory import cmd_delete_all

        backend, config = _get_backend_and_config(api_key, base_url)
        ids = _resolve_ids(config, user_id=user_id, agent_id=agent_id, app_id=app_id, run_id=run_id)
        cmd_delete_all(backend, force=force, dry_run=dry_run, all_=project, **ids, output=output)

    else:  # 实体删除分支
        _fire_telemetry("delete", {"delete_mode": "entity"})
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


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        rich_help_panel="Memory",
        help="Delete a memory, all memories, or an entity.\n\nExamples:\n  memgo delete abc-123-def-456\n  memgo delete abc-123 --dry-run\n  memgo delete --all -u alice --force\n  memgo delete --all --project --force\n  memgo delete --entity -u alice --force",
    )(delete)
