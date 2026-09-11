"""校验外部配置并按环境变量优先级返回新模型。"""

from __future__ import annotations

from collections.abc import Mapping
from dataclasses import fields, replace

from memgo_cli.backend.json import json_object
from memgo_cli.config.models import (
    MemGoConfig,
)


def resolve_config(data: object, env: Mapping[str, str]) -> MemGoConfig:
    """解析配置对象，忽略额外字段并拒绝已知字段类型错误。"""
    raw = json_object(data)
    base = MemGoConfig()
    version = raw.get("version", base.version)
    if type(version) is not int:
        raise ValueError("Invalid config version: expected integer")
    config = replace(base, version=version)
    for name in ("platform", "defaults", "telemetry", "agent_rush"):
        values = json_object(raw.get(name, {}))
        section = getattr(config, name)
        for field in fields(section):
            if field.name not in values:
                continue
            value = values[field.name]
            expected = type(getattr(section, field.name))
            if type(value) is not expected:
                raise ValueError(
                    f"Invalid config field {name}.{field.name}: expected {expected.__name__}"
                )
            section = replace(section, **{field.name: value})
        config = replace(config, **{name: section})
    return replace(
        config,
        platform=replace(
            config.platform,
            api_key=env.get("MEMGO_API_KEY") or config.platform.api_key,
            base_url=env.get("MEMGO_BASE_URL") or config.platform.base_url,
        ),
        defaults=replace(
            config.defaults,
            user_id=env.get("MEMGO_USER_ID") or config.defaults.user_id,
            agent_id=env.get("MEMGO_AGENT_ID") or config.defaults.agent_id,
            app_id=env.get("MEMGO_APP_ID") or config.defaults.app_id,
            run_id=env.get("MEMGO_RUN_ID") or config.defaults.run_id,
        ),
    )
