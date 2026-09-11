"""通过独立子进程发送可选遥测，MEMGO_TELEMETRY=false 可关闭。"""

from __future__ import annotations

import hashlib
import json
import os
import platform
import subprocess
import sys
import uuid

from memgo_cli.backend.json import JsonObject
from memgo_cli.runtime.warning import warn_optional

POSTHOG_API_KEY = "phc_hgJkUVJFYtmaJqrvf6CYN67TIQ8yhXAkWzUn9AMU4yX"
POSTHOG_HOST = "https://us.i.posthog.com/i/v0/e/"


def _is_telemetry_enabled() -> bool:
    """读取遥测环境变量开关。"""
    val = os.environ.get("MEMGO_TELEMETRY", "true").lower()
    return val not in ("false", "0", "no")


def _get_or_create_anonymous_id() -> str:
    """读取持久化匿名标识；首次使用时生成并保存。"""
    from memgo_cli.config.store import load_config, save_config

    config = load_config()
    if config.telemetry.anonymous_id:
        return config.telemetry.anonymous_id

    new_id = f"cli-anon-{uuid.uuid4().hex}"
    config.telemetry.anonymous_id = new_id
    save_config(config)
    return new_id


def _get_distinct_id() -> str:
    """按缓存邮箱、密钥摘要、持久化匿名标识的顺序确定遥测身份。"""
    from memgo_cli.config.store import load_config

    config = load_config()
    if config.platform.user_email:
        return config.platform.user_email
    if config.platform.api_key:
        return hashlib.md5(config.platform.api_key.encode()).hexdigest()
    return _get_or_create_anonymous_id()


def capture_event(
    event_name: str,
    properties: JsonObject | None,
    pre_resolved_email: str | None,
) -> None:
    """通过独立子进程异步发送遥测。

    预先验证的邮箱直接作为身份，密钥仅经子进程标准输入传递。
    """
    if not _is_telemetry_enabled():
        return

    try:
        from memgo_cli import __version__
        from memgo_cli.config.store import CONFIG_FILE, load_config, save_config

        config = load_config()
        distinct_id = pre_resolved_email or _get_distinct_id()

        # 识别匿名用户切换为已知身份
        # 已有匿名 ID 时仅发送一次
        # 通过 $identify 关联注册前后的事件
        # 清除旧标识，避免重复关联
        anon_id_to_alias: str | None = None
        if (
            distinct_id
            and not distinct_id.startswith("cli-anon-")
            and config.telemetry.anonymous_id
        ):
            anon_id_to_alias = config.telemetry.anonymous_id
            config.telemetry.anonymous_id = ""
            save_config(config)

        # 每个 cli.* 事件带上配置中的 agent_mode
        # 该字段标识尚未认领的代理密钥
        # 用于关联初始化、添加和搜索事件
        payload = {
            "api_key": POSTHOG_API_KEY,
            "distinct_id": distinct_id,
            "event": event_name,
            "properties": {
                "source": "CLI",
                "language": "python",
                "cli_version": __version__,
                "agent_mode": bool(config.platform.agent_mode),
                "python_version": sys.version,
                "os": sys.platform,
                "os_version": platform.version(),
                "$process_person_profile": False,
                "$lib": "posthog-python",
                **(properties or {}),
            },
        }

        context = {
            "payload": payload,
            "posthog_host": POSTHOG_HOST,
            "needs_email": not distinct_id or "@" not in distinct_id,
            "memgo_api_key": config.platform.api_key or "",
            "memgo_base_url": config.platform.base_url or "https://api.memgo.ai",
            "config_path": str(CONFIG_FILE),
            "anon_distinct_id_to_alias": anon_id_to_alias,
        }

        child = subprocess.Popen(
            [sys.executable, "-m", "memgo_cli.integrations.telemetry_sender"],
            stdin=subprocess.PIPE,
            stdout=subprocess.DEVNULL,
            stderr=None,
            start_new_session=True,
            close_fds=True,
            text=True,
        )
        if child.stdin is None:
            raise OSError("Telemetry subprocess has no input pipe")
        try:
            child.stdin.write(json.dumps(context))
        finally:
            child.stdin.close()
    except (OSError, ValueError) as error:
        warn_optional("telemetry_spawn", error)
