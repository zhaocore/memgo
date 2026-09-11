"""独立遥测子进程：凭据仅从标准输入读取，失败明确报告。"""

from __future__ import annotations

import json
import sys
import urllib.request
from dataclasses import dataclass
from pathlib import Path

from memgo_cli.backend.json import JsonObject, json_object, json_string
from memgo_cli.runtime.warning import warn_optional


@dataclass(frozen=True)
class SenderContext:
    """校验后的发送配置，不通过进程参数传递凭据。"""

    payload: JsonObject
    posthog_host: str
    needs_email: bool
    memgo_api_key: str
    memgo_base_url: str
    config_path: str
    anon_distinct_id_to_alias: str | None


def _load_context() -> SenderContext:
    """从标准输入读取并校验必要字段，不接受命令行凭据。"""
    raw = json_object(json.loads(sys.stdin.read()))
    payload = json_object(raw.get("payload"))
    for field in ("api_key", "event", "distinct_id"):
        json_string(payload.get(field), f"payload.{field}")
    json_object(payload.get("properties", {}))
    needs_email = raw.get("needs_email", False)
    if not isinstance(needs_email, bool):
        raise ValueError("Invalid needs_email: expected boolean")
    alias = raw.get("anon_distinct_id_to_alias")
    return SenderContext(
        payload=payload,
        posthog_host=json_string(raw.get("posthog_host"), "posthog_host"),
        needs_email=needs_email,
        memgo_api_key=json_string(raw.get("memgo_api_key"), "memgo_api_key"),
        memgo_base_url=json_string(raw.get("memgo_base_url"), "memgo_base_url"),
        config_path=json_string(raw.get("config_path"), "config_path"),
        anon_distinct_id_to_alias=None
        if alias is None
        else json_string(alias, "anon_distinct_id_to_alias"),
    )


def main() -> None:
    """先解析账号身份，再关联匿名身份并发送事件。"""
    context = _load_context()
    payload = context.payload
    if context.needs_email and context.memgo_api_key:
        try:
            payload = _resolve_and_cache_email(context, payload)
        except (OSError, ValueError) as error:
            warn_optional("telemetry_identity", error)
    if context.anon_distinct_id_to_alias:
        _send_identify_event(context, payload, context.anon_distinct_id_to_alias)
    _send_posthog_event(context.posthog_host, payload)


def _send_identify_event(context: SenderContext, payload: JsonObject, anon_id: str) -> None:
    """将匿名历史关联到最终解析出的身份。"""
    properties = json_object(payload.get("properties", {}))
    _send_posthog_event(
        context.posthog_host,
        {
            "api_key": payload["api_key"],
            "event": "$identify",
            "distinct_id": payload["distinct_id"],
            "properties": {
                "$anon_distinct_id": anon_id,
                "$lib": properties.get("$lib", "posthog-python"),
            },
        },
    )


def _resolve_and_cache_email(context: SenderContext, payload: JsonObject) -> JsonObject:
    """查询账号邮箱并缓存，返回更新身份的新负载。"""
    request = urllib.request.Request(
        context.memgo_base_url.rstrip("/") + "/v1/ping/",
        headers={
            "Authorization": "Token " + context.memgo_api_key,
            "Content-Type": "application/json",
        },
    )
    with urllib.request.urlopen(request, timeout=10) as response:
        data = json_object(json.loads(response.read()))
    email = json_string(data.get("user_email"), "user_email")
    _cache_email(context.config_path, email)
    return {**payload, "distinct_id": email}


def _cache_email(config_path: str, email: str) -> None:
    """更新已有配置中的账号邮箱，保留其他字段。"""
    if not config_path:
        return
    path = Path(config_path)
    config = json_object(json.loads(path.read_text(encoding="utf-8")))
    platform = json_object(config.get("platform", {}))
    updated = {**config, "platform": {**platform, "user_email": email}}
    path.write_text(json.dumps(updated, indent=2), encoding="utf-8")


def _send_posthog_event(posthog_host: str, payload: JsonObject) -> None:
    """发送单个事件，不重试可能造成重复计数的写请求。"""
    request = urllib.request.Request(
        posthog_host,
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json"},
    )
    with urllib.request.urlopen(request, timeout=10) as response:
        response.read()


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError) as error:
        warn_optional("telemetry_delivery", error)
        raise SystemExit(1) from error
