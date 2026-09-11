"""配置文件读写与密钥同步。"""

from __future__ import annotations

import json
import os
import stat
from dataclasses import asdict
from pathlib import Path

from memgo_cli.config.models import MemGoConfig
from memgo_cli.config.parse import resolve_config
from memgo_cli.runtime.warning import warn_optional

CONFIG_DIR = Path.home() / ".memgo"
CONFIG_FILE = CONFIG_DIR / "config.json"


def ensure_config_dir() -> Path:
    """创建仅当前用户可访问的配置目录。"""
    CONFIG_DIR.mkdir(parents=True, exist_ok=True)
    os.chmod(CONFIG_DIR, stat.S_IRWXU)
    return CONFIG_DIR


def load_config() -> MemGoConfig:
    """读取文件并应用环境变量覆盖，畸形配置明确失败。"""
    data = json.loads(CONFIG_FILE.read_text(encoding="utf-8")) if CONFIG_FILE.exists() else {}
    return resolve_config(data, os.environ)


def save_config(config: MemGoConfig) -> None:
    """以受限权限保存配置，再同步已有插件密钥。"""
    ensure_config_dir()
    descriptor = os.open(CONFIG_FILE, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600)
    with os.fdopen(descriptor, "w", encoding="utf-8") as stream:
        os.chmod(CONFIG_FILE, 0o600)
        json.dump(asdict(config), stream, indent=2)
    if config.platform.api_key:
        from memgo_cli.integrations.plugin_sync import sync_api_key

        try:
            sync_api_key(config.platform.api_key)
        except (OSError, ValueError) as error:
            warn_optional("plugin_sync", error)
