"""命令 config_set 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

console = Console()
err_console = Console(stderr=True)


def config_set(
    key: str = typer.Argument(..., help="Config key (e.g. platform.api_key)."),
    value: str = typer.Argument(..., help="Value to set."),
) -> None:
    """校验并保存指定配置项。"""
    from memgo_cli.cli.commands.config import cmd_config_set

    cmd_config_set(key, value)


def register(config_app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    config_app.command(
        "set",
        help="Set a configuration value.\n\nExamples:\n  memgo config set defaults.user_id alice\n  memgo config set platform.base_url https://custom.api.memgo.ai",
    )(config_set)
