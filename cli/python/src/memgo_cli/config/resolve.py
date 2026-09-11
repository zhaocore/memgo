"""配置键解析与不可变更新。"""

from __future__ import annotations

from dataclasses import replace

from memgo_cli.config.models import (
    MemGoConfig,
)

SHORT_KEY_ALIASES: dict[str, str] = {
    "api_key": "platform.api_key",
    "base_url": "platform.base_url",
    "user_email": "platform.user_email",
    "user_id": "defaults.user_id",
    "agent_id": "defaults.agent_id",
    "app_id": "defaults.app_id",
    "run_id": "defaults.run_id",
}


def redact_key(key: str) -> str:
    """显示脱敏后的 API 密钥。"""
    if not key:
        return "(not set)"
    if len(key) <= 8:
        return key[:2] + "***"
    return key[:4] + "..." + key[-4:]


def get_nested_value(config: MemGoConfig, dotted_key: str) -> object:
    """读取点分键名对应字段，未知键返回空值。"""
    path = SHORT_KEY_ALIASES.get(dotted_key, dotted_key).split(".")
    value: object = config
    for part in path:
        if part.startswith("_") or not hasattr(value, "__dataclass_fields__"):
            return None
        if part not in value.__dataclass_fields__:
            return None
        value = getattr(value, part)
    return value


def set_nested_value(config: MemGoConfig, dotted_key: str, value: str) -> MemGoConfig:
    """返回更新指定字段后的新配置；未知键和无效整数明确失败。"""
    path = SHORT_KEY_ALIASES.get(dotted_key, dotted_key).split(".")
    current = get_nested_value(config, dotted_key)
    if current is None or type(current) not in (str, bool, int):
        raise ValueError(f"Unknown config key: {dotted_key}")
    converted: str | bool | int = value
    if isinstance(current, bool):
        converted = value.lower() in ("true", "1", "yes")
    elif isinstance(current, int):
        converted = int(value)
    if len(path) == 1:
        if path[0] != "version" or type(converted) is not int:
            raise ValueError(f"Invalid config key: {dotted_key}")
        return replace(config, version=converted)
    if len(path) != 2:
        raise ValueError(f"Unknown config key: {dotted_key}")
    section = getattr(config, path[0])
    return replace(config, **{path[0]: replace(section, **{path[1]: converted})})
