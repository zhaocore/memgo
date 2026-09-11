"""根据已知运行环境识别代理上下文，身份仍由显式参数声明。"""

from __future__ import annotations

import os

_AGENT_CALLER_ENV: tuple[tuple[str, tuple[str, ...]], ...] = (
    ("claude-code", ("CLAUDECODE", "CLAUDE_CODE")),
    ("cursor", ("CURSOR_AGENT", "CURSOR_SESSION_ID")),
    ("codex", ("CODEX_CLI", "OPENAI_CODEX")),
    ("cline", ("CLINE_AGENT", "CLINE")),
    ("continue", ("CONTINUE_AGENT", "CONTINUE_SESSION")),
    ("aider", ("AIDER_SESSION",)),
    ("goose", ("GOOSE_AGENT",)),
    ("windsurf", ("WINDSURF_AGENT",)),
)


def detect_agent_caller() -> str | None:
    """读取已知代理环境变量，返回规范名称或空值。"""
    for name, env_vars in _AGENT_CALLER_ENV:
        if any(os.environ.get(v) for v in env_vars):
            return name
    return None
