"""验证 telemetry 的行为与兼容性。"""

from __future__ import annotations

import io
import json
import subprocess
import sys

import pytest

from memgo_cli.config.models import MemGoConfig
from memgo_cli.config.store import save_config
from memgo_cli.integrations.telemetry import capture_event
from memgo_cli.integrations.telemetry_sender import _load_context


class _CaptureStdin:
    def __init__(self):
        self.buffer = ""
        self.closed = False

    def write(self, value: str) -> None:
        self.buffer += value

    def close(self) -> None:
        self.closed = True


class _DummyProcess:
    def __init__(self):
        self.stdin = _CaptureStdin()


def test_capture_event_writes_context_to_stdin_not_argv(isolate_config, monkeypatch):
    monkeypatch.setenv("MEMGO_TELEMETRY", "true")
    config = MemGoConfig()
    config.platform.api_key = "m0-test-secret"
    config.telemetry.anonymous_id = "cli-anon-test"
    save_config(config)

    captured: dict[str, object] = {}
    proc = _DummyProcess()

    def fake_popen(args, **kwargs):
        captured["args"] = args
        captured["kwargs"] = kwargs
        return proc

    monkeypatch.setattr("memgo_cli.integrations.telemetry.subprocess.Popen", fake_popen)

    capture_event("unit_test_event", {"case": "stdin-secret"}, pre_resolved_email=None)

    argv = captured["args"]
    assert argv == [sys.executable, "-m", "memgo_cli.integrations.telemetry_sender"]
    assert all("m0-test-secret" not in arg for arg in argv)

    kwargs = captured["kwargs"]
    assert kwargs["stdin"] == subprocess.PIPE
    assert kwargs["text"] is True

    ctx = json.loads(proc.stdin.buffer)
    assert ctx["memgo_api_key"] == "m0-test-secret"
    assert ctx["payload"]["event"] == "unit_test_event"

    assert proc.stdin.closed


def test_load_context_reads_from_stdin(monkeypatch):
    monkeypatch.setattr("sys.argv", ["telemetry_sender"])
    value = {
        "payload": {"event": "stdin", "api_key": "test-public", "distinct_id": "test-user"},
        "posthog_host": "http://localhost/capture",
        "memgo_api_key": "test-key",
        "memgo_base_url": "http://localhost",
        "config_path": "",
        "needs_email": False,
    }
    monkeypatch.setattr("sys.stdin", io.StringIO(json.dumps(value)))

    ctx = _load_context()

    assert ctx.payload["event"] == "stdin"


def test_load_context_rejects_argv_credentials(monkeypatch):
    monkeypatch.setattr("sys.argv", ["telemetry_sender", '{"payload": {"event": "argv"}}'])
    monkeypatch.setattr("sys.stdin", io.StringIO(""))

    with pytest.raises(ValueError):
        _load_context()
