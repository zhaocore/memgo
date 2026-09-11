"""可选集成故障的结构化告警。"""

from __future__ import annotations

import json
import sys


def warn_optional(operation: str, error: BaseException) -> None:
    """记录失败操作和异常类型，避免输出密钥及原始响应。"""
    sys.stderr.write(
        json.dumps(
            {
                "level": "warn",
                "event": "optional_operation_failed",
                "operation": operation,
                "error_type": type(error).__name__,
            }
        )
        + "\n"
    )
