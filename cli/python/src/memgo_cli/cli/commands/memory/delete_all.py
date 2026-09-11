"""记忆 delete_all 用例。"""

from __future__ import annotations

import time as _time

import typer
from rich.console import Console

from memgo_cli.backend.types import Backend
from memgo_cli.output.branding import (
    print_error,
    print_info,
    print_success,
    timed_status,
)
from memgo_cli.output.format import (
    format_agent_envelope,
    format_json,
)

console = Console()
err_console = Console(stderr=True)


def cmd_delete_all(
    backend: Backend,
    *,
    force: bool,
    dry_run: bool,
    all_: bool,
    user_id: str | None,
    agent_id: str | None,
    app_id: str | None,
    run_id: str | None,
    output: str,
) -> None:
    """删除匹配范围的全部记忆，支持预览和确认。"""
    from memgo_cli.runtime.state import is_agent_mode, set_current_command

    set_current_command("delete-all")
    if is_agent_mode():
        output = "agent"
        if not force:
            print_error(
                err_console, "Destructive operation requires --force in agent mode.", hint=None
            )
            raise typer.Exit(1)
    if all_:
        # 使用通配符实体 ID 执行项目范围删除
        # 平台没有项目范围预览接口，明确拒绝，绝不把预览执行为删除。
        if dry_run:
            print_error(
                err_console,
                "Project-wide --dry-run is not supported; no memories were deleted.",
                hint=None,
            )
            raise typer.Exit(1)

        if not force:
            confirm = typer.confirm(
                "\n  ⚠  Delete ALL memories across the ENTIRE project? This cannot be undone."
            )
            if not confirm:
                print_info(console, "Cancelled.")
                raise typer.Exit(0)

        _start = _time.perf_counter()
        with timed_status(err_console, "Deleting all memories project-wide...") as _ts:
            try:
                result = backend.delete(
                    all=True,
                    user_id="*",
                    agent_id="*",
                    app_id="*",
                    run_id="*",
                    memory_id=None,
                    delete_linked=False,
                )
            except Exception as e:
                print_error(err_console, str(e), hint=None)
                raise typer.Exit(1) from None
        _elapsed = _time.perf_counter() - _start

        if output == "agent":
            format_agent_envelope(
                console,
                command="delete-all",
                data={"deleted": True, "scope": "project"},
                duration_ms=int(_elapsed * 1000),
                scope=None,
                count=None,
            )
        elif output == "json":
            format_json(console, result)
        elif output != "quiet":
            if isinstance(result, dict) and "message" in result:
                print_info(console, "Deletion started. Memories will be removed in the background.")
            else:
                print_success(console, f"All project memories deleted ({_elapsed:.2f}s)")
        return

    if dry_run:
        # 列出匹配记忆并显示数量
        try:
            results = backend.list_memories(
                user_id=user_id,
                agent_id=agent_id,
                app_id=app_id,
                run_id=run_id,
                page=1,
                page_size=100,
                category=None,
                after=None,
                before=None,
                show_expired=False,
                latest_only=False,
            )
        except Exception as e:
            print_error(err_console, str(e), hint=None)
            raise typer.Exit(1) from None
        count = len(results)
        print_info(console, f"Would delete {count} memor{'y' if count == 1 else 'ies'}.")
        print_info(console, "No changes made (dry run).")
        return

    if not force:
        scope_parts = []
        if user_id:
            scope_parts.append(f"user={user_id}")
        if agent_id:
            scope_parts.append(f"agent={agent_id}")
        if app_id:
            scope_parts.append(f"app={app_id}")
        if run_id:
            scope_parts.append(f"run={run_id}")
        scope = ", ".join(scope_parts) if scope_parts else "ALL entities"

        confirm = typer.confirm(f"\n  ⚠  Delete ALL memories for {scope}? This cannot be undone.")
        if not confirm:
            print_info(console, "Cancelled.")
            raise typer.Exit(0)

    _start = _time.perf_counter()
    with timed_status(err_console, "Deleting all memories...") as _ts:
        try:
            result = backend.delete(
                all=True,
                user_id=user_id,
                agent_id=agent_id,
                app_id=app_id,
                run_id=run_id,
                memory_id=None,
                delete_linked=False,
            )
        except Exception as e:
            print_error(err_console, str(e), hint=None)
            raise typer.Exit(1) from None
    _elapsed = _time.perf_counter() - _start

    scope_ids = {
        k: v
        for k, v in {
            "user_id": user_id,
            "agent_id": agent_id,
            "app_id": app_id,
            "run_id": run_id,
        }.items()
        if v
    }
    if output == "agent":
        format_agent_envelope(
            console,
            command="delete-all",
            data={"deleted": True},
            scope=scope_ids or None,
            duration_ms=int(_elapsed * 1000),
            count=None,
        )
    elif output == "json":
        format_json(console, result)
    elif output != "quiet":
        if isinstance(result, dict) and "message" in result:
            print_info(console, "Deletion started. Memories will be removed in the background.")
        else:
            print_success(console, f"All matching memories deleted ({_elapsed:.2f}s)")
