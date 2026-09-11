"""记忆 delete 用例。"""

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
    format_single_memory,
)

console = Console()
err_console = Console(stderr=True)


def cmd_delete(
    backend: Backend,
    memory_id: str,
    *,
    dry_run: bool,
    force: bool,
    delete_linked: bool,
    output: str,
) -> None:
    """按 ID 删除单条记忆，支持预览和确认。"""
    from memgo_cli.runtime.state import is_agent_mode, set_current_command

    set_current_command("delete")
    if is_agent_mode():
        output = "agent"
    if dry_run:
        # 读取并显示预期删除的记忆
        try:
            mem = backend.get(memory_id)
        except Exception as e:
            print_error(err_console, str(e), hint=None)
            raise typer.Exit(1) from None
        format_single_memory(console, mem, output)
        print_info(console, "No changes made (dry run).")
        return

    _start = _time.perf_counter()
    with timed_status(err_console, "Deleting...") as _ts:
        try:
            result = backend.delete(
                memory_id=memory_id,
                delete_linked=delete_linked,
                all=False,
                user_id=None,
                agent_id=None,
                app_id=None,
                run_id=None,
            )
        except Exception as e:
            print_error(err_console, str(e), hint=None)
            raise typer.Exit(1) from None
    _elapsed = _time.perf_counter() - _start

    if output == "agent":
        format_agent_envelope(
            console,
            command="delete",
            data={"id": memory_id, "deleted": True},
            duration_ms=int(_elapsed * 1000),
            scope=None,
            count=None,
        )
    elif output == "json":
        format_json(console, result)
    elif output != "quiet":
        print_success(console, f"Memory {memory_id[:8]} deleted ({_elapsed:.2f}s)")
