"""命令 main_callback 的参数声明与注册。"""

from __future__ import annotations

import typer
from rich.console import Console

from memgo_cli.cli.context import _fire_telemetry

console = Console()
err_console = Console(stderr=True)


def main_callback(
    ctx: typer.Context,
    version: bool = typer.Option(False, "--version", help="Show version and exit."),
    json_agent: bool = typer.Option(
        False,
        "--json",
        "--agent",
        help="Output as JSON for agent/programmatic use.",
        is_eager=False,
    ),
) -> None:
    """初始化独立调用状态并登记退出清理。"""
    from memgo_cli.runtime.state import finish_invocation, start_invocation

    token = start_invocation(json_agent)
    ctx.call_on_close(lambda: finish_invocation(token))
    if version:
        from memgo_cli.cli.commands.utils import cmd_version

        _fire_telemetry("version", extra=None)
        cmd_version()
        raise typer.Exit()
    if ctx.invoked_subcommand:
        # 记录当前命令名
        # 使代理模式错误信封能定位失败命令
        # 避免 command 字段为空
        from memgo_cli.runtime.state import set_current_command

        set_current_command(ctx.invoked_subcommand)
    if ctx.invoked_subcommand and ctx.invoked_subcommand != "init":
        # 初始化用例自行发送包含完整属性的遥测
        _fire_telemetry(ctx.invoked_subcommand, extra=None)


def register(app: typer.Typer) -> None:
    """注册当前命令及其参数。"""
    app.callback(invoke_without_command=True)(main_callback)
