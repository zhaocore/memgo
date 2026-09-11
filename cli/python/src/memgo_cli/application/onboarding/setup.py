"""交互配置收集与平台验证。"""

from __future__ import annotations

import os
from dataclasses import replace

import typer
from rich.console import Console
from rich.prompt import Prompt

from memgo_cli.backend.json import json_string
from memgo_cli.backend.types import BackendContext
from memgo_cli.config.models import MemGoConfig
from memgo_cli.output.branding import (
    BRAND_COLOR,
    DIM_COLOR,
    print_error,
    print_info,
    print_success,
)
from memgo_cli.runtime.prompt import _prompt_secret
from memgo_cli.runtime.state import caller_type, capture_notice

console = Console()
err_console = Console(stderr=True)


def _setup_platform(config: MemGoConfig) -> MemGoConfig:
    """读取 API 密钥并返回新配置。"""
    console.print()
    console.print(
        f"  [{DIM_COLOR}]Get your API key at https://app.memgo.ai/dashboard/api-keys?utm_source=oss&utm_medium=cli-python[/]"
    )
    console.print()

    api_key = _prompt_secret("  API Key: ")
    if not api_key:
        print_error(err_console, "API key is required.", hint=None)
        raise typer.Exit(1)

    return replace(
        config, platform=replace(config.platform, api_key=api_key, created_via="api_key")
    )


def _setup_defaults(config: MemGoConfig) -> MemGoConfig:
    """收集默认实体 ID 并返回新配置。"""
    console.print()
    print_info(console, "Set default entity IDs (press Enter to skip).\n")

    _default_user = os.environ.get("USER") or os.environ.get("USERNAME") or "memgo-cli"
    user_id = Prompt.ask(
        f"  [{BRAND_COLOR}]Default User ID[/] [{DIM_COLOR}](recommended)[/]",
        default=_default_user,
    )
    return replace(
        config, defaults=replace(config.defaults, user_id=user_id or config.defaults.user_id)
    )


def _validate_platform(config: MemGoConfig) -> MemGoConfig:
    """验证凭据并返回包含平台邮箱的新配置，失败时不保存。"""
    from memgo_cli.backend.platform import PlatformBackend

    console.print()
    print_info(console, "Validating connection...")
    backend = PlatformBackend(config.platform, BackendContext(caller_type, capture_notice))
    try:
        ping_data = backend.ping(timeout=None)
        email = ping_data.get("user_email")
        updated = config
        if email is not None:
            updated = replace(
                config,
                platform=replace(config.platform, user_email=json_string(email, "user_email")),
            )
        print_success(console, "Connected to memgo Platform!")
        return updated
    finally:
        backend.close()
