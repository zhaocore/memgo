"""配置模型及默认值。"""

from __future__ import annotations

from dataclasses import dataclass, field

DEFAULT_BASE_URL = "https://api.memgo.ai"
CONFIG_VERSION = 1


@dataclass
class PlatformConfig:
    """平台连接和账号身份配置。"""

    api_key: str = ""
    base_url: str = DEFAULT_BASE_URL
    user_email: str = ""
    # 代理模式：账号尚未被用户认领
    agent_mode: bool = False  # 密钥未认领时为真
    created_via: str = ""  # 来源取值：agent_mode、email、api_key、existing_key
    agent_caller: str = ""  # 代理模式创建时记录调用方名称，例如 claude-code
    claimed_at: str = ""  # 用户认领后的 ISO 时间戳
    default_user_id: str = ""  # 初始化返回的 user_<slug>，用于默认范围


@dataclass
class DefaultsConfig:
    """命令未提供显式范围时使用的实体 ID。"""

    user_id: str = ""
    agent_id: str = ""
    app_id: str = ""
    run_id: str = ""


@dataclass
class TelemetryConfig:
    """持久化遥测匿名标识。"""

    anonymous_id: str = ""


@dataclass
class AgentRushConfig:
    # 用户确认公开记忆提示的 ISO 时间戳
    # 首次交互添加前为空
    """公开记忆提示的确认状态。"""

    acknowledged_at: str = ""


@dataclass
class MemGoConfig:
    """本地配置文件的完整类型模型。"""

    version: int = CONFIG_VERSION
    defaults: DefaultsConfig = field(default_factory=DefaultsConfig)
    platform: PlatformConfig = field(default_factory=PlatformConfig)
    telemetry: TelemetryConfig = field(default_factory=TelemetryConfig)
    agent_rush: AgentRushConfig = field(default_factory=AgentRushConfig)
