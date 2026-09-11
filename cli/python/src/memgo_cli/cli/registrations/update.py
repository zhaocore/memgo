"""命令 update 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.cli.context import _get_backend
from memgo_cli.runtime.input import _read_stdin

console = Console()
err_console = Console(stderr=True)


def update(
    memory_id: str = typer.Argument(..., help="Memory ID to update."),
    text: str | None = typer.Argument(None, help="New memory text."),
    metadata: str | None = typer.Option(None, "--metadata", "-m", help="Update metadata (JSON)."),
    expires: str | None = typer.Option(None, "--expires", help="Expiration date (YYYY-MM-DD)."),
    timestamp: int | None = typer.Option(
        None, "--timestamp", help="Unix timestamp for the memory."
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
    """解析更新选项并执行记忆更新。"""
    from memgo_cli.cli.commands.memory import cmd_update

    # 未提供文本时读取管道输入
    if text is None:
        text = _read_stdin()

    backend = _get_backend(api_key, base_url)
    cmd_update(
        backend,
        memory_id,
        text,
        metadata=metadata,
        expires=expires,
        timestamp=timestamp,
        output=output,
    )


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        rich_help_panel="Memory",
        help='Update a memory\'s text or metadata.\n\nExamples:\n  memgo update abc-123-def-456 "new text"\n  memgo update abc-123 --metadata \'{{"key":"val"}}\'\n  echo "new text" | memgo update abc-123',
    )(update)
