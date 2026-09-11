"""命令 add 的参数声明与注册。"""

from __future__ import annotations

from pathlib import Path

import typer
from rich.console import Console

from memgo_cli.application.entity_ids import _resolve_ids
from memgo_cli.cli.context import _get_backend_and_config

console = Console()
err_console = Console(stderr=True)


def add(
    text: str | None = typer.Argument(None, help="Text content to add as a memory."),
    user_id: str | None = typer.Option(
        None, "--user-id", "-u", help="Scope to user.", rich_help_panel="Scope"
    ),
    agent_id: str | None = typer.Option(
        None, "--agent-id", help="Scope to agent.", rich_help_panel="Scope"
    ),
    app_id: str | None = typer.Option(
        None, "--app-id", help="Scope to app.", rich_help_panel="Scope"
    ),
    run_id: str | None = typer.Option(
        None, "--run-id", help="Scope to run.", rich_help_panel="Scope"
    ),
    messages: str | None = typer.Option(None, "--messages", help="Conversation messages as JSON."),
    file: Path | None = typer.Option(None, "--file", "-f", help="Read messages from JSON file."),
    metadata: str | None = typer.Option(None, "--metadata", "-m", help="Custom metadata as JSON."),
    immutable: bool = typer.Option(False, "--immutable", help="Prevent future updates."),
    no_infer: bool = typer.Option(False, "--no-infer", help="Skip inference, store raw."),
    expires: str | None = typer.Option(None, "--expires", help="Expiration date (YYYY-MM-DD)."),
    categories: str | None = typer.Option(
        None, "--categories", help="Not supported on add, use --custom-categories instead."
    ),
    custom_instructions: str | None = typer.Option(
        None, "--custom-instructions", help="Custom instructions for fact extraction."
    ),
    agent_custom_instructions: str | None = typer.Option(
        None,
        "--agent-custom-instructions",
        help="Extraction instructions for agent-scoped memories, overriding the project setting.",
    ),
    custom_categories: str | None = typer.Option(
        None,
        "--custom-categories",
        help="Custom categories as a JSON array of {name: description} objects.",
    ),
    structured_data_schema: str | None = typer.Option(
        None, "--structured-data-schema", help="Schema for structured data extraction, as JSON."
    ),
    timestamp: int | None = typer.Option(
        None, "--timestamp", help="Unix timestamp for the memory."
    ),
    output: str = typer.Option(
        "text", "--output", "-o", help="Output format: text, json, quiet.", rich_help_panel="Output"
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
    """解析内容、消息和文件选项并添加记忆。"""
    from memgo_cli.cli.commands.memory import cmd_add

    backend, config = _get_backend_and_config(api_key, base_url)
    ids = _resolve_ids(config, user_id=user_id, agent_id=agent_id, app_id=app_id, run_id=run_id)

    cmd_add(
        backend,
        text,
        **ids,
        messages=messages,
        file=file,
        metadata=metadata,
        immutable=immutable,
        no_infer=no_infer,
        expires=expires,
        categories=categories,
        custom_instructions=custom_instructions,
        agent_custom_instructions=agent_custom_instructions,
        custom_categories=custom_categories,
        structured_data_schema=structured_data_schema,
        timestamp=timestamp,
        output=output,
    )


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        rich_help_panel="Memory",
        help='Add a memory from text, messages, file, or stdin.\n\nExamples:\n  memgo add "I prefer dark mode" --user-id alice\n  echo "text" | memgo add -u alice\n  memgo add --file msgs.json -u alice -o json',
    )(add)
