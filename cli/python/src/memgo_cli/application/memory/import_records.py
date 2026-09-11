"""导入文件的结构化校验，先验证全部记录再发送写请求。"""

from __future__ import annotations

from dataclasses import dataclass

from memgo_cli.backend.json import JsonObject, json_object, json_string


@dataclass(frozen=True)
class ImportRecord:
    """已校验的单条导入记录。"""

    content: str
    user_id: str | None
    agent_id: str | None
    metadata: JsonObject | None


def parse_import_records(value: object) -> list[ImportRecord]:
    """解析单对象或数组；缺少内容和已知字段类型错误时拒绝整个输入。"""
    items = value if isinstance(value, list) else [value]
    result: list[ImportRecord] = []
    for index, item in enumerate(items):
        raw = json_object(item)
        content = json_string(
            raw.get("memory", raw.get("text", raw.get("content"))), f"records[{index}].content"
        )
        if not content:
            raise ValueError(f"Invalid records[{index}].content: expected non-empty string")
        user_id, agent_id, metadata = raw.get("user_id"), raw.get("agent_id"), raw.get("metadata")
        result.append(
            ImportRecord(
                content=content,
                user_id=None
                if user_id is None
                else json_string(user_id, f"records[{index}].user_id"),
                agent_id=None
                if agent_id is None
                else json_string(agent_id, f"records[{index}].agent_id"),
                metadata=None if metadata is None else json_object(metadata),
            )
        )
    return result
