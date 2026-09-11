"""验证 cli_integration 的行为与兼容性。"""

from __future__ import annotations

import json
import os
import re
import subprocess
import sys

import pytest

_ANSI_RE = re.compile(r"\x1b\[[0-9;]*[mKJHABCDfsu]")


def _strip_ansi(text: str) -> str:
    """去掉 ANSI 转义后检查文本。"""
    return _ANSI_RE.sub("", text)


def _run(
    args: list[str],
    env_override: dict | None = None,
    home_dir: str | None = None,
) -> subprocess.CompletedProcess:
    """在隔离环境中启动真实 CLI 子进程并捕获输出。"""
    env = os.environ.copy()
    # 移除所有 MEMGO 环境变量，保证隔离
    for key in list(env.keys()):
        if key.startswith("MEMGO_"):
            del env[key]
    env["MEMGO_TELEMETRY"] = "false"
    env.pop("FORCE_COLOR", None)
    env["PYTHONIOENCODING"] = "utf-8"
    if home_dir:
        env["HOME"] = home_dir
    if env_override:
        env.update(env_override)
    result = subprocess.run(
        [sys.executable, "-m", "memgo_cli", *args],
        capture_output=True,
        encoding="utf-8",
        env=env,
    )
    return subprocess.CompletedProcess(
        args=result.args,
        returncode=result.returncode,
        stdout=_strip_ansi(result.stdout),
        stderr=_strip_ansi(result.stderr),
    )


@pytest.fixture
def clean_home(tmp_path):
    """提供没有用户配置的临时主目录。"""
    return str(tmp_path)


class TestCLIIntegration:
    """验证无需配置的帮助与版本命令。"""

    def test_help(self):
        result = _run(["--help"])
        assert result.returncode == 0
        assert "memgo" in result.stdout
        assert "add" in result.stdout
        assert "search" in result.stdout

    def test_version_flag_only(self):
        from memgo_cli import __version__

        flag = _run(["--version"])
        assert flag.returncode == 0
        assert __version__ in flag.stdout

    def test_version_subcommand_matches_flag_byte_for_byte(self):
        flag = _run(["--version"])
        cmd = _run(["version"])
        assert cmd.returncode == 0
        assert cmd.stdout == flag.stdout

    @pytest.mark.parametrize(
        "args",
        [["help", "--json"], ["--json", "help"], ["help", "--agent"], ["--agent", "help"]],
    )
    def test_help_json_produces_valid_json(self, args):
        result = _run(args)
        assert result.returncode == 0
        spec = json.loads(result.stdout)
        assert spec["name"] == "memgo"
        assert "add" in spec["commands"]

    def test_help_without_json_is_text(self):
        result = _run(["help"])
        assert result.returncode == 0
        with pytest.raises(json.JSONDecodeError):
            json.loads(result.stdout)

    def test_add_help(self):
        result = _run(["add", "--help"])
        assert result.returncode == 0
        assert "user-id" in result.stdout
        assert "messages" in result.stdout

    def test_add_help_has_scope_panel(self):
        """验证帮助按范围选项分组。"""
        result = _run(["add", "--help"])
        assert result.returncode == 0
        assert "Scope" in result.stdout

    def test_search_help(self):
        result = _run(["search", "--help"])
        assert result.returncode == 0
        assert "top-k" in result.stdout

    def test_search_help_documents_filter_json_shape(self):
        result = _run(["search", "--help"])
        assert result.returncode == 0
        assert "AND" in result.stdout
        assert "categories" in result.stdout

    def test_list_help(self):
        result = _run(["list", "--help"])
        assert result.returncode == 0
        assert "page-size" in result.stdout

    def test_delete_help(self):
        result = _run(["delete", "--help"])
        assert result.returncode == 0
        assert "--all" in result.stdout
        assert "--entity" in result.stdout
        assert "--project" in result.stdout
        assert "--force" in result.stdout
        assert "--dry-run" in result.stdout

    def test_entity_list_help(self):
        result = _run(["entity", "list", "--help"])
        assert result.returncode == 0
        assert "entity-type" in result.stdout.lower() or "entity_type" in result.stdout.lower()

    def test_entity_delete_help(self):
        result = _run(["entity", "delete", "--help"])
        assert result.returncode == 0
        assert "--user-id" in result.stdout
        assert "--force" in result.stdout

    def test_import_help(self):
        result = _run(["import", "--help"])
        assert result.returncode == 0

    def test_no_args_shows_help(self):
        """无命令参数时显示帮助并按 Typer 约定返回 2。"""
        result = _run([])
        # 未给命令时 Typer 按约定返回 2
        # 这是帮助展示行为
        assert result.returncode in (0, 2)
        assert "Usage" in result.stdout


class TestCLIIsolated:
    """验证隔离配置下的命令行为。"""

    def test_add_no_key_errors(self, clean_home):
        """缺少密钥时添加命令明确失败。"""
        result = _run(
            ["add", "test", "--user-id", "alice"],
            home_dir=clean_home,
        )
        assert result.returncode != 0
        combined = result.stderr + result.stdout
        assert "API key" in combined or "api" in combined.lower() or "Error" in combined

    def test_search_no_key_errors(self, clean_home):
        """缺少密钥时搜索命令明确失败。"""
        result = _run(
            ["search", "preferences", "--user-id", "alice"],
            home_dir=clean_home,
        )
        assert result.returncode != 0
        combined = result.stderr + result.stdout
        assert "API key" in combined or "Error" in combined

    def test_list_no_key_errors(self, clean_home):
        """缺少密钥时列表命令明确失败。"""
        result = _run(["list"], home_dir=clean_home)
        assert result.returncode != 0
        combined = result.stderr + result.stdout
        assert "API key" in combined or "Error" in combined

    def test_delete_no_id_no_all_errors(self, clean_home):
        """缺少删除目标时拒绝执行。"""
        result = _run(
            ["delete", "--api-key", "m0-fake-key"],
            home_dir=clean_home,
        )
        assert result.returncode != 0
        combined = result.stderr + result.stdout
        assert (
            "memory ID" in combined.lower()
            or "--all" in combined
            or "--entity" in combined
            or "Error" in combined
        )

    def test_config_show_clean(self, clean_home):
        """没有配置文件时仍能显示默认配置。"""
        result = _run(["config", "show"], home_dir=clean_home)
        assert result.returncode == 0
        assert "backend" in result.stdout.lower() or "platform" in result.stdout.lower()

    def test_config_set_and_get_roundtrip(self, clean_home):
        """写入配置后能够读取同一值。"""
        _run(
            ["config", "set", "defaults.user_id", "integration-test-user"],
            home_dir=clean_home,
        )
        result = _run(
            ["config", "get", "defaults.user_id"],
            home_dir=clean_home,
        )
        assert result.returncode == 0
        assert "integration-test-user" in result.stdout

    def test_import_nonexistent_file(self, clean_home):
        """导入不存在的文件时明确失败。"""
        result = _run(
            ["import", "/nonexistent/file.json", "--api-key", "m0-fake"],
            home_dir=clean_home,
        )
        assert result.returncode != 0
        combined = result.stderr + result.stdout
        assert "Failed" in combined or "Error" in combined or "error" in combined

    def test_add_no_content_errors(self, clean_home):
        """缺少内容时拒绝添加。"""
        result = _run(
            ["add", "--user-id", "alice", "--api-key", "m0-fake"],
            home_dir=clean_home,
        )
        assert result.returncode != 0
        combined = result.stderr + result.stdout
        assert "No content" in combined or "Error" in combined


class TestCLINewFeatures:
    """验证范围选项和实体删除命令。"""

    def test_search_help_has_limit(self):
        result = _run(["search", "--help"])
        assert result.returncode == 0
        assert "--limit" in result.stdout

    def test_delete_entity_via_delete_flag(self):
        """帮助中保留实体删除选项。"""
        result = _run(["delete", "--help"])
        assert result.returncode == 0
        assert "--entity" in result.stdout

    def test_entity_delete_has_scope_options(self):
        """实体删除公开范围选项。"""
        result = _run(["entity", "delete", "--help"])
        assert result.returncode == 0
        assert "--user-id" in result.stdout
        assert "--force" in result.stdout
        assert "--app-id" in result.stdout
        assert "--run-id" in result.stdout
