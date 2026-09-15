#!/usr/bin/env python3
"""Install MemGo lifecycle hooks into ~/.codex/hooks.json。

Codex 只发现 ~/.codex/hooks.json 或 <repo>/.codex/hooks.json 下的 hooks,
没有插件宿主机制自动接线。本安装器读取 hooks/codex-hooks.json 模板, 把
${PLUGIN_ROOT} 占位符改写为本集成目录的绝对路径, 再合并进 ~/.codex/hooks.json。

重复运行幂等: 已有的 MemGo 条目(按命令串里的 OWNER_MARKER 识别)先移除,
再写入新条目, 升级不会留重复。

Usage:
  python3 install_codex_hooks.py              # 安装或更新
  python3 install_codex_hooks.py --uninstall  # 移除 MemGo 条目

安装后, 本地 MCP 通过 <本目录>/.codex-mcp.json 注册, 指向共享 MCP server
(integrations/mcp/memgo_mcp.py), 需把 ${CODEX_PLUGIN_ROOT} 替换为绝对路径,
或把 .codex-mcp.json 合并进 ~/.codex/mcp.json。
"""

from __future__ import annotations

import argparse
import json
import platform
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
PLUGIN_ROOT = SCRIPT_DIR.parent

CODEX_DIR = Path.home() / ".codex"
HOOKS_FILE = CODEX_DIR / "hooks.json"
CONFIG_FILE = CODEX_DIR / "config.toml"

TEMPLATE_FILE = PLUGIN_ROOT / "hooks" / "codex-hooks.json"

# 识别本安装器拥有的条目用的子串。命令串里包含绝对路径, 取固定的
# "/codex/scripts/" 段, 跨安装路径稳定且不会误伤其他含 "codex" 的条目。
OWNER_MARKER = "/codex/scripts/"


def load_template() -> dict:
    raw = TEMPLATE_FILE.read_text()
    raw = raw.replace("${PLUGIN_ROOT}", str(PLUGIN_ROOT))
    return json.loads(raw)


def load_existing() -> dict:
    if not HOOKS_FILE.exists():
        return {"hooks": {}}
    try:
        return json.loads(HOOKS_FILE.read_text())
    except (json.JSONDecodeError, OSError) as e:
        print(f"error: failed to read {HOOKS_FILE}: {e}", file=sys.stderr)
        sys.exit(1)


def is_owned_entry(entry: dict) -> bool:
    for hook in entry.get("hooks", []):
        if OWNER_MARKER in hook.get("command", ""):
            return True
    return False


def strip_owned_entries(config: dict) -> dict:
    hooks = config.get("hooks", {}) or {}
    for event in list(hooks.keys()):
        hooks[event] = [e for e in hooks[event] if not is_owned_entry(e)]
        if not hooks[event]:
            del hooks[event]
    config["hooks"] = hooks
    return config


def merge_template(config: dict, template: dict) -> dict:
    hooks = config.setdefault("hooks", {})
    for event, entries in template.get("hooks", {}).items():
        hooks.setdefault(event, []).extend(entries)
    return config


def write_config(config: dict) -> None:
    CODEX_DIR.mkdir(parents=True, exist_ok=True)
    HOOKS_FILE.write_text(json.dumps(config, indent=2) + "\n")


def feature_flag_enabled() -> bool:
    if not CONFIG_FILE.exists():
        return False
    content = CONFIG_FILE.read_text()
    for line in content.splitlines():
        stripped = line.split("#", 1)[0].strip().replace(" ", "")
        if stripped == "codex_hooks=true":
            return True
    return False


def print_feature_flag_hint() -> None:
    print()
    print("Codex hooks feature flag is not enabled.")
    print(f"Add this to {CONFIG_FILE}:")
    print()
    print("  [features]")
    print("  codex_hooks = true")
    print()
    print("Then restart Codex.")


def print_mcp_hint() -> None:
    print()
    print(f"Local MCP server: {PLUGIN_ROOT / '.codex-mcp.json'}")
    print("Merge it into ~/.codex/mcp.json with ${CODEX_PLUGIN_ROOT} set to the")
    print(f"absolute path {PLUGIN_ROOT}, or copy it into the project. It points")
    print("at the shared stdio MCP server (../mcp/memgo_mcp.py).")
    print("Required env for MCP + hooks: MEMGO_API_KEY (and MEMGO_BASE_URL if not default).")


def main() -> int:
    parser = argparse.ArgumentParser(description="Install or remove MemGo Codex hooks.")
    parser.add_argument(
        "--uninstall",
        action="store_true",
        help="Remove MemGo entries from ~/.codex/hooks.json and exit.",
    )
    args = parser.parse_args()

    config = load_existing()

    if args.uninstall:
        config = strip_owned_entries(config)
        write_config(config)
        print(f"Removed MemGo hooks from {HOOKS_FILE}")
        return 0

    # Codex lifecycle hooks register .sh paths directly in ~/.codex/hooks.json.
    # On native Windows .sh has no default handler, so Codex spawning a hook
    # triggers "Open With" dialogs (one OpenWith.exe per event).
    if platform.system() == "Windows":
        print(
            "Codex lifecycle hooks register .sh scripts directly, which Windows\n"
            "cannot execute without a bash interpreter on PATH. Re-run this\n"
            "installer from WSL or Git Bash, or use MemGo via MCP only.\n",
            file=sys.stderr,
        )
        return 2

    if not TEMPLATE_FILE.exists():
        print(f"error: template not found at {TEMPLATE_FILE}", file=sys.stderr)
        return 1

    template = load_template()
    config = strip_owned_entries(config)
    config = merge_template(config, template)
    write_config(config)

    print(f"Installed MemGo hooks into {HOOKS_FILE}")
    print(f"Plugin path: {PLUGIN_ROOT}")
    print("Events: PreToolUse, SessionStart, UserPromptSubmit, PostToolUse, Stop, PreCompact")
    print_mcp_hint()

    if not feature_flag_enabled():
        print_feature_flag_hint()

    return 0


if __name__ == "__main__":
    sys.exit(main())
