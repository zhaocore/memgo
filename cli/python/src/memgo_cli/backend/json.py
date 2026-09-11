"""递归 JSON 类型和外部数据校验。"""

from __future__ import annotations

import math
from collections.abc import Mapping, Sequence
from typing import TypeAlias

JsonValue: TypeAlias = (
    "bool | int | float | str | Sequence[JsonValue] | Mapping[str, JsonValue] | None"
)
JsonObject: TypeAlias = dict[str, JsonValue]


def json_value(value: object) -> JsonValue:
    """校验并复制 JSON 数据，拒绝不支持的值与非有限数字。"""
    if value is None or isinstance(value, (str, bool, int)):
        return value
    if isinstance(value, float) and math.isfinite(value):
        return value
    if isinstance(value, list):
        return [json_value(item) for item in value]
    if isinstance(value, dict) and all(isinstance(key, str) for key in value):
        return {key: json_value(item) for key, item in value.items()}
    raise ValueError("Expected valid JSON data")


def json_object(value: object) -> JsonObject:
    """取得 JSON 对象，其他顶层类型明确失败。"""
    parsed = json_value(value)
    if not isinstance(parsed, dict):
        raise ValueError("Expected a JSON object")
    return parsed


def json_records(value: object) -> list[JsonObject]:
    """从数组或 results/memories 信封中读取对象列表。"""
    parsed = json_value(value)
    if isinstance(parsed, dict):
        parsed = parsed.get("results", parsed.get("memories"))
    if not isinstance(parsed, list):
        raise ValueError("Expected an array or results/memories array envelope")
    return [json_object(item) for item in parsed]


def json_result(value: object) -> JsonObject | list[JsonObject]:
    """校验添加响应，保留单对象或数组的协议形状。"""
    if isinstance(value, list):
        return [json_object(item) for item in value]
    return json_object(value)


def json_string(value: object, field: str) -> str:
    """校验需要字符串的协议字段，不进行隐式类型转换。"""
    if not isinstance(value, str):
        raise ValueError(f"Invalid field {field}: expected string")
    return value


def json_messages(value: object) -> list[JsonObject]:
    """校验非空消息数组，每项必须包含非空角色和字符串内容。"""
    if not isinstance(value, list) or not value:
        raise ValueError("Invalid messages: expected non-empty array")
    records = [json_object(item) for item in value]
    for index, record in enumerate(records):
        role = json_string(record.get("role"), f"messages[{index}].role")
        if not role:
            raise ValueError(f"Invalid messages[{index}].role: expected non-empty string")
        json_string(record.get("content"), f"messages[{index}].content")
    return records
