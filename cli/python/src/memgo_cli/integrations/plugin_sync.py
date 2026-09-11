"""同步已有 Claude 设置及 shell 密钥条目。

只更新已存在的条目，不创建集成配置；写入采用临时文件加原子替换。
"""

from __future__ import annotations

import contextlib
import json
import os
import re
import tempfile
from pathlib import Path

from memgo_cli.runtime.warning import warn_optional

# 仅处理已知格式的配置文件
_CLAUDE_SETTINGS = Path.home() / ".claude" / "settings.json"
_SHELL_RCS = [Path.home() / ".zshrc", Path.home() / ".bashrc", Path.home() / ".bash_profile"]


def sync_api_key(api_key: str) -> list[str]:
    """同步已存在的插件和 shell 密钥条目，返回实际更新的路径。"""
    if not api_key:
        return []
    updated: list[str] = []
    if _update_claude_settings(_CLAUDE_SETTINGS, api_key):
        updated.append(str(_CLAUDE_SETTINGS))
    for rc in _SHELL_RCS:
        if _update_shell_rc(rc, api_key):
            updated.append(str(rc))
    return updated


def _update_claude_settings(path: Path, api_key: str) -> bool:
    """更新已有 env.MEMGO_API_KEY 条目，返回是否发生写入。"""
    if not path.is_file():
        return False
    try:
        with path.open("r", encoding="utf-8") as f:
            data = json.load(f)
    except (json.JSONDecodeError, OSError) as error:
        warn_optional("plugin_sync_read", error)
        return False
    if not isinstance(data, dict):
        warn_optional("plugin_sync_parse", ValueError("Expected settings object"))
        return False
    env = data.get("env")
    if not isinstance(env, dict) or "MEMGO_API_KEY" not in env:
        # 未找到已有条目，不创建
        return False
    if env["MEMGO_API_KEY"] == api_key:
        return False  # 已经同步，无需写入
    env["MEMGO_API_KEY"] = api_key
    _atomic_write_text(path, json.dumps(data, indent=2, ensure_ascii=False) + "\n")
    return True


# 匹配双引号、单引号或无引号的密钥导出行
# 行尾只匹配空格和制表符
# 保留文件末尾换行
_RC_LINE = re.compile(
    r'^([ \t]*export[ \t]+MEMGO_API_KEY[ \t]*=[ \t]*)(["\']?)([^"\'\n]*)(["\']?)[ \t]*$',
    re.MULTILINE,
)


def _update_shell_rc(path: Path, api_key: str) -> bool:
    """替换已有的密钥导出行，保留其他 shell 内容。"""
    if not path.is_file():
        return False
    try:
        text = path.read_text(encoding="utf-8")
    except OSError as error:
        warn_optional("plugin_sync_read", error)
        return False
    match = _RC_LINE.search(text)
    if not match:
        return False  # 不存在对应导出行
    if match.group(3) == api_key:
        return False
    new_text = _RC_LINE.sub(lambda m: f'{m.group(1)}"{api_key}"', text, count=1)
    _atomic_write_text(path, new_text)
    return True


def _atomic_write_text(path: Path, content: str) -> None:
    """先写临时文件再原子替换，保留原文件权限。"""
    dirname = path.parent
    fd, tmp_path = tempfile.mkstemp(prefix=f".{path.name}.", suffix=".tmp", dir=dirname)
    try:
        with os.fdopen(fd, "w", encoding="utf-8") as f:
            f.write(content)
        # 保留原文件权限
        if path.exists():
            os.chmod(tmp_path, path.stat().st_mode & 0o777)
        os.replace(tmp_path, path)
    except Exception:
        with contextlib.suppress(OSError):
            os.unlink(tmp_path)
        raise
