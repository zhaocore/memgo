"""机器可读命令帮助。"""

from __future__ import annotations

import json as _json

import typer
from rich.console import Console

from memgo_cli import __version__
from memgo_cli.cli.help import _build_help_json
from memgo_cli.output.branding import BRAND_COLOR

console = Console()
err_console = Console(stderr=True)


def help(
    json: bool = typer.Option(False, "--json", help="Output machine-readable JSON for LLM agents."),
) -> None:
    """显示文本帮助或机器可读命令描述。"""
    from memgo_cli.runtime.state import is_agent_mode

    if json or is_agent_mode():
        console.print_json(_json.dumps(_build_help_json()))
    else:
        console.print(
            f"[{BRAND_COLOR}]◆ memgo CLI[/] v{__version__} — The Memory Layer for AI Agents\n"
        )
        console.print("Usage: memgo <command> [OPTIONS]\n")
        console.print("[bold]Commands:[/]")
        console.print("  add              Add a memory from text, messages, file, or stdin")
        console.print("  search           Query your memory store (semantic, keyword, hybrid)")
        console.print("  get              Get a specific memory by ID")
        console.print("  list             List memories with optional filters")
        console.print("  update           Update a memory's text or metadata")
        console.print("  delete           Delete a memory, all memories, or an entity")
        console.print("  import           Import memories from a JSON file")
        console.print("  config           Manage configuration (show, get, set)")
        console.print("  entity           Manage entities (list, delete)")
        console.print("  event            Inspect background events (list, status)")
        console.print("  init             Interactive setup wizard")
        console.print("  status           Check connectivity and authentication")
        console.print()
        console.print("  memgo <command> --help    Get help for a command")
        console.print("  memgo help --json         Machine-readable help (for LLM agents)")
        console.print()


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        rich_help_panel="Management",
        help="Show help. Use --json for machine-readable output (for LLM agents).\n\nExamples:\n  memgo help\n  memgo help --json",
    )(help)
