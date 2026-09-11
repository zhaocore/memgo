"""AGENTRUSH 请求协议。"""

from __future__ import annotations

import httpx

from memgo_cli.backend.json import JsonObject
from memgo_cli.config.models import PlatformConfig


def request_agent_rush(config: PlatformConfig, path: str, body: JsonObject) -> httpx.Response:
    """发送带代理模式标记的请求，由用例处理配额错误。"""
    with httpx.Client(timeout=30.0) as client:
        return client.post(
            f"{config.base_url.rstrip('/')}{path}",
            headers={
                "X-MemGo-Source": "cli",
                "X-MemGo-Client-Language": "python",
                "X-MemGo-Mode": "agent-rush",
                "Authorization": f"Token {config.api_key}",
                "Content-Type": "application/json",
            },
            json=body,
        )
