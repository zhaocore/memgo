"""记忆 list 用例。"""

from __future__ import annotations

import time as _time

import typer
from rich.console import Console

from memgo_cli.backend.types import Backend
from memgo_cli.output.branding import (
    print_error,
    print_info,
    timed_status,
)
from memgo_cli.output.format import (
    format_agent_envelope,
    format_memories_table,
    format_memories_text,
    print_result_summary,
)

console = Console()
err_console = Console(stderr=True)


def cmd_list(
    backend: Backend,
    *,
    user_id: str | None,
    agent_id: str | None,
    app_id: str | None,
    run_id: str | None,
    page: int,
    page_size: int,
    category: str | None,
    after: str | None,
    before: str | None,
    show_expired: bool,
    latest_only: bool,
    output: str,
) -> None:
    """列出指定范围的记忆。"""
    from memgo_cli.runtime.state import is_agent_mode, set_current_command

    set_current_command("list")
    if is_agent_mode():
        output = "agent"
    if page_size < 1:
        print_error(err_console, "--page-size must be >= 1.", hint=None)
        raise typer.Exit(1)
    if page < 1:
        print_error(err_console, "--page must be >= 1.", hint=None)
        raise typer.Exit(1)

    _start = _time.perf_counter()
    with timed_status(err_console, "Listing memories...") as _ts:
        try:
            results = backend.list_memories(
                user_id=user_id,
                agent_id=agent_id,
                app_id=app_id,
                run_id=run_id,
                page=page,
                page_size=page_size,
                category=category,
                after=after,
                before=before,
                show_expired=show_expired,
                latest_only=latest_only,
            )
        except Exception as e:
            print_error(err_console, str(e), hint=None)
            raise typer.Exit(1) from None
    _elapsed = _time.perf_counter() - _start

    if output == "quiet":
        return

    if output in ("json", "agent"):
        scope = {
            k: v
            for k, v in {
                "user_id": user_id,
                "agent_id": agent_id,
                "app_id": app_id,
                "run_id": run_id,
            }.items()
            if v
        }
        format_agent_envelope(
            console,
            command="list",
            data=results,
            scope=scope or None,
            count=len(results),
            duration_ms=int(_elapsed * 1000),
        )
    elif output == "table":
        if results:
            format_memories_table(console, results, show_score=False)
            print_result_summary(
                console,
                len(results),
                duration_secs=_elapsed,
                page=page,
                user_id=user_id,
                agent_id=agent_id,
            )
        else:
            console.print()
            print_info(console, "No memories found.")
            console.print()
    else:
        if results:
            format_memories_text(console, results, title="memories")
            print_result_summary(
                console,
                len(results),
                duration_secs=_elapsed,
                page=page,
                user_id=user_id,
                agent_id=agent_id,
            )
        else:
            console.print()
            print_info(console, "No memories found.")
            console.print()
