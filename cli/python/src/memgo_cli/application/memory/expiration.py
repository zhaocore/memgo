"""记忆过期日期校验。"""

from __future__ import annotations

import re
from datetime import date

import typer
from rich.console import Console

from memgo_cli.output.branding import (
    print_error,
)

console = Console()
err_console = Console(stderr=True)


def _validate_expires(value: str) -> None:
    """校验未来的 YYYY-MM-DD 日期，无效时明确退出。"""
    if not re.match(r"^\d{4}-\d{2}-\d{2}$", value):
        print_error(
            err_console,
            "Invalid date format for --expires. Use YYYY-MM-DD (e.g. 2025-12-31).",
            hint=None,
        )
        raise typer.Exit(1)
    if date.fromisoformat(value) <= date.today():
        print_error(err_console, "--expires date must be in the future.", hint=None)
        raise typer.Exit(1)
