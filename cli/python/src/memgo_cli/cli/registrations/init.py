"""初始化入口分发及账号配置。"""

from __future__ import annotations

import typer
from rich.console import Console

console = Console()
err_console = Console(stderr=True)


def init(
    api_key: str | None = typer.Option(None, "--api-key", help="API key (skip prompt)."),
    user_id: str | None = typer.Option(
        None, "--user-id", "-u", help="Default user ID (skip prompt)."
    ),
    email: str | None = typer.Option(None, "--email", help="Login via email verification code."),
    code: str | None = typer.Option(
        None, "--code", help="Verification code (use with --email for non-interactive login)."
    ),
    force: bool = typer.Option(
        False, "--force", help="Overwrite existing config without confirmation."
    ),
    agent_signal: bool = typer.Option(
        False, "--agent", help="Bootstrap an unattended Agent Mode account (no email required)."
    ),
    source: str | None = typer.Option(
        None,
        "--source",
        help="Channel attribution for signup (e.g. github, hn, ph).",
    ),
    agent_caller: str | None = typer.Option(
        None,
        "--agent-caller",
        help="Self-declared agent identity (e.g. claude-code, cursor). Used with --agent to attribute Agent Mode signups.",
    ),
) -> None:
    """解析初始化选项并执行账号配置流程。"""
    from memgo_cli.application.onboarding.init import run_init

    run_init(
        api_key=api_key,
        user_id=user_id,
        email=email,
        code=code,
        force=force,
        source=source,
        agent=agent_signal,
        agent_caller=agent_caller,
    )


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.command(
        rich_help_panel="Management",
        help="Interactive setup wizard for memgo CLI.\n\nExamples:\n  memgo init\n  memgo init --api-key m0-xxx --user-id alice\n  memgo init --email alice@company.com\n  memgo init --email alice@company.com --code 482901\n  memgo init --agent --agent-caller claude-code   # AI agent self-identifies on Agent Mode bootstrap\n  memgo init --email alice@company.com  # Claims an existing Agent Mode key when one is present",
    )(init)
