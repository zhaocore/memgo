"""命令 config_show 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

console = Console()
err_console = Console(stderr=True)


def config_show(
    output: str = typer.Option(
        "text", "--output", "-o", help="Output: text, json.", rich_help_panel="Output"
    ),
) -> None:
    """显示脱敏配置。"""
    from memgo_cli.cli.commands.config import cmd_config_show

    cmd_config_show(output=output)


def register(config_app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    config_app.command(
        "show",
        help="Display current configuration (secrets redacted).\n\nExamples:\n  memgo config show\n  memgo config show -o json",
    )(config_show)
