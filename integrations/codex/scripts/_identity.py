"""Resolve MemGo identity: API key, user_id, settings。

API key 解析(第一个非空生效):
  1. MEMGO_API_KEY 环境变量(显式 / shell profile)
  2. 从 shell profile 文件抽取(~/.zshrc, ~/.bashrc 等)
     某些桌面端不继承 shell 环境变量, 此兜底覆盖在 profile 里设置
     MEMGO_API_KEY 的场景。

user_id 解析:
  1. MEMGO_USER_ID 环境变量(显式覆盖)
  2. $USER, 否则 "default"

agent_id(仓库身份)解析见 _project.resolve_agent_id。

Settings 解析:
  ~/.memgo/settings.json(用户可编辑, 缺失时回退默认值)
"""

from __future__ import annotations

import os
import re
from pathlib import Path


def _extract_key_from_shell_profiles() -> str:
    """从 shell profile 文件抽取 MEMGO_API_KEY。

    处理常见的 ``export MEMGO_API_KEY=...`` 写法, 不 source 整个
    profile(避免副作用)。
    """
    profiles = [".zshrc", ".bashrc", ".zprofile", ".bash_profile", ".profile"]
    pattern = re.compile(r'^\s*(?:export\s+)?MEMGO_API_KEY=(.+)$')

    for name in profiles:
        path = Path.home() / name
        if not path.is_file():
            continue
        try:
            for line in path.read_text(encoding="utf-8", errors="replace").splitlines():
                m = pattern.match(line)
                if not m:
                    continue
                value = m.group(1).strip()
                value = re.sub(r'#.*$', '', value).strip()
                value = value.strip("\"'")
                if value and not value.startswith("$"):
                    return value
        except OSError:
            continue
    return ""


def resolve_api_key() -> str:
    key = os.environ.get("MEMGO_API_KEY", "").strip()
    if key:
        return key
    key = _extract_key_from_shell_profiles()
    if key:
        return key
    return ""


def resolve_user_id() -> str:
    explicit = os.environ.get("MEMGO_USER_ID", "").strip()
    if explicit:
        return explicit
    return os.environ.get("USER") or "default"


def resolve_config() -> dict:
    """从 ~/.memgo/settings.json 解析设置, 失败回退默认值。"""
    try:
        from load_settings import load_settings
        return load_settings()
    except ImportError:
        return {
            "auto_save": True,
            "auto_search": True,
            "search_limit": 10,
            "debug": False,
        }
