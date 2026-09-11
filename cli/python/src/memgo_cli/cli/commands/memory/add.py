"""记忆 add 用例。"""

from __future__ import annotations

import json
import sys
from pathlib import Path

import typer

from memgo_cli.backend.json import json_object, json_messages
from rich.console import Console

from memgo_cli.application.memory.expiration import _validate_expires
from memgo_cli.backend.json import JsonObject, json_records, json_string
from memgo_cli.backend.types import Backend
from memgo_cli.output.branding import (
    print_error,
    print_scope,
    print_success,
    timed_status,
)
from memgo_cli.output.format import (
    format_add_result,
    format_agent_envelope,
)
from memgo_cli.runtime.input import _stdin_is_piped

console = Console()
err_console = Console(stderr=True)


def cmd_add(
    backend: Backend,
    text: str | None,
    *,
    user_id: str | None,
    agent_id: str | None,
    app_id: str | None,
    run_id: str | None,
    messages: str | None,
    file: Path | None,
    metadata: str | None,
    immutable: bool,
    no_infer: bool,
    expires: str | None,
    categories: str | None,
    custom_instructions: str | None,
    agent_custom_instructions: str | None,
    custom_categories: str | None,
    structured_data_schema: str | None,
    timestamp: int | None,
    output: str,
) -> None:
    """读取输入并添加记忆，按输出模式呈现处理结果。"""
    from memgo_cli.runtime.state import is_agent_mode, set_current_command

    set_current_command("add")
    if is_agent_mode():
        output = "agent"

    if categories:
        print_error(
            err_console,
            "--categories is not supported on add. Use --custom-categories instead.",
            hint=None,
        )
        raise typer.Exit(1)

    msgs = None
    content = text

    # 读取文件内容
    if file:
        try:
            raw = Path(file).read_text()
            msgs = json_messages(json.loads(raw))
        except (OSError, ValueError) as e:
            print_error(err_console, f"Failed to read file: {e}", hint=None)
            raise typer.Exit(1) from None

    # 解析消息 JSON
    elif messages:
        try:
            msgs = json_messages(json.loads(messages))
        except ValueError as e:
            print_error(err_console, f"Invalid JSON in --messages: {e}", hint=None)
            raise typer.Exit(1) from None

    # 仅在真实管道或文件重定向时读取标准输入
    elif not content and _stdin_is_piped():
        content = sys.stdin.read().strip()

    if not content and not msgs:
        print_error(
            err_console,
            "No content provided. Pass text, --messages, --file, or pipe via stdin.",
            hint=None,
        )
        raise typer.Exit(1)

    meta = None
    if metadata:
        try:
            meta = json_object(json.loads(metadata))
        except ValueError:
            print_error(err_console, "Invalid JSON in --metadata.", hint=None)
            raise typer.Exit(1) from None

    custom_cats = None
    if custom_categories:
        try:
            custom_cats = json_records(json.loads(custom_categories))
        except ValueError:
            print_error(err_console, "Invalid JSON in --custom-categories.", hint=None)
            raise typer.Exit(1) from None

    schema = None
    if structured_data_schema:
        try:
            schema = json_object(json.loads(structured_data_schema))
        except ValueError:
            print_error(err_console, "Invalid JSON in --structured-data-schema.", hint=None)
            raise typer.Exit(1) from None

    if expires:
        _validate_expires(expires)

    with timed_status(err_console, "Adding memory...") as ts:
        try:
            result = backend.add(
                content=content,
                messages=msgs,
                user_id=user_id,
                agent_id=agent_id,
                app_id=app_id,
                run_id=run_id,
                metadata=meta,
                immutable=immutable,
                infer=not no_infer,
                expires=expires,
                custom_instructions=custom_instructions,
                agent_custom_instructions=agent_custom_instructions,
                custom_categories=custom_cats,
                structured_data_schema=schema,
                timestamp=timestamp,
            )
        except Exception as e:
            ts.error_msg = str(e)
            raise typer.Exit(1) from None

    if output == "quiet":
        return

    # 所有输出模式统一按 event_id 合并 PENDING 项
    results_list = json_records(
        result if isinstance(result, list) else result.get("results", [result])
    )
    seen_events: set[str] = set()
    deduped: list[JsonObject] = []
    for r in results_list:
        if r.get("status") == "PENDING":
            eid = json_string(r.get("event_id", ""), "event_id")
            if eid and eid in seen_events:
                continue
            if eid:
                seen_events.add(eid)
        deduped.append(r)
    # 更新局部结果，供后续格式化使用
    if isinstance(result, dict) and "results" in result:
        result = {**result, "results": deduped}
    else:
        result = deduped

    if output == "agent":
        scope = {
            k: v
            for k, v in {
                "user_id": user_id,
                "agent_id": agent_id,
                "app_id": app_id,
                "run_id": run_id,
            }.items()
            if v
        }
        format_agent_envelope(
            console,
            command="add",
            data=deduped,
            scope=scope or None,
            count=len(deduped),
            duration_ms=None,
        )
        return

    if output == "json":
        format_add_result(console, result, output)
        return

    console.print()
    print_scope(console, user_id=user_id, agent_id=agent_id, app_id=app_id, run_id=run_id)
    count = len(deduped)
    all_pending = count > 0 and all(r.get("status") == "PENDING" for r in deduped)
    if all_pending:
        print_success(
            console,
            f"Memory queued — {count} event{'s' if count != 1 else ''} pending",
        )
    else:
        print_success(
            console, f"Memory processed — {count} memor{'y' if count == 1 else 'ies'} extracted"
        )
    format_add_result(console, result, output)
