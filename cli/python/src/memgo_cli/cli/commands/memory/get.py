"""记忆 get 用例。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.backend.types import Backend
from memgo_cli.output.branding import (
    print_error,
    timed_status,
)
from memgo_cli.output.format import (
    format_agent_envelope,
    format_single_memory,
)

console = Console()
err_console = Console(stderr=True)


def cmd_get(backend: Backend, memory_id: str, *, output: str) -> None:
    """读取并显示指定 ID 的记忆。"""
    from memgo_cli.runtime.state import is_agent_mode, set_current_command

    set_current_command("get")
    if is_agent_mode():
        output = "agent"
    with timed_status(err_console, "Fetching memory...") as _ts:
        try:
            result = backend.get(memory_id)
        except Exception as e:
            print_error(err_console, str(e), hint=None)
            raise typer.Exit(1) from None

    if output == "agent":
        format_agent_envelope(
            console, command="get", data=result, duration_ms=None, scope=None, count=None
        )
    else:
        format_single_memory(console, result, output)
