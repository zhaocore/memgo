"""验证 agent_mode 的行为与兼容性。"""

from __future__ import annotations

import os
import re
import subprocess
import sys

import pytest

_ANSI_RE = re.compile(r"\x1b\[[0-9;]*[mKJHABCDfsu]")


def _strip_ansi(text: str) -> str:
    return _ANSI_RE.sub("", text)


def _run(args: list[str], home_dir: str | None = None) -> subprocess.CompletedProcess:
    env = os.environ.copy()
    for key in list(env.keys()):
        if key.startswith("MEMGO_"):
            del env[key]
    env["MEMGO_TELEMETRY"] = "false"
    env.pop("FORCE_COLOR", None)
    env["PYTHONIOENCODING"] = "utf-8"
    if home_dir:
        env["HOME"] = home_dir
    result = subprocess.run(
        [sys.executable, "-m", "memgo_cli", *args],
        capture_output=True,
        encoding="utf-8",
        env=env,
        timeout=15,
    )
    return subprocess.CompletedProcess(
        args=result.args,
        returncode=result.returncode,
        stdout=_strip_ansi(result.stdout),
        stderr=_strip_ansi(result.stderr),
    )


@pytest.fixture
def clean_home(tmp_path):
    return str(tmp_path)


class TestInitFlagSurface:
    """初始化帮助保留代理模式和认领选项。"""

    def test_init_help_lists_agent_flag(self):
        result = _run(["init", "--help"])
        assert result.returncode == 0
        assert "--agent" in result.stdout

    def test_init_help_describes_agent_mode(self):
        result = _run(["init", "--help"])
        assert result.returncode == 0
        # 帮助说明应解释 --agent 的行为
        # 方便代理发现初始化入口
        assert "Agent Mode" in result.stdout or "unattended" in result.stdout.lower()

    def test_init_help_lists_source_flag(self):
        result = _run(["init", "--help"])
        assert result.returncode == 0
        assert "--source" in result.stdout

    def test_init_help_lists_email_and_code(self):
        # 保留邮箱认领选项
        result = _run(["init", "--help"])
        assert result.returncode == 0
        assert "--email" in result.stdout
        assert "--code" in result.stdout


class TestArgvPreprocessing:
    """init 的 --agent 必须保留给初始化流程，不能被全局 JSON 别名吞掉。"""

    def test_init_with_agent_reaches_subcommand(self, clean_home):
        # 测试不访问真实平台
        # 使用不可达地址触发请求失败
        # 确认 --agent 进入初始化请求分支
        # 而非交互向导
        result = subprocess.run(
            [sys.executable, "-m", "memgo_cli", "init", "--agent"],
            capture_output=True,
            encoding="utf-8",
            env={
                **{k: v for k, v in os.environ.items() if not k.startswith("MEMGO_")},
                "HOME": clean_home,
                "MEMGO_BASE_URL": "http://127.0.0.1:1",  # 不可达测试地址
                "FORCE_COLOR": "0",
                "PYTHONIOENCODING": "utf-8",
            },
            timeout=15,
        )
        combined = _strip_ansi(result.stdout + result.stderr).lower()
        # 允许连接错误或初始化请求错误
        # 但必须证明代理模式分支已执行
        assert (
            "agent" in combined
            or "connect" in combined
            or "network" in combined
            or "fetch" in combined
            or "bootstrap" in combined
        ), f"Expected bootstrap attempt, got: {combined!r}"


class TestJsonEnvelopeParity:
    """在不可达后端下验证初始化失败，真实成功流程由本地 HTTP 验收覆盖。"""

    def test_init_agent_json_no_traceback_on_network_failure(self, clean_home):
        result = subprocess.run(
            [sys.executable, "-m", "memgo_cli", "init", "--agent", "--json"],
            capture_output=True,
            encoding="utf-8",
            env={
                **{k: v for k, v in os.environ.items() if not k.startswith("MEMGO_")},
                "HOME": clean_home,
                "MEMGO_BASE_URL": "http://127.0.0.1:1",
                "FORCE_COLOR": "0",
                "PYTHONIOENCODING": "utf-8",
            },
            timeout=15,
        )
        combined = _strip_ansi(result.stdout + result.stderr)
        assert "Traceback (most recent call last)" not in combined
        assert result.returncode != 0


class TestInitInCommandList:
    """顶层帮助公开 init 命令。"""

    def test_top_level_help_lists_init(self):
        result = _run(["--help"])
        assert result.returncode == 0
        assert "init" in result.stdout
