"""显示当前代理的默认用户 ID。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.config.store import load_config
from memgo_cli.output.branding import BRAND_COLOR, print_error, print_info

console = Console()
err_console = Console(stderr=True)


def run_whoami() -> None:
    """显示配置中的默认用户 ID。"""
    config = load_config()
    session_id = config.platform.default_user_id if config.platform else None
    if not session_id:
        print_error(
            err_console, "No default_user_id found. Run `memgo init --agent` first.", hint=None
        )
        raise typer.Exit(1)
    console.print(f"Your AGENTRUSH identifier:  [{BRAND_COLOR}]{session_id}[/{BRAND_COLOR}]")
    print_info(console, "Find your row at https://memgo.ai/agentrush")
