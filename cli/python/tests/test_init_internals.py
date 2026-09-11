"""验证 init_internals 的行为与兼容性。"""

from __future__ import annotations

from unittest.mock import MagicMock

import httpx
import pytest

from memgo_cli.application.onboarding.key import _ping_key
from memgo_cli.integrations.plugin_sync import _update_claude_settings, _update_shell_rc

# 密钥有效性探测


class _Resp:
    def __init__(self, status_code: int) -> None:
        self.status_code = status_code


def test_ping_key_200_is_valid(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(httpx, "get", lambda *a, **kw: _Resp(200))
    assert _ping_key("k", "http://x", timeout=5.0) is True


def test_ping_key_401_is_invalid(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(httpx, "get", lambda *a, **kw: _Resp(401))
    assert _ping_key("k", "http://x", timeout=5.0) is False


def test_ping_key_403_is_invalid(monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(httpx, "get", lambda *a, **kw: _Resp(403))
    assert _ping_key("k", "http://x", timeout=5.0) is False


def test_ping_key_5xx_is_not_definitively_invalid(monkeypatch: pytest.MonkeyPatch) -> None:
    # 上游瞬时故障不得触发新账号创建
    monkeypatch.setattr(httpx, "get", lambda *a, **kw: _Resp(503))
    assert _ping_key("k", "http://x", timeout=5.0) is True


def test_ping_key_connect_error_prefers_reuse(monkeypatch: pytest.MonkeyPatch) -> None:
    # 网络故障不得重新创建密钥
    def boom(*a, **kw):
        raise httpx.ConnectError("nope")

    monkeypatch.setattr(httpx, "get", boom)
    assert _ping_key("k", "http://x", timeout=5.0) is True


def test_ping_key_timeout_prefers_reuse(monkeypatch: pytest.MonkeyPatch) -> None:
    def boom(*a, **kw):
        raise httpx.ReadTimeout("slow")

    monkeypatch.setattr(httpx, "get", boom)
    assert _ping_key("k", "http://x", timeout=5.0) is True


# shell 配置同步


def test_shell_rc_updates_existing_export_preserves_trailing_newline(tmp_path) -> None:
    rc = tmp_path / ".zshrc"
    rc.write_text('export MEMGO_API_KEY="old"\n', encoding="utf-8")
    changed = _update_shell_rc(rc, "newkey")
    assert changed is True
    assert rc.read_text(encoding="utf-8") == 'export MEMGO_API_KEY="newkey"\n'


def test_shell_rc_does_not_create_new_export(tmp_path) -> None:
    rc = tmp_path / ".zshrc"
    rc.write_text("alias ll='ls -la'\n", encoding="utf-8")
    changed = _update_shell_rc(rc, "newkey")
    assert changed is False
    assert rc.read_text(encoding="utf-8") == "alias ll='ls -la'\n"


def test_shell_rc_preserves_surrounding_content(tmp_path) -> None:
    rc = tmp_path / ".zshrc"
    original = "# my zshrc\nalias ll='ls -la'\nexport MEMGO_API_KEY='old'\nexport OTHER=keepme\n"
    rc.write_text(original, encoding="utf-8")
    _update_shell_rc(rc, "newkey")
    after = rc.read_text(encoding="utf-8")
    assert "alias ll='ls -la'\n" in after
    assert "export OTHER=keepme\n" in after
    assert "# my zshrc\n" in after
    assert 'export MEMGO_API_KEY="newkey"\n' in after


def test_shell_rc_idempotent_when_already_matching(tmp_path) -> None:
    rc = tmp_path / ".zshrc"
    rc.write_text('export MEMGO_API_KEY="same"\n', encoding="utf-8")
    assert _update_shell_rc(rc, "same") is False


def test_shell_rc_missing_file_is_noop(tmp_path) -> None:
    rc = tmp_path / ".zshrc"  # 文件不存在
    assert _update_shell_rc(rc, "x") is False


# Claude 设置同步


def test_claude_settings_does_not_create_env_block(tmp_path) -> None:
    import json

    settings = tmp_path / "settings.json"
    settings.write_text(json.dumps({"otherKey": 1}), encoding="utf-8")
    changed = _update_claude_settings(settings, "newkey")
    assert changed is False
    # 原内容不变
    assert json.loads(settings.read_text(encoding="utf-8")) == {"otherKey": 1}


def test_claude_settings_does_not_create_memgo_entry_in_existing_env(tmp_path) -> None:
    import json

    settings = tmp_path / "settings.json"
    settings.write_text(json.dumps({"env": {"OTHER_KEY": "x"}}), encoding="utf-8")
    changed = _update_claude_settings(settings, "newkey")
    assert changed is False


def test_claude_settings_updates_existing_entry(tmp_path) -> None:
    import json

    settings = tmp_path / "settings.json"
    settings.write_text(
        json.dumps({"env": {"MEMGO_API_KEY": "old", "OTHER": "y"}}, indent=2),
        encoding="utf-8",
    )
    changed = _update_claude_settings(settings, "fresh")
    assert changed is True
    data = json.loads(settings.read_text(encoding="utf-8"))
    assert data["env"]["MEMGO_API_KEY"] == "fresh"
    assert data["env"]["OTHER"] == "y"  # 保留其他字段


def test_claude_settings_idempotent(tmp_path) -> None:
    import json

    settings = tmp_path / "settings.json"
    settings.write_text(json.dumps({"env": {"MEMGO_API_KEY": "same"}}), encoding="utf-8")
    assert _update_claude_settings(settings, "same") is False


def test_claude_settings_malformed_json_is_noop(tmp_path) -> None:
    settings = tmp_path / "settings.json"
    settings.write_text("{ this is not json", encoding="utf-8")
    assert _update_claude_settings(settings, "x") is False


# 初始化限流错误映射


def test_bootstrap_403_permission_surfaces_ratelimit(monkeypatch, capsys) -> None:
    """将初始化接口的权限拒绝映射为每日注册限额提示。"""
    from memgo_cli.application.onboarding.agent_mode import bootstrap_via_backend
    from memgo_cli.config.models import MemGoConfig

    fake_resp = MagicMock()
    fake_resp.status_code = 403
    fake_resp.text = '{"detail": "You do not have permission to perform this action."}'
    fake_resp.json = MagicMock(
        return_value={"detail": "You do not have permission to perform this action."}
    )

    class _Client:
        def __init__(self, *a, **kw):
            pass

        def __enter__(self):
            return self

        def __exit__(self, *a):
            return False

        def post(self, *a, **kw):
            return fake_resp

    monkeypatch.setattr(httpx, "Client", _Client)
    cfg = MemGoConfig()
    cfg.platform.base_url = "https://api.memgo.ai"
    import typer

    with pytest.raises(typer.Exit):
        bootstrap_via_backend(cfg, source=None, agent_caller=None)

    captured = capsys.readouterr()
    combined = captured.out + captured.err
    assert "Daily Agent Mode signup limit reached" in combined
    assert "permission to perform this action" not in combined
