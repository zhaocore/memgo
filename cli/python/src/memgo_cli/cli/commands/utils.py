"""连接状态、版本和文件导入命令。"""

from __future__ import annotations

import json
import time as _time
from pathlib import Path

import typer
from rich.console import Console
from rich.panel import Panel
from rich.progress import track

from memgo_cli import __version__
from memgo_cli.backend.types import Backend
from memgo_cli.output.branding import (
    BRAND_COLOR,
    DIM_COLOR,
    ERROR_COLOR,
    SUCCESS_COLOR,
    print_error,
    print_success,
    timed_status,
)

console = Console()
err_console = Console(stderr=True)


def cmd_status(
    backend: Backend,
    *,
    user_id: str | None,
    agent_id: str | None,
    output: str,
) -> None:
    """显示后端连接和鉴权状态。"""
    from memgo_cli.output.format import format_agent_envelope
    from memgo_cli.runtime.state import is_agent_mode, set_current_command

    set_current_command("status")
    if is_agent_mode():
        output = "agent"

    _start = _time.perf_counter()
    with timed_status(err_console, "Checking connection...") as _ts:
        result = backend.status(user_id=user_id, agent_id=agent_id)
    _elapsed = _time.perf_counter() - _start

    if output in ("json", "agent"):
        format_agent_envelope(
            console,
            command="status",
            data={
                "connected": result.get("connected", False),
                "backend": result.get("backend", "?"),
                "base_url": result.get("base_url", ""),
            },
            duration_ms=int(_elapsed * 1000),
            scope=None,
            count=None,
        )
        return

    lines = []
    if result.get("connected"):
        lines.append(f"  [{SUCCESS_COLOR}]●[/] Connected")
    else:
        lines.append(f"  [{ERROR_COLOR}]●[/] Disconnected")

    lines.append(f"  [{DIM_COLOR}]Backend:[/]  {result.get('backend', '?')}")
    if result.get("base_url"):
        lines.append(f"  [{DIM_COLOR}]API URL:[/]  {result['base_url']}")
    if result.get("error"):
        lines.append(f"  [{ERROR_COLOR}]Error:[/]    {result['error']}")
        if "Authentication failed" in str(result["error"]):
            lines.append("")
            lines.append(
                f"  [{DIM_COLOR}]Run [bold]memgo init[/bold] to reconfigure your API key[/]"
            )
            lines.append(
                f"  [{DIM_COLOR}]Get a key at [bold]https://app.memgo.ai/dashboard/api-keys?utm_source=oss&utm_medium=cli-python[/bold][/]"
            )
    lines.append(f"  [{DIM_COLOR}]Latency:[/]  {_elapsed:.2f}s")

    content = "\n".join(lines)
    panel = Panel(
        content,
        title=f"[{BRAND_COLOR}]Connection Status[/]",
        title_align="left",
        border_style=BRAND_COLOR,
        padding=(1, 1),
    )
    console.print()
    console.print(panel)
    console.print()


def cmd_version() -> None:
    """显示已安装包版本。"""
    console.print(f"  [{BRAND_COLOR}]◆ MemGo[/] CLI v{__version__}")


def cmd_import(
    backend: Backend,
    file_path: str,
    *,
    user_id: str | None,
    agent_id: str | None,
    output: str,
) -> None:
    """从 JSON 文件导入记忆。"""
    from memgo_cli.output.format import format_agent_envelope
    from memgo_cli.runtime.state import is_agent_mode, set_current_command

    set_current_command("import")
    if is_agent_mode():
        output = "agent"

    try:
        data = json.loads(Path(file_path).read_text())
    except (FileNotFoundError, json.JSONDecodeError) as e:
        print_error(err_console, f"Failed to read file: {e}", hint=None)
        raise typer.Exit(1) from None

    if not isinstance(data, list):
        data = [data]

    added = 0
    failed = 0
    _start = _time.perf_counter()
    for item in track(
        data, description=f"[{DIM_COLOR}]Importing memories...[/]", console=err_console
    ):
        content = item.get("memory", item.get("text", item.get("content", "")))
        if not content:
            failed += 1
            continue
        try:
            backend.add(
                content=content,
                user_id=user_id or item.get("user_id"),
                agent_id=agent_id or item.get("agent_id"),
                metadata=item.get("metadata"),
                messages=None,
                app_id=None,
                run_id=None,
                immutable=False,
                infer=True,
                expires=None,
                custom_instructions=None,
                agent_custom_instructions=None,
                custom_categories=None,
                structured_data_schema=None,
                timestamp=None,
            )
            added += 1
        except Exception:
            failed += 1
    _elapsed = _time.perf_counter() - _start

    if output in ("json", "agent"):
        scope = {k: v for k, v in {"user_id": user_id, "agent_id": agent_id}.items() if v}
        format_agent_envelope(
            console,
            command="import",
            data={"added": added, "failed": failed},
            scope=scope or None,
            duration_ms=int(_elapsed * 1000),
            count=None,
        )
        return

    print_success(err_console, f"Imported {added} memories ({_elapsed:.2f}s)")
    if failed:
        print_error(err_console, f"{failed} memories failed to import.", hint=None)
