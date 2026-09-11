"""邮箱格式校验及验证码登录。"""

from __future__ import annotations

import re
import sys

import typer
from rich.console import Console
from rich.prompt import Prompt

from memgo_cli.backend.auth import send_email_code, verify_email_code
from memgo_cli.backend.json import JsonObject, json_object
from memgo_cli.output.branding import (
    BRAND_COLOR,
    print_error,
    print_success,
)

console = Console()
err_console = Console(stderr=True)
_EMAIL_RE = re.compile(r"^[^@\s]+@[^@\s]+\.[^@\s]+$")


def _validate_email(email: str) -> None:
    """邮箱格式无效时显示明确错误并退出。"""
    if not _EMAIL_RE.match(email):
        print_error(err_console, f"Invalid email address: {email!r}", hint=None)
        raise typer.Exit(1)


def _email_login(
    email: str,
    code: str | None,
    base_url: str,
) -> JsonObject:
    """发送并验证邮箱验证码，返回登录响应对象。"""
    url = base_url.rstrip("/")

    if not code:
        # 第一步：请求发送验证码
        resp = send_email_code(url, email)
        if resp.status_code == 429:
            print_error(err_console, "Too many attempts. Try again in a few minutes.", hint=None)
            raise typer.Exit(1)
        if resp.status_code != 200:
            try:
                detail = resp.json().get("error", resp.text)
            except (ValueError, AttributeError):
                detail = resp.text
            print_error(err_console, f"Failed to send code: {detail}", hint=None)
            raise typer.Exit(1)

        print_success(console, "Verification code sent! Check your email.")

        # 第二步：读取用户验证码
        if not sys.stdin.isatty():
            print_error(
                err_console,
                "No --code provided and terminal is non-interactive.",
                hint="Run: memgo init --email <email> --code <code>",
            )
            raise typer.Exit(1)
        console.print()
        code = Prompt.ask(f"  [{BRAND_COLOR}]Verification Code[/]")
        if not code:
            print_error(err_console, "Code is required.", hint=None)
            raise typer.Exit(1)

    # 第三步：验证验证码
    resp = verify_email_code(url, email, code, None)
    if resp.status_code == 429:
        print_error(err_console, "Too many attempts. Try again in a few minutes.", hint=None)
        raise typer.Exit(1)
    if resp.status_code != 200:
        try:
            detail = resp.json().get("error", resp.text)
        except (ValueError, AttributeError):
            detail = resp.text
        print_error(err_console, f"Verification failed: {detail}", hint=None)
        raise typer.Exit(1)

    return json_object(resp.json())
