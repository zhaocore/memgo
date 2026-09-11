"""显式实体 ID 与配置默认范围解析。"""

from __future__ import annotations

from memgo_cli.config.models import MemGoConfig


def _resolve_ids(
    config: MemGoConfig,
    *,
    user_id: str | None,
    agent_id: str | None,
    app_id: str | None,
    run_id: str | None,
) -> dict[str, str | None]:
    """解析实体范围。

    提供任意显式 ID 时仅使用显式值；否则读取配置默认值，避免混合范围。
    """
    has_explicit = any([user_id, agent_id, app_id, run_id])
    if has_explicit:
        return {
            "user_id": user_id or None,
            "agent_id": agent_id or None,
            "app_id": app_id or None,
            "run_id": run_id or None,
        }
    return {
        "user_id": config.defaults.user_id or None,
        "agent_id": config.defaults.agent_id or None,
        "app_id": config.defaults.app_id or None,
        "run_id": config.defaults.run_id or None,
    }
