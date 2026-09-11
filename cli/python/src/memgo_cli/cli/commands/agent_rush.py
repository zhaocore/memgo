"""AGENTRUSH 公共记忆命令及配额提示。"""

from __future__ import annotations

import sys
from datetime import datetime, timezone

import httpx
import typer
from rich.console import Console

from memgo_cli.backend.agent_rush import request_agent_rush
from memgo_cli.backend.json import JsonObject, json_object, json_records
from memgo_cli.config.store import load_config, save_config
from memgo_cli.output.branding import print_error, print_success

console = Console()
err_console = Console(stderr=True)

_PII_WARNING_LINES = (
    "",
    "[yellow]⚠️  AGENTRUSH memories are PUBLIC — visible to any other player.[/yellow]",
    "[yellow]   Do not include real names, emails, secrets, work content, or PII.[/yellow]",
    "",
)


_ERROR_HINTS = {
    "agentrush_search_first": "Run 3 'memgo agent-rush search' commands before adding.",
    "agentrush_search_quota": "You've used your 3 lifetime searches.",
    "agentrush_add_quota": "You've used your 3 lifetime adds.",
    "agentrush_not_agent_mode": "Re-run 'memgo init --agent' to bootstrap an agent-mode key.",
    "agentrush_length": "Memory text must be 50-1000 characters.",
    "agentrush_no_urls": "URLs are not allowed.",
    "agentrush_blocklist": "Content contains a blocked term.",
    "agentrush_global_quota": "Event-wide cap reached. Try again later.",
    "agentrush_not_provisioned": "AGENTRUSH is not provisioned in this environment.",
}


def _call(path: str, body: JsonObject) -> JsonObject:
    """调用 AGENTRUSH 接口并显示配额错误提示。"""
    config = load_config()
    if not config.platform.api_key:
        print_error(err_console, "Not initialized. Run `memgo init --agent` first.", hint=None)
        raise typer.Exit(1)
    try:
        resp = request_agent_rush(config.platform, path, body)
    except httpx.HTTPError as exc:
        print_error(err_console, f"Network error: {exc}", hint=None)
        raise typer.Exit(1) from exc
    try:
        data = resp.json()
    except ValueError as error:
        print_error(
            err_console,
            f"AGENTRUSH {path}: HTTP {resp.status_code}, response is not valid JSON",
            hint=None,
        )
        raise typer.Exit(1) from error
    if resp.status_code >= 400:
        code = (
            json_object(data.get("error") or {}).get("code", "unknown")
            if isinstance(data, dict)
            else "unknown"
        )
        print_error(err_console, f"AGENTRUSH error: {code}", hint=None)
        hint = _ERROR_HINTS.get(str(code))
        if hint:
            console.print(f"  [dim]{hint}[/dim]")
        raise typer.Exit(1)
    return json_object(data)


def _ensure_warning_acknowledged() -> None:
    """首次交互添加前确认公开记忆提示。

    终端确认后保存时间；非交互调用仅向标准错误显示提示，不等待输入。
    """
    config = load_config()
    if config.agent_rush.acknowledged_at:
        return

    is_tty = sys.stdin.isatty() and sys.stdout.isatty()
    if not is_tty:
        for line in _PII_WARNING_LINES:
            err_console.print(line)
        return

    for line in _PII_WARNING_LINES:
        console.print(line)
    answer = typer.prompt("   Continue? [y/N]", default="N", show_default=False).strip().lower()
    if answer not in ("y", "yes"):
        print_error(err_console, "Aborted.", hint=None)
        raise typer.Exit(1)

    config.agent_rush.acknowledged_at = datetime.now(timezone.utc).isoformat()
    save_config(config)


def run_agent_rush_add(content: str) -> None:
    """确认公开提示后提交记忆。"""
    _ensure_warning_acknowledged()
    result = _call("/v1/agent-rush/memories/", {"content": content})
    event_id = result.get("event_id", "?")
    print_success(console, f"Memory submitted (event_id: {event_id})")


def run_agent_rush_search(query: str) -> None:
    """检索并显示最多五条公共记忆。"""
    result = _call("/v1/agent-rush/memories/search/", {"query": query})
    memories = json_records(result.get("results") or result.get("memories") or [])
    if not memories:
        console.print("[dim](no results)[/dim]")
        return
    for i, m in enumerate(memories[:5], start=1):
        text = m.get("memory") if isinstance(m, dict) else str(m)
        console.print(f"  {i}. {text}")
