"""记忆 search 用例。"""

from __future__ import annotations

import json
import time as _time

import typer
from rich.console import Console

from memgo_cli.backend.json import json_object
from memgo_cli.backend.types import Backend
from memgo_cli.output.branding import (
    print_error,
    print_info,
    timed_status,
)
from memgo_cli.output.format import (
    format_agent_envelope,
    format_json,
    format_memories_table,
    format_memories_text,
    print_result_summary,
)

console = Console()
err_console = Console(stderr=True)


def cmd_search(
    backend: Backend,
    query: str,
    *,
    user_id: str | None,
    agent_id: str | None,
    app_id: str | None,
    run_id: str | None,
    top_k: int,
    threshold: float,
    rerank: bool,
    keyword: bool,
    filter_json: str | None,
    fields: str | None,
    show_expired: bool,
    reference_date: str | None,
    latest_only: bool,
    output: str,
) -> None:
    """检索记忆并显示结果。"""
    from memgo_cli.runtime.state import is_agent_mode, set_current_command

    set_current_command("search")
    if is_agent_mode():
        output = "agent"
    filters = None
    if filter_json:
        try:
            filters = json_object(json.loads(filter_json))
        except ValueError:
            print_error(err_console, "Invalid JSON in --filter.", hint=None)
            raise typer.Exit(1) from None

    field_list = None
    if fields:
        field_list = [f.strip() for f in fields.split(",")]

    if top_k < 1:
        print_error(err_console, "--top-k must be >= 1.", hint=None)
        raise typer.Exit(1)
    if not (0.0 <= threshold <= 1.0):
        print_error(err_console, "--threshold must be between 0.0 and 1.0.", hint=None)
        raise typer.Exit(1)

    _start = _time.perf_counter()
    with timed_status(err_console, "Searching memories...") as _ts:
        try:
            results = backend.search(
                query,
                user_id=user_id,
                agent_id=agent_id,
                app_id=app_id,
                run_id=run_id,
                top_k=top_k,
                threshold=threshold,
                rerank=rerank,
                keyword=keyword,
                filters=filters,
                fields=field_list,
                show_expired=show_expired,
                reference_date=reference_date,
                latest_only=latest_only,
            )
        except Exception as e:
            print_error(err_console, str(e), hint=None)
            raise typer.Exit(1) from None
    _elapsed = _time.perf_counter() - _start

    if output == "quiet":
        return

    if output == "agent":
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
            command="search",
            data=results,
            scope=scope or None,
            count=len(results),
            duration_ms=int(_elapsed * 1000),
        )
        return

    if output == "json":
        format_json(console, results)
    elif output == "table":
        if results:
            format_memories_table(console, results, show_score=True)
            print_result_summary(
                console,
                len(results),
                duration_secs=_elapsed,
                user_id=user_id,
                agent_id=agent_id,
                page=None,
            )
        else:
            console.print()
            print_info(console, "No memories found matching your query.")
            console.print()
    else:
        if results:
            format_memories_text(console, results, title="memories")
            print_result_summary(
                console,
                len(results),
                duration_secs=_elapsed,
                user_id=user_id,
                agent_id=agent_id,
                page=None,
            )
        else:
            console.print()
            print_info(console, "No memories found matching your query.")
            console.print()
