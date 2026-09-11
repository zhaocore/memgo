"""验证 config 的行为与兼容性。"""

from __future__ import annotations

import os

import pytest

from memgo_cli.config.models import MemGoConfig
from memgo_cli.config.resolve import get_nested_value, redact_key, set_nested_value
from memgo_cli.config.store import load_config, save_config


class TestRedactKey:
    def test_empty_key(self):
        assert redact_key("") == "(not set)"

    def test_short_key(self):
        assert redact_key("abc") == "ab***"

    def test_normal_key(self):
        result = redact_key("m0-abcdefgh12345678")
        assert result == "m0-a...5678"
        assert "abcdefgh" not in result

    def test_exact_8_chars(self):
        # 长度不超过八个字符，使用完整遮罩
        assert redact_key("12345678") == "12***"


class TestConfig:
    def test_default_config(self):
        config = MemGoConfig()
        assert config.platform.base_url == "https://api.memgo.ai"
        assert config.platform.api_key == ""

    def test_save_and_load(self, isolate_config):
        config = MemGoConfig()
        config.platform.api_key = "m0-test-key"

        save_config(config)

        loaded = load_config()
        assert loaded.platform.api_key == "m0-test-key"

    def test_env_var_override(self, isolate_config, monkeypatch):
        config = MemGoConfig()
        config.platform.api_key = "file-key"
        save_config(config)

        monkeypatch.setenv("MEMGO_API_KEY", "env-key")
        loaded = load_config()
        assert loaded.platform.api_key == "env-key"

    def test_load_nonexistent_config(self, isolate_config):
        config = load_config()
        assert config.platform.api_key == ""

    def test_config_file_permissions(self, isolate_config):
        config = MemGoConfig()
        config.platform.api_key = "secret"
        save_config(config)

        from memgo_cli.config.store import CONFIG_FILE

        mode = os.stat(CONFIG_FILE).st_mode & 0o777
        if os.name != "nt":
            assert mode == 0o600

    def test_defaults_save_and_load(self, isolate_config):
        config = MemGoConfig()
        config.defaults.user_id = "alice"
        config.defaults.agent_id = "support-bot"
        config.defaults.app_id = "my-app"
        config.defaults.run_id = "run-001"

        save_config(config)
        loaded = load_config()

        assert loaded.defaults.user_id == "alice"
        assert loaded.defaults.agent_id == "support-bot"
        assert loaded.defaults.app_id == "my-app"
        assert loaded.defaults.run_id == "run-001"

    def test_defaults_env_var_override(self, isolate_config, monkeypatch):
        config = MemGoConfig()
        config.defaults.user_id = "file-user"
        save_config(config)

        monkeypatch.setenv("MEMGO_USER_ID", "env-user")
        monkeypatch.setenv("MEMGO_AGENT_ID", "env-agent")
        loaded = load_config()
        assert loaded.defaults.user_id == "env-user"
        assert loaded.defaults.agent_id == "env-agent"

    def test_backward_compat_no_defaults_key(self, isolate_config):
        """兼容不含 defaults 的旧配置。"""
        import json

        from memgo_cli.config.store import CONFIG_FILE, ensure_config_dir

        ensure_config_dir()
        # 写入没有 defaults 字段的旧配置
        data = {
            "version": 1,
            "platform": {"api_key": "m0-test", "base_url": "https://api.memgo.ai"},
        }
        with open(CONFIG_FILE, "w") as f:
            json.dump(data, f)

        loaded = load_config()
        assert loaded.platform.api_key == "m0-test"
        assert loaded.defaults.user_id == ""
        assert loaded.defaults.agent_id == ""

    def test_default_config_has_empty_defaults(self):
        config = MemGoConfig()
        assert config.defaults.user_id == ""
        assert config.defaults.agent_id == ""
        assert config.defaults.app_id == ""
        assert config.defaults.run_id == ""


class TestNestedAccess:
    def test_get_nested_value(self):
        config = MemGoConfig()
        config.platform.api_key = "test-key"
        assert get_nested_value(config, "platform.api_key") == "test-key"

    def test_get_nonexistent_key(self):
        config = MemGoConfig()
        assert get_nested_value(config, "nonexistent.key") is None

    def test_set_nested_value(self):
        config = MemGoConfig()
        updated = set_nested_value(config, "platform.api_key", "new-key")
        assert updated.platform.api_key == "new-key"
        assert config.platform.api_key == ""

    def test_set_int_value_rejects_invalid_input(self):
        config = MemGoConfig()
        with pytest.raises(ValueError):
            set_nested_value(config, "version", "abc")
        assert config.version == 1

    def test_set_nonexistent_key(self):
        config = MemGoConfig()
        with pytest.raises(ValueError):
            set_nested_value(config, "nonexistent.key", "val")

    def test_get_defaults_user_id(self):
        config = MemGoConfig()
        config.defaults.user_id = "alice"
        assert get_nested_value(config, "defaults.user_id") == "alice"

    def test_set_defaults_user_id(self):
        config = MemGoConfig()
        updated = set_nested_value(config, "defaults.user_id", "bob")
        assert updated.defaults.user_id == "bob"
        assert config.defaults.user_id == ""


class TestResolveIds:
    def test_cli_flag_overrides_default(self):
        from memgo_cli.application.entity_ids import _resolve_ids

        config = MemGoConfig()
        config.defaults.user_id = "default-user"
        ids = _resolve_ids(config, user_id="cli-user", agent_id=None, app_id=None, run_id=None)
        assert ids["user_id"] == "cli-user"

    def test_default_used_when_flag_is_none(self):
        from memgo_cli.application.entity_ids import _resolve_ids

        config = MemGoConfig()
        config.defaults.user_id = "default-user"
        config.defaults.agent_id = "default-agent"
        ids = _resolve_ids(config, user_id=None, agent_id=None, app_id=None, run_id=None)
        assert ids["user_id"] == "default-user"
        assert ids["agent_id"] == "default-agent"

    def test_none_when_neither_set(self):
        from memgo_cli.application.entity_ids import _resolve_ids

        config = MemGoConfig()
        ids = _resolve_ids(config, user_id=None, agent_id=None, app_id=None, run_id=None)
        assert ids["user_id"] is None
        assert ids["agent_id"] is None
        assert ids["app_id"] is None
        assert ids["run_id"] is None

    def test_empty_string_treated_as_unset(self):
        from memgo_cli.application.entity_ids import _resolve_ids

        config = MemGoConfig()
        config.defaults.user_id = ""
        ids = _resolve_ids(config, user_id=None, agent_id=None, app_id=None, run_id=None)
        assert ids["user_id"] is None
