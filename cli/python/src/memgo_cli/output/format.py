"""文本、表格、JSON 和静默输出格式。"""

from __future__ import annotations

import json
from collections.abc import Mapping
from datetime import datetime

from rich.console import Console
from rich.panel import Panel
from rich.table import Table
from rich.text import Text

from memgo_cli.backend.json import (
    JsonObject,
    JsonValue,
    json_object,
    json_records,
    json_string,
    json_value,
)
from memgo_cli.output.branding import ACCENT_COLOR, BRAND_COLOR, DIM_COLOR, SUCCESS_COLOR, _sym


def format_memories_text(console: Console, memories: list[JsonObject], title: str) -> None:
    """以易读文本显示记忆列表。"""
    count = len(memories)
    console.print(f"\n[{BRAND_COLOR}]Found {count} {title}:[/]\n")

    for i, mem in enumerate(memories, 1):
        memory_text = json_string(mem.get("memory") or mem.get("text") or "", "memory.text")
        mem_id = json_string(mem.get("id") or "", "memory.id")[:8]
        score = mem.get("score")
        created = _format_date(mem.get("created_at"))
        category = mem.get("categories", [None])
        if isinstance(category, list):
            category = category[0] if category else None

        line = Text()
        line.append(f"  {i}. ", style="bold")
        line.append(memory_text, style="white")
        console.print(line)

        details = []
        if score is not None:
            details.append(f"Score: {score:.2f}")
        if mem_id:
            details.append(f"ID: {mem_id}")
        if created:
            details.append(f"Created: {created}")
        if category:
            details.append(f"Category: {category}")

        if details:
            detail_str = " · ".join(details)
            console.print(f"     [{DIM_COLOR}]{detail_str}[/]")
        console.print()


def format_memories_table(
    console: Console, memories: list[JsonObject], *, show_score: bool
) -> None:
    """以 Rich 表格显示记忆列表。"""
    table = Table(
        border_style=BRAND_COLOR,
        header_style=f"bold {ACCENT_COLOR}",
        row_styles=["", "dim"],
        padding=(0, 1),
    )
    table.add_column("ID", style="dim", max_width=38, no_wrap=True)
    if show_score:
        table.add_column("Score", max_width=7, justify="right")
    table.add_column("Memory", max_width=50, no_wrap=False)
    table.add_column("Category", max_width=14)
    table.add_column("Created", max_width=12)

    for mem in memories:
        mem_id = json_string(mem.get("id") or "", "memory.id")
        memory_text = json_string(mem.get("memory") or mem.get("text") or "", "memory.text")
        if len(memory_text) > 60:
            memory_text = memory_text[:57] + "..."
        categories = mem.get("categories", [])
        if isinstance(categories, list) and categories:
            cat = (
                json_string(categories[0], "memory.categories")
                if len(categories) == 1
                else f"{categories[0]} (+{len(categories) - 1})"
            )
        else:
            cat = "—"
        created = _format_date(mem.get("created_at")) or "—"
        if show_score:
            score = mem.get("score")
            score_str = f"{score:.2f}" if score is not None else "—"
            table.add_row(mem_id, score_str, memory_text, cat, created)
        else:
            table.add_row(mem_id, memory_text, cat, created)

    console.print()
    console.print(table)
    console.print()


def format_json(console: Console, data: object) -> None:
    """输出带缩进的 JSON 数据。"""
    console.print_json(json.dumps(data, default=str))


def format_single_memory(console: Console, mem: JsonObject, output: str) -> None:
    """显示单条记忆及其元数据。"""
    if output == "json":
        format_json(console, mem)
        return

    memory_text = json_string(mem.get("memory") or mem.get("text") or "", "memory.text")
    mem_id = json_string(mem.get("id") or "", "memory.id")

    lines = []
    lines.append(f"  [white bold]{memory_text}[/]")
    lines.append("")

    if mem_id:
        lines.append(f"  [{DIM_COLOR}]ID:[/]         {mem_id}")
    created = _format_date(mem.get("created_at"))
    if created:
        lines.append(f"  [{DIM_COLOR}]Created:[/]    {created}")
    updated = _format_date(mem.get("updated_at"))
    if updated:
        lines.append(f"  [{DIM_COLOR}]Updated:[/]    {updated}")
    meta = mem.get("metadata")
    if meta:
        lines.append(f"  [{DIM_COLOR}]Metadata:[/]   {json.dumps(meta)}")
    categories = mem.get("categories")
    if categories:
        cat_str = (
            ", ".join(json_string(item, "memory.categories") for item in categories)
            if isinstance(categories, list)
            else categories
        )
        lines.append(f"  [{DIM_COLOR}]Categories:[/] {cat_str}")

    content = "\n".join(lines)
    panel = Panel(
        content,
        title=f"[{BRAND_COLOR}]Memory[/]",
        title_align="left",
        border_style=BRAND_COLOR,
        padding=(1, 1),
    )
    console.print()
    console.print(panel)
    console.print()


def format_add_result(console: Console, result: JsonObject | list[JsonObject], output: str) -> None:
    """显示添加结果，合并同一事件的待处理项。"""
    if output == "json":
        format_json(console, result)
        return
    if output == "quiet":
        return

    # API 通常返回包含 results 数组的对象
    results = json_records(result if isinstance(result, list) else result.get("results", [result]))
    if not results:
        console.print(f"  [{DIM_COLOR}]No memories extracted.[/]")
        return

    console.print()
    seen_pending_events: set[str] = set()
    for r in results:
        # 识别平台异步 PENDING 响应
        if r.get("status") == "PENDING":
            event_id = json_string(r.get("event_id", ""), "event_id")
            # 按 event_id 合并待处理项
            if event_id and event_id in seen_pending_events:
                continue
            if event_id:
                seen_pending_events.add(event_id)
            icon = f"[{ACCENT_COLOR}]{_sym('⧗', '...')}[/]"
            parts = [f"  {icon} [{DIM_COLOR}]{'Queued':<10}[/]"]
            parts.append("[white]Processing in background[/]")
            console.print("  ".join(parts))
            if event_id:
                console.print(f"  [{DIM_COLOR}]  event_id: {event_id}[/]")
                console.print(f"  [{DIM_COLOR}]  → Check status: memgo event status {event_id}[/]")
            continue

        event = json_string(r.get("event", "ADD"), "event")
        memory = r.get("memory") or r.get("text") or r.get("content") or r.get("data") or ""
        mem_id = json_string(r.get("id") or r.get("memory_id") or "", "memory.id")[:8]

        if event == "ADD":
            icon = f"[{SUCCESS_COLOR}]+[/]"
            label = "Added"
        elif event == "UPDATE":
            icon = f"[{ACCENT_COLOR}]~[/]"
            label = "Updated"
        elif event == "DELETE":
            icon = "[red]-[/]"
            label = "Deleted"
        elif event == "NOOP":
            icon = f"[{DIM_COLOR}]·[/]"
            label = "No change"
        else:
            icon = f"[{DIM_COLOR}]?[/]"
            label = event

        # 构造结果显示行
        parts = [f"  {icon} [{DIM_COLOR}]{label:<10}[/]"]
        if memory:
            parts.append(f"[white]{memory}[/]")
        if mem_id:
            parts.append(f"[{DIM_COLOR}]({mem_id})[/]")
        console.print("  ".join(parts))
    console.print()


def format_json_envelope(
    console: Console,
    *,
    command: str,
    data: object,
    duration_ms: int | None,
    scope: Mapping[str, JsonValue] | None,
    count: int | None,
    status: str,
    error: str | None,
) -> None:
    """输出包含状态、数据和平台通知的 JSON 信封。"""
    envelope: JsonObject = {
        "status": status,
        "command": command,
    }
    if duration_ms is not None:
        envelope["duration_ms"] = duration_ms
    if scope is not None:
        envelope["scope"] = scope
    if count is not None:
        envelope["count"] = count
    if error:
        envelope["error"] = error
    envelope["data"] = json_value(data)

    # 平台返回未认领账号提示时，将其加入 JSON 信封
    # 代理读取输出即可获得提示
    # 无需检查 HTTP 响应头
    from memgo_cli.runtime.state import take_notice

    notice = take_notice()
    if notice:
        envelope["memgo_notice"] = notice

    console.print_json(json.dumps(envelope, default=str))


def sanitize_agent_data(command: str, data: object) -> object:
    """按命令提取代理需要的响应字段。"""

    def pick(obj: JsonObject, keys: list[str]) -> JsonObject:
        """提取指定字段，忽略响应中的其他数据。"""
        return {k: obj[k] for k in keys if k in obj}

    if data is None:
        return data

    if command == "add":
        items = json_records(data if isinstance(data, list) else [data])
        result = []
        for item in items:
            if item.get("status") == "PENDING":
                result.append(pick(item, ["status", "event_id"]))
            else:
                result.append(pick(item, ["id", "memory", "event"]))
        return result

    if command == "search":
        return [
            pick(r, ["id", "memory", "score", "created_at", "categories", "expiration_date"])
            for r in json_records(data)
        ]

    if command == "list":
        return [
            pick(r, ["id", "memory", "created_at", "categories", "expiration_date"])
            for r in json_records(data)
        ]

    if command == "get":
        return pick(
            json_object(data),
            [
                "id",
                "memory",
                "created_at",
                "updated_at",
                "categories",
                "metadata",
                "expiration_date",
            ],
        )

    if command == "update":
        return pick(json_object(data), ["id", "memory", "expiration_date"])

    if command in ("delete", "delete-all", "entity delete"):
        return data

    if command == "entity list":
        result = []
        for r in json_records(data):
            item = pick(r, ["type", "count"])
            item["name"] = r.get("name") or r.get("id", "")
            result.append(item)
        return result

    if command == "event list":
        return [
            pick(r, ["id", "event_type", "status", "latency", "created_at"])
            for r in json_records(data)
        ]

    if command == "event status":
        ev = json_object(data)
        raw_results = json_records(ev.get("results") or [])
        sanitized_results = []
        for r in raw_results:
            nested = r.get("data") or {}
            memory = nested.get("memory") if isinstance(nested, dict) else None
            sanitized_results.append(
                {
                    "id": r.get("id"),
                    "event": r.get("event"),
                    "user_id": r.get("user_id"),
                    "memory": memory,
                }
            )
        event_result = pick(
            ev, ["id", "event_type", "status", "latency", "created_at", "updated_at"]
        )
        event_result["results"] = sanitized_results
        return event_result

    # 直接保留状态、导入和配置命令数据
    return data


def format_agent_envelope(
    console: Console,
    *,
    command: str,
    data: object,
    duration_ms: int | None,
    scope: Mapping[str, JsonValue] | None,
    count: int | None,
) -> None:
    """输出代理模式使用的结构化 JSON 信封。"""
    envelope: JsonObject = {
        "status": "success",
        "command": command,
    }
    if duration_ms is not None:
        envelope["duration_ms"] = duration_ms
    if scope:
        filtered = {k: v for k, v in scope.items() if v}
        if filtered:
            envelope["scope"] = filtered
    if count is not None:
        envelope["count"] = count
    envelope["data"] = json_value(sanitize_agent_data(command, data))

    # 将未认领账号通知放入信封
    # 代理读取 JSON 即可获得提示
    from memgo_cli.runtime.state import take_notice

    notice = take_notice()
    if notice:
        envelope["memgo_notice"] = notice

    console.print_json(json.dumps(envelope, default=str))


def print_result_summary(
    console: Console,
    count: int,
    *,
    duration_secs: float | None,
    page: int | None,
    **scope_ids: str | None,
) -> None:
    """在列表末尾显示结果数量和截断提示。"""
    parts = [f"{count} result{'s' if count != 1 else ''}"]
    if page is not None:
        parts.append(f"page {page}")
    scope_parts = [f"{k}={v}" for k, v in scope_ids.items() if v]
    if scope_parts:
        parts.append(", ".join(scope_parts))
    if duration_secs is not None:
        parts.append(f"{duration_secs:.2f}s")

    summary = " · ".join(parts)
    console.print(f"  [{DIM_COLOR}]{summary}[/]")
    console.print()


def _format_date(dt_str: object) -> str | None:
    """提取日期显示文本。"""
    if not dt_str:
        return None
    try:
        dt = datetime.fromisoformat(json_string(dt_str, "date").replace("Z", "+00:00"))
        return dt.strftime("%Y-%m-%d")
    except (ValueError, AttributeError):
        return str(dt_str)[:10] if dt_str else None
