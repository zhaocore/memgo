"""命令 config_get 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

console = Console()
err_console = Console(stderr=True)


def config_get(
    key: str = typer.Argument(..., help="Config key (e.g. platform.api_key)."),
) -> None:
    """读取指定配置项。"""
    from memgo_cli.cli.commands.config import cmd_config_get

    cmd_config_get(key)


def register(config_app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    config_app.command(
        "get",
        help="Get a configuration value.\n\nExamples:\n  memgo config get platform.api_key\n  memgo config get defaults.user_id",
    )(config_get)
