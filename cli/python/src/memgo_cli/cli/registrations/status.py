"""命令 status 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.cli.context import _get_backend_and_config

console = Console()
err_console = Console(stderr=True)


def status(
    output: str = typer.Option(
        "text", "--output", "-o", help="Output: text, json.", rich_help_panel="Output"
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
    """通过 ping 接口检查连接和鉴权状态。"""
    from memgo_cli.cli.commands.utils import cmd_status

    backend, config = _get_backend_and_config(api_key, base_url)
    cmd_status(
        backend,
        user_id=config.defaults.user_id or None,
        agent_id=config.defaults.agent_id or None,
        output=output,
    )


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        rich_help_panel="Management",
        help="Check connectivity and authentication.\n\nExamples:\n  memgo status\n  memgo status -o json",
    )(status)
