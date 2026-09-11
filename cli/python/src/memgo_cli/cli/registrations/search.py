"""命令 search 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.application.entity_ids import _resolve_ids
from memgo_cli.cli.context import _get_backend_and_config
from memgo_cli.output.branding import print_error
from memgo_cli.runtime.input import _read_stdin

console = Console()
err_console = Console(stderr=True)


def search(
    query: str | None = typer.Argument(None, help="Search query."),
    user_id: str | None = typer.Option(
        None, "--user-id", "-u", help="Filter by user.", rich_help_panel="Scope"
    ),
    agent_id: str | None = typer.Option(
        None, "--agent-id", help="Filter by agent.", rich_help_panel="Scope"
    ),
    app_id: str | None = typer.Option(
        None, "--app-id", help="Filter by app.", rich_help_panel="Scope"
    ),
    run_id: str | None = typer.Option(
        None, "--run-id", help="Filter by run.", rich_help_panel="Scope"
    ),
    top_k: int = typer.Option(
        10, "--top-k", "-k", "--limit", help="Number of results.", rich_help_panel="Search"
    ),
    threshold: float = typer.Option(
        0.3, "--threshold", help="Minimum similarity score.", rich_help_panel="Search"
    ),
    rerank: bool = typer.Option(
        False, "--rerank", help="Enable reranking (Platform only).", rich_help_panel="Search"
    ),
    keyword: bool = typer.Option(
        False, "--keyword", help="Use keyword search.", rich_help_panel="Search"
    ),
    filter_json: str | None = typer.Option(
        None,
        "--filter",
        help='Advanced filter as JSON: {"AND": [...]} or {"OR": [...]}, '
        'e.g. {"AND": [{"categories": {"in": ["work"]}}]}.',
        rich_help_panel="Search",
    ),
    fields: str | None = typer.Option(
        None,
        "--fields",
        help="Specific fields to return (comma-separated).",
        rich_help_panel="Search",
    ),
    show_expired: bool = typer.Option(
        False, "--show-expired", help="Include expired memories.", rich_help_panel="Search"
    ),
    reference_date: str | None = typer.Option(
        None,
        "--reference-date",
        help="Reference date for relative queries (YYYY-MM-DD or unix timestamp).",
        rich_help_panel="Search",
    ),
    latest_only: bool = typer.Option(
        False,
        "--latest-only",
        help="Only return the latest version of each memory.",
        rich_help_panel="Search",
    ),
    output: str = typer.Option(
        "text", "--output", "-o", help="Output: text, json, table.", rich_help_panel="Output"
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
    """解析检索选项并查询记忆。"""
    from memgo_cli.cli.commands.memory import cmd_search

    # 未提供查询时读取管道输入
    if query is None:
        query = _read_stdin()
    if not query or not query.strip():
        print_error(err_console, "Search query cannot be empty.", hint=None)
        raise typer.Exit(1)

    backend, config = _get_backend_and_config(api_key, base_url)
    ids = _resolve_ids(config, user_id=user_id, agent_id=agent_id, app_id=app_id, run_id=run_id)

    cmd_search(
        backend,
        query,
        **ids,
        top_k=top_k,
        threshold=threshold,
        rerank=rerank,
        keyword=keyword,
        filter_json=filter_json,
        fields=fields,
        show_expired=show_expired,
        reference_date=reference_date,
        latest_only=latest_only,
        output=output,
    )


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        rich_help_panel="Memory",
        help='Query your memory store — semantic, keyword, or hybrid retrieval.\n\nExamples:\n  memgo search "preferences" --user-id alice\n  memgo search "tools" -u alice -o json -k 5\n  echo "preferences" | memgo search -u alice\n  memgo search "invoices" -u alice --filter \'{"AND": [{"categories": {"in": ["work"]}}]}\'',
    )(search)
