"""命令依赖装配、鉴权预检和遥测触发。"""

from __future__ import annotations

from dataclasses import replace

import typer
from rich.console import Console

from memgo_cli.backend.json import JsonObject
from memgo_cli.backend.types import Backend, BackendContext
from memgo_cli.config.models import MemGoConfig
from memgo_cli.output.branding import print_error
from memgo_cli.runtime.state import (
    caller_type,
    capture_notice,
    register_cleanup,
    set_validated_email,
    validated_email,
)
from memgo_cli.runtime.warning import warn_optional

console = Console()
err_console = Console(stderr=True)


def _fire_telemetry(command_name: str, extra: JsonObject | None) -> None:
    """异步发送命令遥测，可选集成失败时记录告警。"""
    try:
        from memgo_cli.integrations.telemetry import capture_event

        props: JsonObject = {"command": command_name}
        if extra:
            props.update(extra)
        capture_event(f"cli.{command_name}", props, pre_resolved_email=validated_email())
    except (OSError, ValueError) as error:
        warn_optional("telemetry", error)


def _get_backend_and_config(
    api_key: str | None,
    base_url: str | None,
) -> tuple[Backend, MemGoConfig]:
    """加载配置并创建连接器，预检凭据且缓存平台邮箱。"""
    from memgo_cli.backend import get_backend
    from memgo_cli.backend.errors import AuthError
    from memgo_cli.config.store import load_config, save_config

    config = load_config()

    if api_key:
        config = replace(config, platform=replace(config.platform, api_key=api_key))
    if base_url:
        config = replace(config, platform=replace(config.platform, base_url=base_url))

    if not config.platform.api_key:
        print_error(
            err_console,
            "No API key configured.",
            hint="Run 'memgo init' or set MEMGO_API_KEY environment variable.",
        )
        raise typer.Exit(1)

    backend = get_backend(config, BackendContext(caller_type, capture_notice))
    register_cleanup(backend.close)

    # 使用短超时预检密钥
    try:
        ping_data = backend.ping(timeout=5.0)
        email = ping_data.get("user_email") if isinstance(ping_data, dict) else None
        if isinstance(email, str) and email:
            set_validated_email(email)
            if config.platform.user_email != email:
                config = replace(config, platform=replace(config.platform, user_email=email))
                save_config(config)
    except AuthError:
        print_error(
            err_console,
            "Invalid or expired API key.",
            hint="Run 'memgo init' or set MEMGO_API_KEY environment variable.",
        )
        raise typer.Exit(1) from None

    return backend, config


def _get_backend(
    api_key: str | None,
    base_url: str | None,
) -> Backend:
    """创建已完成凭据预检的 Platform 连接器。"""
    backend, _config = _get_backend_and_config(api_key, base_url)
    return backend
