"""已有密钥复用探测。"""

from __future__ import annotations

import httpx
from rich.console import Console

from memgo_cli.backend.auth import ping_key
from memgo_cli.runtime.warning import warn_optional

console = Console()
err_console = Console(stderr=True)


def _ping_key(api_key: str, base_url: str, timeout: float) -> bool:
    """探测现有密钥是否明确失效。

    仅 401/403 返回假；网络故障记录告警并保留既有密钥，避免误建账号。
    """
    try:
        resp = ping_key(base_url, api_key, timeout)
    except httpx.HTTPError as error:
        warn_optional("reuse_key_probe", error)
        return True  # 结果不确定时保留既有密钥
    return resp.status_code not in (401, 403)
