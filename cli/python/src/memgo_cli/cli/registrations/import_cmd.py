"""命令 import_cmd 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.application.entity_ids import _resolve_ids
from memgo_cli.cli.context import _get_backend_and_config

console = Console()
err_console = Console(stderr=True)


def import_cmd(
    file_path: str = typer.Argument(..., help="JSON file to import."),
    user_id: str | None = typer.Option(
        None, "--user-id", "-u", help="Override user ID.", rich_help_panel="Scope"
    ),
    agent_id: str | None = typer.Option(
        None, "--agent-id", help="Override agent ID.", rich_help_panel="Scope"
    ),
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
    """解析文件路径和实体范围并导入记忆。"""
    from memgo_cli.cli.commands.utils import cmd_import

    backend, config = _get_backend_and_config(api_key, base_url)
    ids = _resolve_ids(config, user_id=user_id, agent_id=agent_id, app_id=None, run_id=None)
    cmd_import(backend, file_path, user_id=ids["user_id"], agent_id=ids["agent_id"], output=output)


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        "import",
        rich_help_panel="Management",
        help="Import memories from a JSON file.\n\nExamples:\n  memgo import data.json --user-id alice\n  memgo import data.json -u alice -o json",
    )(import_cmd)
