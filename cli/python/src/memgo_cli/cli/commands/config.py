"""配置显示、读取和更新命令。"""

from __future__ import annotations

from rich.console import Console
from rich.table import Table

from memgo_cli.config.resolve import get_nested_value, redact_key, set_nested_value
from memgo_cli.config.store import load_config, save_config
from memgo_cli.output.branding import (
    ACCENT_COLOR,
    BRAND_COLOR,
    DIM_COLOR,
    print_error,
    print_success,
)

console = Console()
err_console = Console(stderr=True)


def cmd_config_show(*, output: str) -> None:
    """显示已脱敏的当前配置。"""
    from memgo_cli.output.format import format_agent_envelope
    from memgo_cli.runtime.state import is_agent_mode, set_current_command

    set_current_command("config show")
    if is_agent_mode():
        output = "agent"

    config = load_config()

    if output in ("json", "agent"):
        format_agent_envelope(
            console,
            command="config show",
            data={
                "defaults": {
                    "user_id": config.defaults.user_id or None,
                    "agent_id": config.defaults.agent_id or None,
                    "app_id": config.defaults.app_id or None,
                    "run_id": config.defaults.run_id or None,
                },
                "platform": {
                    "api_key": redact_key(config.platform.api_key),
                    "base_url": config.platform.base_url,
                },
            },
            duration_ms=None,
            scope=None,
            count=None,
        )
        return

    console.print()
    console.print(f"  [{BRAND_COLOR}]◆ memgo Configuration[/]\n")

    table = Table(border_style=BRAND_COLOR, header_style=f"bold {ACCENT_COLOR}", padding=(0, 2))
    table.add_column("Key", style="bold")
    table.add_column("Value")

    # 默认范围
    table.add_row(
        "defaults.user_id",
        config.defaults.user_id or f"[{DIM_COLOR}](not set)[/]",
    )
    table.add_row(
        "defaults.agent_id",
        config.defaults.agent_id or f"[{DIM_COLOR}](not set)[/]",
    )
    table.add_row(
        "defaults.app_id",
        config.defaults.app_id or f"[{DIM_COLOR}](not set)[/]",
    )
    table.add_row(
        "defaults.run_id",
        config.defaults.run_id or f"[{DIM_COLOR}](not set)[/]",
    )
    table.add_row("", "")

    # 平台配置
    table.add_row("[bold]platform.api_key[/]", redact_key(config.platform.api_key))
    table.add_row("platform.base_url", config.platform.base_url)

    console.print(table)
    console.print()


def cmd_config_get(key: str) -> None:
    """读取并显示指定配置项。"""
    from memgo_cli.output.format import format_agent_envelope
    from memgo_cli.runtime.state import is_agent_mode, set_current_command

    set_current_command("config get")
    config = load_config()
    value = get_nested_value(config, key)

    if value is None:
        print_error(err_console, f"Unknown config key: {key}", hint=None)
        return

    display_value = (
        redact_key(str(value)) if ("api_key" in key or "key" in key.split(".")[-1:]) else str(value)
    )

    if is_agent_mode():
        format_agent_envelope(
            console,
            command="config get",
            data={"key": key, "value": display_value},
            duration_ms=None,
            scope=None,
            count=None,
        )
    else:
        console.print(display_value)


def cmd_config_set(key: str, value: str) -> None:
    """校验配置项并保存更新后的配置。"""
    from memgo_cli.output.format import format_agent_envelope
    from memgo_cli.runtime.state import is_agent_mode, set_current_command

    set_current_command("config set")
    config = load_config()
    try:
        updated = set_nested_value(config, key, value)
    except ValueError:
        print_error(err_console, f"Unknown or invalid config key: {key}", hint=None)
        return
    if updated:
        save_config(updated)
        display = redact_key(value) if "key" in key else value
        if is_agent_mode():
            format_agent_envelope(
                console,
                command="config set",
                data={"key": key, "value": display},
                duration_ms=None,
                scope=None,
                count=None,
            )
        else:
            print_success(console, f"{key} = {display}")
    else:
        print_error(err_console, f"Unknown config key: {key}", hint=None)
