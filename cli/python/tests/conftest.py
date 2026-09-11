"""验证 conftest 的行为与兼容性。"""

from __future__ import annotations

import os
from unittest.mock import MagicMock

import pytest

from memgo_cli.backend.types import Backend
from memgo_cli.config.models import MemGoConfig


@pytest.fixture(autouse=True)
def isolate_config(tmp_path, monkeypatch):
    """隔离配置目录和环境变量，避免访问用户真实配置。"""
    fake_config_dir = tmp_path / ".memgo"
    fake_config_file = fake_config_dir / "config.json"
    monkeypatch.setattr("memgo_cli.config.store.CONFIG_DIR", fake_config_dir)
    monkeypatch.setattr("memgo_cli.config.store.CONFIG_FILE", fake_config_file)
    # 同时替换直接导入配置的命令引用
    monkeypatch.setattr("memgo_cli.cli.commands.config.CONFIG_DIR", fake_config_dir, raising=False)
    # 清理 MEMGO 环境变量
    for key in list(os.environ.keys()):
        if key.startswith("MEMGO_"):
            monkeypatch.delenv(key, raising=False)
    monkeypatch.setenv("MEMGO_TELEMETRY", "false")
    monkeypatch.setattr("memgo_cli.application.onboarding.init.CONFIG_FILE", fake_config_file)
    monkeypatch.setattr(
        "memgo_cli.integrations.plugin_sync._CLAUDE_SETTINGS", tmp_path / ".claude/settings.json"
    )
    monkeypatch.setattr(
        "memgo_cli.integrations.plugin_sync._SHELL_RCS", [tmp_path / ".zshrc", tmp_path / ".bashrc"]
    )
    return fake_config_dir


@pytest.fixture
def mock_backend():
    """提供具有固定返回值的测试后端。"""
    backend = MagicMock(spec=Backend)

    # 后端默认返回值
    backend.add.return_value = {
        "results": [
            {
                "id": "abc-123-def-456",
                "memory": "User prefers dark mode",
                "event": "ADD",
            }
        ]
    }

    backend.search.return_value = [
        {
            "id": "abc-123-def-456",
            "memory": "User prefers dark mode",
            "score": 0.92,
            "created_at": "2026-02-15T10:30:00Z",
            "categories": ["preferences"],
        },
        {
            "id": "ghi-789-jkl-012",
            "memory": "User uses vim keybindings",
            "score": 0.78,
            "created_at": "2026-03-01T14:00:00Z",
            "categories": ["tools"],
        },
    ]

    backend.get.return_value = {
        "id": "abc-123-def-456",
        "memory": "User prefers dark mode",
        "created_at": "2026-02-15T10:30:00Z",
        "updated_at": "2026-02-20T08:00:00Z",
        "metadata": {"source": "onboarding"},
        "categories": ["preferences"],
    }

    backend.list_memories.return_value = [
        {
            "id": "abc-123-def-456",
            "memory": "User prefers dark mode",
            "created_at": "2026-02-15T10:30:00Z",
            "categories": ["preferences"],
        },
        {
            "id": "ghi-789-jkl-012",
            "memory": "User uses vim keybindings",
            "created_at": "2026-03-01T14:00:00Z",
            "categories": ["tools"],
        },
    ]

    backend.update.return_value = {"id": "abc-123-def-456", "memory": "Updated memory"}
    backend.delete.return_value = {"status": "deleted"}
    backend.status.return_value = {
        "connected": True,
        "backend": "platform",
        "base_url": "https://api.memgo.ai",
    }
    backend.delete_entities.return_value = {"message": "Entity deleted"}
    backend.entities.return_value = [
        {"name": "alice", "count": 5},
        {"name": "bob", "count": 3},
    ]
    backend.list_events.return_value = [
        {
            "id": "evt-abc-123-def-456",
            "event_type": "ADD",
            "status": "SUCCEEDED",
            "graph_status": None,
            "latency": 1234.5,
            "created_at": "2026-04-01T10:00:00Z",
            "updated_at": "2026-04-01T10:00:01Z",
        },
        {
            "id": "evt-def-456-ghi-789",
            "event_type": "SEARCH",
            "status": "PENDING",
            "graph_status": None,
            "latency": None,
            "created_at": "2026-04-01T10:01:00Z",
            "updated_at": "2026-04-01T10:01:00Z",
        },
    ]
    backend.get_event.return_value = {
        "id": "evt-abc-123-def-456",
        "event_type": "ADD",
        "status": "SUCCEEDED",
        "graph_status": "SUCCEEDED",
        "latency": 1234.5,
        "created_at": "2026-04-01T10:00:00Z",
        "updated_at": "2026-04-01T10:00:01Z",
        "results": [
            {
                "id": "mem-abc-123",
                "event": "ADD",
                "user_id": "alice",
                "data": {"memory": "User prefers dark mode"},
            }
        ],
    }

    return backend


@pytest.fixture
def sample_config():
    """提供测试使用的配置对象。"""
    config = MemGoConfig()
    config.platform.api_key = "m0-test-key-12345678"
    config.platform.base_url = "https://api.memgo.ai"
    return config


@pytest.fixture(autouse=True)
def isolate_invocation():
    """隔离直接调用命令函数的测试状态，并关闭测试创建的连接。"""
    from memgo_cli.runtime.state import finish_invocation, start_invocation

    token = start_invocation(False)
    try:
        yield
    finally:
        finish_invocation(token)
