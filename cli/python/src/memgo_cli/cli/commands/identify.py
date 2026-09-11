"""声明当前代理模式密钥的调用方名称。"""

from __future__ import annotations

import httpx
import typer
from rich.console import Console

from memgo_cli.backend.auth import identify_agent
from memgo_cli.backend.json import json_object, json_string
from memgo_cli.config.store import load_config, save_config
from memgo_cli.output.branding import print_error, print_success

console = Console()
err_console = Console(stderr=True)

_SOURCE_HEADERS = {
    "X-MemGo-Source": "cli",
    "X-MemGo-Client-Language": "python",
}


def run_identify(name: str) -> None:
    """更新当前代理模式密钥的调用方名称。"""
    config = load_config()
    if not config.platform.api_key:
        print_error(
            err_console, "No API key configured. Run `memgo init --agent` first.", hint=None
        )
        raise typer.Exit(1)
    if not config.platform.agent_mode:
        print_error(err_console, "This command only works on unclaimed agent-mode keys.", hint=None)
        raise typer.Exit(1)

    name = (name or "").strip()
    if not name:
        print_error(err_console, "Agent name is required.", hint=None)
        raise typer.Exit(1)

    base_url = (config.platform.base_url or "https://api.memgo.ai").rstrip("/")
    try:
        resp = identify_agent(base_url, config.platform.api_key, name)
    except httpx.HTTPError as exc:
        print_error(err_console, f"Network error: {exc}", hint=None)
        raise typer.Exit(1) from exc

    if resp.status_code != 200:
        try:
            detail = resp.json().get("error", resp.text)
        except (ValueError, AttributeError):
            detail = resp.text
        print_error(err_console, f"Identify failed: {detail}", hint=None)
        raise typer.Exit(1)

    canonical = json_string(json_object(resp.json()).get("agent_caller"), "agent_caller")
    config.platform.agent_caller = canonical
    save_config(config)
    print_success(console, f"Identified as {canonical}.")
