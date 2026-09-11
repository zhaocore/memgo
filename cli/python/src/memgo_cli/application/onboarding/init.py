"""初始化入口分发及账号配置。"""

from __future__ import annotations

import os
import sys

import httpx
import typer
from rich.console import Console
from rich.prompt import Prompt

from memgo_cli.application.onboarding.email import _email_login, _validate_email
from memgo_cli.application.onboarding.key import _ping_key
from memgo_cli.application.onboarding.setup import (
    _setup_defaults,
    _setup_platform,
    _validate_platform,
)
from memgo_cli.backend.auth import identify_reused_key
from memgo_cli.backend.json import JsonObject, json_object, json_string
from memgo_cli.config.models import DEFAULT_BASE_URL, MemGoConfig
from memgo_cli.config.store import CONFIG_FILE, load_config, save_config
from memgo_cli.output.branding import (
    BRAND_COLOR,
    DIM_COLOR,
    print_banner,
    print_error,
    print_info,
    print_success,
)
from memgo_cli.runtime.warning import warn_optional

console = Console()
err_console = Console(stderr=True)


def run_init(
    *,
    api_key: str | None,
    user_id: str | None,
    email: str | None,
    code: str | None,
    force: bool,
    source: str | None,
    agent: bool,
    agent_caller: str | None,
) -> None:
    """执行初始化、密钥复用或邮箱认领流程。

    邮箱加代理配置进入认领；代理环境优先复用现有密钥，无可用密钥才创建账号。
    显式密钥和用户 ID 跳过交互，否则按终端类型收集输入。
    """
    from memgo_cli.application.onboarding.agent_mode import bootstrap_via_backend, claim_via_otp
    from memgo_cli.integrations.agent_detect import detect_agent_caller
    from memgo_cli.integrations.telemetry import capture_event
    from memgo_cli.runtime.state import is_agent_mode as _global_agent_mode

    def _fire_init(mode: str, *, claimed: bool) -> None:
        """发送包含初始化方式和认领状态的遥测。"""
        props: JsonObject = {"command": "init", "mode": mode}
        if agent_caller:
            # 身份来自 --agent-caller 显式声明，不从环境变量推测
            props["agent_caller"] = agent_caller
        if source:
            props["signup_source"] = source
        if claimed:
            props["claimed_agent_mode"] = True
        capture_event("cli.init", props, pre_resolved_email=None)

    config = MemGoConfig()

    base_url = os.environ.get("MEMGO_BASE_URL", config.platform.base_url or DEFAULT_BASE_URL)
    config.platform.base_url = base_url

    if code and not email:
        print_error(err_console, "--code requires --email.", hint=None)
        raise typer.Exit(1)

    # 邮箱与已有代理账号进入认领流程
    if email and CONFIG_FILE.exists():
        existing = load_config()
        if existing.platform.agent_mode and existing.platform.api_key:
            email = email.strip().lower()
            _validate_email(email)
            print_info(console, f"Claiming Agent Mode account to {email}...")
            claim_via_otp(existing, email=email, code=code)
            _fire_init("email", claimed=True)
            return

    # 代理模式在配置覆盖确认前处理
    # 优先复用有效密钥
    # 复用时不触发覆盖确认
    # 只有没有有效密钥时才创建账号
    _agent_ctx = agent or _global_agent_mode() or (detect_agent_caller() is not None)
    if not api_key and not email and _agent_ctx:
        from memgo_cli.output.format import format_json_envelope
        from memgo_cli.runtime.state import is_agent_mode as _is_json_mode

        def _emit_reuse(source: str) -> None:
            """按输出模式显示既有密钥复用结果。"""
            if _is_json_mode():
                format_json_envelope(
                    console,
                    command="init",
                    data={
                        "api_key_saved": False,
                        "api_key_source": source,
                        "agent_mode": False,
                        "message": "Existing MemGo API key found and reused. No Agent Mode key was created.",
                    },
                    duration_ms=None,
                    scope=None,
                    count=None,
                    status="success",
                    error=None,
                )
            else:
                msg = (
                    "Existing MEMGO_API_KEY is valid; reusing it. No new Agent Mode key was minted."
                    if source == "env"
                    else "Existing API key in config is valid; reusing it. No new Agent Mode key was minted."
                )
                print_success(console, msg)

        def _maybe_identify(key: str) -> None:
            """同步复用密钥的代理名称；可选更新失败时告警，不中断复用。"""
            if not agent_caller:
                return
            try:
                resp = identify_reused_key(base_url, key, agent_caller)
                # 同时更新本地配置中的调用方名称
                if resp.status_code == 200 and CONFIG_FILE.exists():
                    try:
                        cfg = load_config()
                        cfg.platform.agent_caller = json_string(
                            json_object(resp.json()).get("agent_caller"), "agent_caller"
                        )
                        save_config(cfg)
                    except (OSError, ValueError) as error:
                        warn_optional("identify_config", error)
                elif resp.status_code != 200:
                    warn_optional("identify_reused_key", ValueError(f"HTTP {resp.status_code}"))
            except httpx.HTTPError as error:
                warn_optional("identify_reused_key", error)

        # 优先复用环境变量中的有效密钥
        _env_key = (os.environ.get("MEMGO_API_KEY") or "").strip()
        if _env_key and _ping_key(_env_key, base_url, timeout=5.0):
            _maybe_identify(_env_key)
            _emit_reuse("env")
            _fire_init("existing_key", claimed=False)
            return
        # 其次复用配置文件中的有效密钥
        if CONFIG_FILE.exists():
            _existing = load_config()
            if _existing.platform.api_key and _ping_key(
                _existing.platform.api_key, base_url, timeout=5.0
            ):
                _maybe_identify(_existing.platform.api_key)
                _emit_reuse("config")
                _fire_init("existing_key", claimed=False)
                return
        # 没有有效密钥时创建代理账号
        # 调用方身份仅来自 --agent-caller
        # 环境变量仅用于识别代理上下文
        # 不自动填充调用方身份
        bootstrap_via_backend(config, source=source, agent_caller=agent_caller)
        _fire_init("agent", claimed=False)
        return

    # 覆盖已有密钥配置前显示确认
    if not force and CONFIG_FILE.exists():
        existing = load_config()
        if existing.platform.api_key:
            from memgo_cli.config.resolve import redact_key

            console.print(
                f"\n  [{BRAND_COLOR}]Existing configuration found[/] "
                f"[{DIM_COLOR}](API key: {redact_key(existing.platform.api_key)})[/]"
            )
            if sys.stdin.isatty():
                confirm = typer.confirm("  Overwrite existing config? This cannot be undone.")
                if not confirm:
                    print_info(console, "Cancelled. Use --force to skip this check.")
                    raise typer.Exit(0)
            else:
                print_error(
                    err_console,
                    "Existing config would be overwritten.",
                    hint="Use --force to overwrite.",
                )
                raise typer.Exit(1)

    # 邮箱登录流程
    if email:
        if api_key:
            print_error(err_console, "Cannot use both --api-key and --email.", hint=None)
            raise typer.Exit(1)

        email = email.strip().lower()
        _validate_email(email)

        print_banner(console)
        console.print()
        print_info(console, f"Logging in as {email}...\n")

        result = _email_login(email, code, base_url)

        api_key_val = result.get("api_key")
        if not isinstance(api_key_val, str) or not api_key_val:
            print_error(
                err_console,
                "Auth succeeded but no API key was returned. Contact support.",
                hint=None,
            )
            raise typer.Exit(1)
        config.platform.api_key = api_key_val
        config.platform.base_url = base_url
        config.platform.user_email = email
        config.platform.created_via = "email"
        config.defaults.user_id = (
            user_id or os.environ.get("USER") or os.environ.get("USERNAME") or "memgo-cli"
        )

        save_config(config)

        console.print()
        print_success(console, "Authenticated! Configuration saved to ~/.memgo/config.json")
        console.print()
        console.print(f"  [{DIM_COLOR}]Get started:[/]")
        console.print(f'  [{DIM_COLOR}]  memgo add "I prefer dark mode"[/]')
        console.print(f'  [{DIM_COLOR}]  memgo search "preferences"[/]')
        console.print()
        return

    # API 密钥配置流程
    # 代理模式已在前面处理
    # 有效密钥复用不会触发覆盖确认

    # 非交互环境补齐默认值，支持管道和 CI
    if not sys.stdin.isatty():
        if not api_key:
            print_error(
                err_console,
                "Non-interactive terminal detected and --api-key is required.",
                hint="Run: memgo init --api-key <key>, --email <addr>, or --agent for unattended Agent Mode bootstrap.",
            )
            raise typer.Exit(1)
        user_id = user_id or os.environ.get("USER") or os.environ.get("USERNAME") or "memgo-cli"

    # 密钥和用户 ID 均已提供时跳过交互
    if api_key and user_id:
        config.platform.api_key = api_key
        config.platform.created_via = "api_key"
        config.defaults.user_id = user_id
        config = _validate_platform(config)
        save_config(config)
        print_success(console, "Configuration saved to ~/.memgo/config.json")
        return

    print_banner(console)
    console.print()
    print_info(console, "Welcome! Let's set up your memgo CLI.\n")

    # 没有密钥参数时询问认证方式
    if not api_key:
        console.print(f"  [{BRAND_COLOR}]How would you like to authenticate?[/]")
        console.print(f"  [{DIM_COLOR}]1.[/] Login with email [{DIM_COLOR}](recommended)[/]")
        console.print(f"  [{DIM_COLOR}]2.[/] Enter API key manually")
        console.print()
        choice = Prompt.ask(f"  [{BRAND_COLOR}]Choose[/]", choices=["1", "2"], default="1")

        if choice == "1":
            console.print()
            email_addr = Prompt.ask(f"  [{BRAND_COLOR}]Email[/]")
            if not email_addr:
                print_error(err_console, "Email is required.", hint=None)
                raise typer.Exit(1)

            email_addr = email_addr.strip().lower()
            _validate_email(email_addr)
            print_info(console, f"Logging in as {email_addr}...\n")

            result = _email_login(email_addr, None, base_url)

            api_key_val = result.get("api_key")
            if not isinstance(api_key_val, str) or not api_key_val:
                print_error(
                    err_console,
                    "Auth succeeded but no API key was returned. Contact support.",
                    hint=None,
                )
                raise typer.Exit(1)
            config.platform.api_key = api_key_val
            config.platform.base_url = base_url
            config.platform.user_email = email_addr
            config.platform.created_via = "email"
            config.defaults.user_id = (
                user_id or os.environ.get("USER") or os.environ.get("USERNAME") or "memgo-cli"
            )

            save_config(config)

            console.print()
            print_success(console, "Authenticated! Configuration saved to ~/.memgo/config.json")
            console.print()
            console.print(f"  [{DIM_COLOR}]Get started:[/]")
            console.print(f'  [{DIM_COLOR}]  memgo add "I prefer dark mode"[/]')
            console.print(f'  [{DIM_COLOR}]  memgo search "preferences"[/]')
            console.print()
            return

    # API 密钥流程
    if api_key:
        config.platform.api_key = api_key
        config.platform.created_via = "api_key"
    else:
        config = _setup_platform(config)

    if user_id:
        config.defaults.user_id = user_id
    else:
        config = _setup_defaults(config)

    config = _validate_platform(config)

    save_config(config)
    console.print()
    print_success(console, "Configuration saved to ~/.memgo/config.json")
    console.print()
    console.print(f"  [{DIM_COLOR}]Get started:[/]")
    if config.defaults.user_id:
        console.print(f'  [{DIM_COLOR}]  memgo add "I prefer dark mode"[/]')
        console.print(f'  [{DIM_COLOR}]  memgo search "preferences"[/]')
    else:
        console.print(f'  [{DIM_COLOR}]  memgo add "I prefer dark mode" --user-id alice[/]')
        console.print(f'  [{DIM_COLOR}]  memgo search "preferences" --user-id alice[/]')
    console.print()
