"""验收同进程调用隔离和真实终端密钥输入。"""

from __future__ import annotations

import json
import os
import select
import subprocess
import sys
import time

import pytest
from typer.testing import CliRunner

from memgo_cli.app import create_app
from memgo_cli.runtime.state import get_current_command, is_agent_mode


def test_repeated_invocation_restores_output_mode_and_state():
    runner = CliRunner()
    app = create_app()
    first = runner.invoke(app, ["--json", "config", "show"])
    assert first.exit_code == 0, first.output
    assert json.loads(first.stdout)["status"] == "success"
    assert not is_agent_mode()
    assert get_current_command() == ""
    second = runner.invoke(app, ["config", "show"])
    assert second.exit_code == 0, second.output
    assert not second.stdout.lstrip().startswith("{")
    third = runner.invoke(app, ["--json", "config", "set", "invalid.key", "x"])
    assert third.exit_code == 1
    assert json.loads(third.stdout)["status"] == "error"
    assert not is_agent_mode()
    assert get_current_command() == ""


@pytest.mark.skipif(sys.platform == "win32", reason="POSIX 伪终端验收")
@pytest.mark.parametrize(
    "typed", [b"private-secret\r", b"discard\x15private-secreX\x7ft\r", b"\x03", b"\x04"]
)
def test_secret_prompt_masks_input_and_restores_terminal(tmp_path, typed):
    import pty

    master, slave = pty.openpty()
    code = """
import sys, termios
from memgo_cli.runtime.prompt import _prompt_secret
before = termios.tcgetattr(sys.stdin.fileno())
try:
    value = _prompt_secret("SECRET_READY:")
    assert value == "private-secret"
except (KeyboardInterrupt, EOFError):
    pass
finally:
    after = termios.tcgetattr(sys.stdin.fileno())
    # macOS 内核切换规范模式后可能自动设置 PENDIN，不属于调用方终端配置。
    before[3] &= ~termios.PENDIN
    after[3] &= ~termios.PENDIN
    assert after == before
print("RESTORED", flush=True)
"""
    env = {**os.environ, "HOME": str(tmp_path), "MEMGO_TELEMETRY": "false"}
    process = subprocess.Popen(
        [sys.executable, "-c", code], stdin=slave, stdout=slave, stderr=slave, env=env
    )
    os.close(slave)
    transcript = bytearray()
    sent = False
    deadline = time.monotonic() + 10
    try:
        while time.monotonic() < deadline:
            ready, _, _ = select.select([master], [], [], 0.1)
            if ready:
                try:
                    chunk = os.read(master, 4096)
                except OSError:
                    break
                if not chunk:
                    break
                transcript.extend(chunk)
                if not sent and b"SECRET_READY:" in transcript:
                    os.write(master, typed)
                    sent = True
            if process.poll() is not None and not ready:
                break
        assert sent, bytes(transcript)
        assert process.wait(timeout=2) == 0, bytes(transcript)
        assert b"RESTORED" in transcript
        assert b"private-secret" not in transcript
        assert b"discard" not in transcript
    finally:
        if process.poll() is None:
            process.kill()
            process.wait(timeout=2)
        os.close(master)
