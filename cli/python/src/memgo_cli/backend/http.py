"""HTTP 响应校验、错误分类和敏感字段脱敏。"""

from __future__ import annotations

import json
import re

import httpx

from memgo_cli.backend.errors import APIError, AuthError, NotFoundError
from memgo_cli.backend.json import JsonValue, json_value


def redact_response(response: httpx.Response) -> str:
    """移除响应中回显的鉴权凭据及常见敏感字段。"""
    text = response.text
    for name, secret in response.request.headers.items():
        if name.lower() in ("authorization", "x-api-key") and secret:
            text = text.replace(secret, "[redacted]")
            token = re.sub(r"^(Token|Bearer) ", "", secret)
            if token:
                text = text.replace(token, "[redacted]")
    return re.sub(
        r'("(?:api_key|access_token|refresh_token|password|secret|authorization)"\s*:\s*)"[^"]*"',
        r'\1"[redacted]"',
        text,
        flags=re.IGNORECASE,
    )


def read_response(response: httpx.Response) -> tuple[JsonValue, str | None]:
    """校验成功响应并提取通知，失败时返回可分类的脱敏异常。"""
    request = response.request
    if response.is_error:
        detail = f"{request.method} {request.url.path}: HTTP {response.status_code}, response={redact_response(response)}"
        if response.status_code in (401, 403):
            raise AuthError(detail)
        if response.status_code == 404:
            raise NotFoundError(detail)
        raise APIError(detail)
    if response.status_code == 204:
        return {}, response.headers.get("X-MemGo-Notice-Message")
    try:
        data = json_value(response.json())
    except (ValueError, json.JSONDecodeError) as error:
        raise APIError(
            f"{request.method} {request.url.path}: HTTP {response.status_code}, response is not valid JSON"
        ) from error
    notice = None
    if isinstance(data, dict):
        notice = data.pop("memgo_notice", None)
    elif isinstance(data, list) and data and isinstance(data[0], dict):
        notice = data[0].pop("memgo_notice", None)
    if notice is None:
        notice = response.headers.get("X-MemGo-Notice-Message")
    if notice is not None and not isinstance(notice, str):
        raise APIError("Invalid memgo_notice: expected string")
    return data, notice
