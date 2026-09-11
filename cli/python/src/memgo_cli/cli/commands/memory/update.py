"""记忆 update 用例。"""

from __future__ import annotations

import json
import time as _time

import typer

from memgo_cli.backend.json import json_object
from rich.console import Console

from memgo_cli.application.memory.expiration import _validate_expires
from memgo_cli.backend.types import Backend
from memgo_cli.output.branding import (
    print_error,
    print_success,
    timed_status,
)
from memgo_cli.output.format import (
    format_agent_envelope,
    format_json,
)

console = Console()
err_console = Console(stderr=True)


def cmd_update(
    backend: Backend,
    memory_id: str,
    text: str | None,
    *,
    metadata: str | None,
    expires: str | None,
    timestamp: int | None,
    output: str,
) -> None:
    """更新指定记忆的内容或元数据。"""
    from memgo_cli.runtime.state import is_agent_mode, set_current_command

    set_current_command("update")
    if is_agent_mode():
        output = "agent"
    meta = None
    if metadata:
        try:
            meta = json_object(json.loads(metadata))
        except ValueError:
            print_error(err_console, "Invalid JSON in --metadata.", hint=None)
            raise typer.Exit(1) from None

    if expires:
        _validate_expires(expires)

    _start = _time.perf_counter()
    with timed_status(err_console, "Updating memory...") as _ts:
        try:
            result = backend.update(
                memory_id,
                content=text,
                metadata=meta,
                expiration_date=expires,
                timestamp=timestamp,
            )
        except Exception as e:
            print_error(err_console, str(e), hint=None)
            raise typer.Exit(1) from None
    _elapsed = _time.perf_counter() - _start

    if output == "agent":
        format_agent_envelope(
            console,
            command="update",
            data=result,
            duration_ms=int(_elapsed * 1000),
            scope=None,
            count=None,
        )
    elif output == "json":
        format_json(console, result)
    elif output != "quiet":
        print_success(console, f"Memory {memory_id[:8]} updated ({_elapsed:.2f}s)")
