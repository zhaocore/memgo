"""代理账号创建和邮箱验证码认领用例。"""

from __future__ import annotations

import json
import sys
from copy import deepcopy
from datetime import datetime, timezone

import httpx
import typer
from rich.console import Console
from rich.prompt import Prompt

from memgo_cli.backend.auth import bootstrap_agent, send_email_code, verify_email_code
from memgo_cli.backend.json import JsonObject, json_object, json_string
from memgo_cli.config.models import MemGoConfig
from memgo_cli.config.store import save_config
from memgo_cli.output.branding import (
    BRAND_COLOR,
    DIM_COLOR,
    print_error,
    print_success,
)

console = Console()
err_console = Console(stderr=True)


def _validate_envelope(envelope: object) -> None:
    """拒绝缺少非空密钥或默认用户 ID 的初始化响应。"""
    if not isinstance(envelope, dict):
        print_error(err_console, "Bootstrap response was not a JSON object.", hint=None)
        raise typer.Exit(1)
    for field in ("api_key", "default_user_id"):
        value = envelope.get(field)
        if not isinstance(value, str) or not value:
            print_error(
                err_console,
                f"Bootstrap response missing required field {field!r} — please update the CLI.",
                hint=None,
            )
            raise typer.Exit(1)


def bootstrap_via_backend(
    config: MemGoConfig,
    *,
    source: str | None,
    agent_caller: str | None,
) -> None:
    """创建代理账号并保存独立配置副本。

    source 和 agent_caller 按原协议传递；输入配置不修改，失败时退出。
    """
    config = deepcopy(config)
    base_url = (config.platform.base_url or "https://api.memgo.ai").rstrip("/")
    body: JsonObject = {}
    if source:
        body["source"] = source
    if agent_caller:
        body["agent_caller"] = agent_caller

    try:
        resp = bootstrap_agent(base_url, body)
    except httpx.HTTPError as exc:
        print_error(err_console, f"Network error contacting MemGo: {exc}", hint=None)
        raise typer.Exit(1) from exc

    if resp.status_code == 429:
        print_error(err_console, "Rate-limited. Try again in a few minutes.", hint=None)
        raise typer.Exit(1)
    if resp.status_code == 503:
        print_error(err_console, "Agent Mode is temporarily disabled. Try again later.", hint=None)
        raise typer.Exit(1)
    if resp.status_code != 200:
        detail = resp.text
        try:
            err_body = resp.json()
            detail = err_body.get("error") or err_body.get("detail") or resp.text
        except (json.JSONDecodeError, ValueError, AttributeError):
            pass
        # 后端限流装饰器抛出 PermissionDenied
        # DRF 将其转换为通用 403 权限错误
        # 此处按既有协议显示每日注册限额提示
        if resp.status_code == 403 and "permission" in str(detail).lower():
            print_error(
                err_console,
                "Daily Agent Mode signup limit reached for this network (5/day). Try again from a different IP or after midnight UTC.",
                hint=None,
            )
            raise typer.Exit(1)
        print_error(err_console, f"Bootstrap failed: {detail}", hint=None)
        raise typer.Exit(1)

    envelope = resp.json()
    _validate_envelope(envelope)
    config.platform.api_key = envelope["api_key"]
    config.platform.base_url = base_url
    config.platform.agent_mode = True
    config.platform.created_via = "agent_mode"
    config.platform.agent_caller = agent_caller or ""
    config.platform.claimed_at = ""
    config.platform.default_user_id = envelope["default_user_id"]
    # 将初始化返回的用户 ID 作为默认记忆范围
    config.defaults.user_id = envelope["default_user_id"]
    save_config(config)

    from memgo_cli.output.format import format_json_envelope
    from memgo_cli.runtime.state import capture_notice, is_agent_mode

    if is_agent_mode():
        notice = envelope.get("memgo_notice")
        capture_notice(None if notice is None else json_string(notice, "memgo_notice"))
        format_json_envelope(
            console,
            command="init",
            data={
                "api_key_saved": True,
                "agent_mode": True,
                "default_user_id": envelope["default_user_id"],
            },
            duration_ms=None,
            scope=None,
            count=None,
            status="success",
            error=None,
        )
        return

    print_success(console, f"Agent Mode active. Default user_id: {envelope['default_user_id']}")
    notice = envelope.get("memgo_notice")
    if notice:
        console.print(f"\n[yellow]🔔 {notice}[/yellow]\n")
    else:
        # 兼容尚未返回统一通知的后端版本
        claim_cmd = envelope.get("claim_command", "memgo init --email <your-email>")
        console.print(f"  [{DIM_COLOR}]To claim this account later: {claim_cmd}[/]")

    if not agent_caller:
        console.print(
            f"  [{DIM_COLOR}]If you are an AI agent, identify yourself: "
            f"`memgo identify <your-name>` (e.g. claude-code, cursor).[/]"
        )


def claim_via_otp(config: MemGoConfig, *, email: str, code: str | None) -> None:
    """通过邮箱验证码认领代理账号并保存配置副本。

    验证请求携带原代理密钥；成功后更新认领状态和邮箱，密钥不变。
    """
    config = deepcopy(config)
    base_url = (config.platform.base_url or "https://api.memgo.ai").rstrip("/")
    if not config.platform.api_key or not config.platform.agent_mode:
        print_error(
            err_console,
            "This command requires an active Agent Mode config. Run `memgo init` first.",
            hint=None,
        )
        raise typer.Exit(1)

    raw_key = config.platform.api_key

    if not code:
        send = send_email_code(base_url, email)
        if send.status_code == 429:
            print_error(err_console, "Too many attempts. Try again in a few minutes.", hint=None)
            raise typer.Exit(1)
        if send.status_code != 200:
            try:
                detail = send.json().get("error", send.text)
            except (ValueError, AttributeError):
                detail = send.text
            print_error(err_console, f"Failed to send code: {detail}", hint=None)
            raise typer.Exit(1)

        print_success(console, f"Verification code sent to {email}. Check your inbox.")

        if not sys.stdin.isatty():
            print_error(
                err_console,
                "No --code provided and terminal is non-interactive.",
                hint=f"Re-run: memgo init --email {email} --code <code>",
            )
            raise typer.Exit(1)

        console.print()
        code = Prompt.ask(f"  [{BRAND_COLOR}]Verification Code[/]")
        if not code:
            print_error(err_console, "Code is required.", hint=None)
            raise typer.Exit(1)

    # 一次请求完成验证码验证和账号认领
    verify = verify_email_code(base_url, email, code, raw_key)

    if verify.status_code != 200:
        try:
            err_body = verify.json()
            detail = err_body.get("error", verify.text)
            code_str = err_body.get("code", "")
        except (json.JSONDecodeError, ValueError, AttributeError):
            detail = verify.text
            code_str = ""
        print_error(err_console, f"Claim failed: {detail}", hint=None)
        if code_str == "email_already_claimed":
            console.print(
                f"  [{DIM_COLOR}]Tip: this email already has a MemGo account. Sign in at app.memgo.ai with your existing credentials.[/]"
            )
        raise typer.Exit(1)

    claim_body = json_object(verify.json())
    if claim_body.get("claimed") is not True:
        print_error(err_console, "Unexpected verify response: claimed must be true", hint=None)
        raise typer.Exit(1)

    config.platform.agent_mode = False
    config.platform.claimed_at = json_string(
        claim_body.get("claimed_at") or _utcnow_iso(), "claimed_at"
    )
    config.platform.user_email = email
    config.platform.created_via = "email"
    save_config(config)
    from memgo_cli.output.format import format_json_envelope
    from memgo_cli.runtime.state import is_agent_mode

    if is_agent_mode():
        format_json_envelope(
            console,
            command="init",
            data={"claimed": True, "agent_mode": False, "user_email": email},
            duration_ms=None,
            scope=None,
            count=None,
            status="success",
            error=None,
        )
        return
    print_success(console, f"Agent claimed to {email}. Your API key is unchanged.")


def _utcnow_iso() -> str:
    """返回当前 UTC 时间的 ISO 文本。"""
    return datetime.now(timezone.utc).isoformat()
