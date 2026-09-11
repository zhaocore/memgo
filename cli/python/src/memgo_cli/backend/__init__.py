"""后端公共端口与装配入口。"""

from memgo_cli.backend.factory import get_backend
from memgo_cli.backend.types import Backend, BackendContext

__all__ = ["Backend", "BackendContext", "get_backend"]
