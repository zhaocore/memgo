"""账号初始化、邮箱验证与代理身份协议。"""

from __future__ import annotations

import httpx

from memgo_cli.backend.json import JsonObject

SOURCE_HEADERS = {"X-MemGo-Source": "cli", "X-MemGo-Client-Language": "python"}


def bootstrap_agent(base_url: str, body: JsonObject) -> httpx.Response:
    """请求创建未认领的代理账号。"""
    with httpx.Client(timeout=30.0) as client:
        return client.post(
            f"{base_url.rstrip('/')}/api/v1/auth/agent_mode/",
            headers={**SOURCE_HEADERS, "Content-Type": "application/json"},
            json=body,
        )


def send_email_code(base_url: str, email: str) -> httpx.Response:
    """请求向指定邮箱发送验证码。"""
    with httpx.Client(timeout=30.0) as client:
        return client.post(
            f"{base_url.rstrip('/')}/api/v1/auth/email_code/",
            headers=SOURCE_HEADERS,
            json={"email": email},
        )


def verify_email_code(
    base_url: str, email: str, code: str, agent_key: str | None
) -> httpx.Response:
    """验证邮箱，并在提供密钥时认领代理账号。"""
    body = {"email": email, "code": code.strip()}
    if agent_key is not None:
        body["agent_mode_api_key"] = agent_key
    with httpx.Client(timeout=30.0) as client:
        return client.post(
            f"{base_url.rstrip('/')}/api/v1/auth/email_code/verify/",
            headers=SOURCE_HEADERS,
            json=body,
        )


def identify_agent(base_url: str, api_key: str, name: str) -> httpx.Response:
    """更新代理账号的调用方名称。"""
    with httpx.Client(timeout=30.0) as client:
        return client.patch(
            f"{base_url.rstrip('/')}/api/v1/auth/agent_mode/caller/",
            headers={
                **SOURCE_HEADERS,
                "Authorization": f"Token {api_key}",
                "Content-Type": "application/json",
            },
            json={"agent_caller": name},
        )


def identify_reused_key(base_url: str, api_key: str, name: str) -> httpx.Response:
    """按既有密钥复用协议同步调用方名称。"""
    return httpx.patch(
        f"{base_url.rstrip('/')}/api/v1/auth/agent_mode/caller/",
        headers={"Authorization": f"Token {api_key}", "Content-Type": "application/json"},
        json={"agent_caller": name},
        timeout=10.0,
    )


def ping_key(base_url: str, api_key: str, timeout: float) -> httpx.Response:
    """在指定超时内探测密钥，状态由调用方判断。"""
    return httpx.get(
        f"{base_url.rstrip('/')}/v1/ping/",
        headers={"Authorization": f"Token {api_key}"},
        timeout=timeout,
    )
