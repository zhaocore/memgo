"""命令行入口与统一退出处理。"""

from __future__ import annotations

import json
import sys

import httpx
from rich.console import Console

from memgo_cli.backend.errors import APIError, AuthError, NotFoundError
from memgo_cli.cli.program import create_app
from memgo_cli.output.branding import print_error


def main() -> None:
    """规范化全局 JSON 选项，执行独立命令树且不修改进程参数。"""
    args = sys.argv[1:]
    flags = {"--json"} if "init" in args else {"--json", "--agent"}
    if any(arg in flags for arg in args):
        args = ["--json", *[arg for arg in args if arg not in flags]]
    try:
        create_app()(args=args, prog_name="memgo")
    except (
        APIError,
        AuthError,
        NotFoundError,
        httpx.HTTPError,
        ValueError,
        OSError,
        EOFError,
    ) as error:
        if "--json" in args:
            command_args = [arg for arg in args if arg != "--json"]
            command = command_args[0] if command_args else ""
            if command in ("config", "entity", "event", "agent-rush") and len(command_args) > 1:
                command = f"{command} {command_args[1]}"
            print(
                json.dumps(
                    {"status": "error", "command": command, "error": str(error), "data": None}
                )
            )
        else:
            print_error(Console(stderr=True), str(error), hint=None)
        raise SystemExit(1) from None
