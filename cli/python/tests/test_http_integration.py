"""用本地 HTTP 服务验收真实 CLI 进程，不调用外部平台。"""

from __future__ import annotations

import json
import os
import subprocess
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from threading import Thread
from urllib.parse import urlsplit

import pytest


@pytest.fixture
def platform_server():
    records = []
    replies = {}

    class Handler(BaseHTTPRequestHandler):
        def log_message(self, format, *args):
            """关闭测试服务访问日志，避免输出测试请求内容。"""

        def dispatch(self):
            body = self.rfile.read(int(self.headers.get("Content-Length", "0")))
            records.append((self.command, self.path, dict(self.headers), body))
            path = urlsplit(self.path).path
            status, value = replies.get(
                (self.command, path), (404, {"error": "Unexpected test route"})
            )
            payload = value if isinstance(value, bytes) else json.dumps(value).encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(payload)))
            self.end_headers()
            self.wfile.write(payload)

        do_GET = do_POST = do_PUT = do_PATCH = do_DELETE = dispatch

    server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
    thread = Thread(target=server.serve_forever, daemon=True)
    thread.start()
    replies[("GET", "/v1/ping/")] = (200, {"user_email": "local@example.test"})
    try:
        yield f"http://127.0.0.1:{server.server_port}", records, replies
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)


@pytest.fixture
def run_cli(tmp_path, platform_server):
    base_url, _records, _replies = platform_server
    env = {key: value for key, value in os.environ.items() if not key.startswith("MEMGO_")}
    env.update(
        HOME=str(tmp_path),
        MEMGO_TELEMETRY="false",
        MEMGO_BASE_URL=base_url,
        MEMGO_API_KEY="local-test-key",
        NO_COLOR="1",
        PYTHONIOENCODING="utf-8",
    )
    env.pop("PYTHONPATH", None)

    def run(args, expected=0):
        result = subprocess.run(
            [sys.executable, "-m", "memgo_cli", *args],
            input="",
            text=True,
            capture_output=True,
            env=env,
            timeout=15,
            cwd=tmp_path,
        )
        assert result.returncode == expected, (result.stdout, result.stderr)
        assert "Traceback" not in result.stderr
        return result

    return run, env


def test_crud_protocol_and_json_output(run_cli, platform_server):
    run, _env = run_cli
    _url, records, replies = platform_server
    memory = {"id": "mem/a?b#c", "memory": "喜欢中文", "user_id": "alice"}
    replies.update(
        {
            ("POST", "/v3/memories/add/"): (200, {"results": [memory], "memgo_notice": "notice"}),
            ("POST", "/v3/memories/search/"): (200, {"results": [memory]}),
            ("POST", "/v3/memories/"): (200, {"results": [memory]}),
            ("GET", "/v1/memories/mem%2Fa%3Fb%23c/"): (200, memory),
            ("PUT", "/v1/memories/mem%2Fa%3Fb%23c/"): (200, {**memory, "memory": "更新"}),
            ("DELETE", "/v1/memories/mem%2Fa%3Fb%23c/"): (204, b""),
        }
    )
    commands = [
        ["add", "喜欢中文", "-u", "alice"],
        ["search", "偏好", "-u", "alice", "--latest-only"],
        ["list", "-u", "alice"],
        ["get", memory["id"]],
        ["update", memory["id"], "更新"],
        ["delete", memory["id"], "--force", "--delete-linked"],
    ]
    for args in commands:
        result = run([*args, "--json"])
        envelope = json.loads(result.stdout)
        assert envelope["status"] == "success"
        assert envelope["command"] == args[0]
        if args[0] == "add":
            assert envelope["memgo_notice"] == "notice"
    requests = [record for record in records if not record[1].startswith("/v1/ping/")]
    assert len(requests) == 6
    for _method, _path, headers, _body in requests:
        assert headers["Authorization"] == "Token local-test-key"
        assert headers["X-MemGo-Client-Language"] == "python"
        assert headers["X-MemGo-Caller-Type"] == "agent"
    assert json.loads(requests[0][3])["messages"] == [{"role": "user", "content": "喜欢中文"}]
    assert json.loads(requests[1][3])["filters"] == {"user_id": "alice"}
    assert json.loads(requests[1][3])["latest_only"] is True
    assert "delete_linked=true" in requests[-1][1]
    assert requests[-1][3] == b""


@pytest.mark.parametrize(
    "status,payload",
    [(503, {"error": "offline", "api_key": "local-test-key"}), (200, b"bad json"), (200, [])],
)
def test_preflight_failures_are_json_and_do_not_write(run_cli, platform_server, status, payload):
    run, _env = run_cli
    _url, records, replies = platform_server
    replies[("GET", "/v1/ping/")] = (status, payload)
    result = run(["--json", "add", "content"], expected=1)
    envelope = json.loads(result.stdout)
    assert envelope["status"] == "error"
    assert envelope["command"] == "add"
    assert "local-test-key" not in result.stdout + result.stderr
    assert all(record[0] == "GET" for record in records)


def test_failed_init_does_not_save_key(run_cli, platform_server, tmp_path):
    run, _env = run_cli
    _url, _records, replies = platform_server
    replies[("GET", "/v1/ping/")] = (401, {"detail": "invalid"})
    run(["init", "--api-key", "local-test-key", "--user-id", "alice"], expected=1)
    assert not (tmp_path / ".memgo/config.json").exists()


def test_project_dry_run_never_deletes(run_cli, platform_server):
    run, _env = run_cli
    _url, records, _replies = platform_server
    result = run(["delete", "--all", "--project", "--dry-run", "--force", "--json"], expected=1)
    assert "--dry-run" in json.loads(result.stdout)["error"]
    assert all(record[0] == "GET" for record in records)


@pytest.mark.parametrize(
    "options", [["--messages", '[{"role":"user"}]'], ["--metadata", "[]"], ["--messages", "42"]]
)
def test_invalid_inputs_never_reach_add(run_cli, platform_server, options):
    run, _env = run_cli
    _url, records, _replies = platform_server
    run(["add", "text", *options, "--json"], expected=1)
    assert all(record[0] == "GET" for record in records)


def test_bootstrap_claim_and_private_config(run_cli, platform_server, tmp_path):
    run, env = run_cli
    env.pop("MEMGO_API_KEY")
    _url, records, replies = platform_server
    replies[("POST", "/api/v1/auth/agent_mode/")] = (
        200,
        {"api_key": "local-new-key", "default_user_id": "user_local"},
    )
    replies[("POST", "/api/v1/auth/email_code/verify/")] = (
        200,
        {"claimed": True, "claimed_at": "2026-09-11T00:00:00Z"},
    )
    result = run(["init", "--agent", "--agent-caller", "test-agent", "--json"])
    assert json.loads(result.stdout)["data"]["agent_mode"] is True
    path = tmp_path / ".memgo/config.json"
    config = json.loads(path.read_text())
    assert config["platform"]["api_key"] == "local-new-key"
    assert config["defaults"]["user_id"] == "user_local"
    assert path.stat().st_mode & 0o777 == 0o600
    result = run(["init", "--email", "local@example.test", "--code", "123456", "--json"])
    assert json.loads(result.stdout)["data"]["claimed"] is True
    config = json.loads(path.read_text())
    assert config["platform"]["agent_mode"] is False
    assert config["platform"]["api_key"] == "local-new-key"
    assert json.loads(records[-1][3])["agent_mode_api_key"] == "local-new-key"


def test_telemetry_sender_uses_stdin_and_local_http(run_cli, platform_server, tmp_path):
    _run, env = run_cli
    url, records, replies = platform_server
    replies[("POST", "/capture")] = (200, {})
    context = {
        "payload": {
            "api_key": "local-public-key",
            "event": "acceptance",
            "distinct_id": "local-user",
            "properties": {},
        },
        "posthog_host": url + "/capture",
        "needs_email": False,
        "memgo_api_key": "local-test-key",
        "memgo_base_url": url,
        "config_path": "",
    }
    result = subprocess.run(
        [sys.executable, "-m", "memgo_cli.integrations.telemetry_sender"],
        input=json.dumps(context),
        text=True,
        capture_output=True,
        env=env,
        cwd=tmp_path,
        timeout=15,
    )
    assert result.returncode == 0, result.stderr
    assert json.loads(records[-1][3])["event"] == "acceptance"
    assert "local-test-key" not in result.stdout + result.stderr


def test_installed_console_entrypoint(run_cli, tmp_path):
    _run, env = run_cli
    entry = Path(sys.executable).with_name("memgo")
    result = subprocess.run(
        [str(entry), "--version"], capture_output=True, text=True, env=env, cwd=tmp_path, timeout=15
    )
    assert result.returncode == 0, result.stderr
    assert "CLI v" in result.stdout


def test_import_failure_reports_counts_and_nonzero_exit(run_cli, platform_server, tmp_path):
    run, _env = run_cli
    _url, _records, replies = platform_server
    replies[("POST", "/v3/memories/add/")] = (503, {"error": "maintenance"})
    path = tmp_path / "import.json"
    path.write_text(json.dumps([{"memory": "first"}, {"memory": "second"}]))
    result = run(["import", str(path), "--json"], expected=1)
    envelope = json.loads(result.stdout)
    assert envelope["status"] == "error"
    assert envelope["data"] == {"added": 0, "failed": 2}
    assert "HTTP 503" in envelope["error"]


def test_import_validates_all_records_before_writing(run_cli, platform_server, tmp_path):
    run, _env = run_cli
    _url, records, _replies = platform_server
    path = tmp_path / "import.json"
    path.write_text(json.dumps([{"memory": "first"}, {"memory": 42}]))
    run(["import", str(path), "--json"], expected=1)
    assert all(record[0] == "GET" for record in records)
