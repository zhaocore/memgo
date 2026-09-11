"""按已实现的协议装配后端连接器。"""

from __future__ import annotations

from memgo_cli.backend.platform import PlatformBackend
from memgo_cli.backend.types import Backend, BackendContext
from memgo_cli.config.models import MemGoConfig


def get_backend(config: MemGoConfig, context: BackendContext) -> Backend:
    """创建 Platform 连接器，自定义地址也遵循 Platform 协议。"""
    return PlatformBackend(config.platform, context)
