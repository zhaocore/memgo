"""保留安装入口和命令树导出。"""

from memgo_cli.cli.program import create_app
from memgo_cli.cli.run import main

app = create_app()

__all__ = ["app", "create_app", "main"]
