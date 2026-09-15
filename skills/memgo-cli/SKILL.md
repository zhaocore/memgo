---
name: memgo-cli
description: >
  MemGo CLI -- the command-line interface for MemGo memory operations.
  TRIGGER when: user mentions "memgo cli", "memgo 命令", "memgo command line",
  "memgo-cli", "bin/memgo", or is running memgo commands in a terminal/shell
  (memgo add, memgo search, memgo list, memgo get, memgo init, memgo config,
  memgo import, memgo status). Also triggers when the query includes CLI flags
  like --user-id, --output, --json, --agent, --base-url, or describes
  bash/zsh/terminal/shell usage against a MemGo memory server.
  DO NOT TRIGGER when: user asks about programmatic SDK integration in Go/Python/TS
  code (use the memgo skill), or about the self-hosted server API / MCP wiring
  (use the memgo-integrate skill).
license: Apache-2.0
metadata:
  author: MemGo
  version: "1.0.0"
  category: ai-memory
  tags: "cli, terminal, memory, ai, command-line"
compatibility: Go build (`go build -o bin/memgo ./cli/go/cmd/memgo`) or a published binary; parallel implementations live in cli/python and cli/node. MEMGO_API_KEY env var.
---

# MemGo CLI

The command-line interface for the MemGo memory platform. Add, search, list, update, and delete memories from the terminal -- for developers, AI agents, and CI/CD pipelines. The Go CLI (`bin/memgo`) is the shipped implementation; `cli/python` and `cli/node` are parallel ports with the same command surface (parity locked by `tests/contract/cli_parity_golden.json`).

## Install

Build the Go binary from the MemGo repo:

```bash
cd /path/to/memgo
go build -o bin/memgo ./cli/go/cmd/memgo
```

Or use the prebuilt `bin/memgo` (and the server `bin/memgo-server`). The Python (`cli/python`) and Node (`cli/node`) ports install the same `memgo` binary name and share the same commands, flags, and output formats.

## Setup

**Interactive setup wizard:**

```bash
memgo init
```

Prompts for the API key and a default user ID. If `~/.memgo/config.json` already exists, pass `--force` to overwrite.

**Fully non-interactive (for agents and CI):**

```bash
memgo init --api-key <key> --user-id alice --force
```

**Or set the environment variables directly:**

```bash
export MEMGO_API_KEY="mgsk-xxx"
export MEMGO_BASE_URL="https://memgo.wxget.com"   # 自托管 server 根地址
```

`MEMGO_API_KEY` is required for every command. `MEMGO_BASE_URL` selects the backend:
- Host `api.memgo.ai` (or unset) → **platform** backend, `Authorization: Token` auth.
- Any other host (e.g. `https://memgo.wxget.com`) → **OSS** backend, `X-API-Key` auth, talking to the self-hosted MemGo server.

Never commit API keys, `.env`, or `~/.memgo/config.json`.

## Quick Reference

### Add a memory
```bash
memgo add "I prefer dark mode" --user-id alice
```

### Search memories
```bash
memgo search "preferences" --user-id alice
```

### List all memories for a user
```bash
memgo list --user-id alice
```

### Get a specific memory
```bash
memgo get <memory-id>
```

### Update a memory
```bash
memgo update <memory-id> "new text"
```

### Delete a single memory
```bash
memgo delete <memory-id> --force
```

### Delete all memories for a user
```bash
memgo delete --all --user-id alice --force
```

## Agent / JSON Mode

Use `--json` (or its alias `--agent`) for structured output suitable for LLM consumption. Every command wraps its response in a standard envelope on stdout:

```json
{
  "status": "success",
  "command": "search",
  "scope": { "user_id": "alice" },
  "count": 2,
  "data": [
    { "id": "mem-abc", "memory": "User prefers dark mode", "score": 0.92 }
  ]
}
```

On error the envelope carries `"status": "error"` and a message, with `data: null`. In agent mode all progress and error text go to stderr (or vanish), so stdout is always clean, parseable JSON. Note: the Go CLI envelope omits the `duration_ms` field present in the upstream Python/Node CLIs.

## Two Backends

The CLI detects the backend from the base URL:

| Base URL | Backend | Auth header | Target |
|----------|---------|-------------|--------|
| `https://api.memgo.ai` (default) | platform | `Authorization: Token <key>` | MemGo hosted platform |
| anything else, e.g. `https://memgo.wxget.com` | OSS | `X-API-Key: <key>` | self-hosted MemGo server |

The OSS backend is synchronous (no event polling) and has a smaller surface: no `app_id`, no `--immutable`, no `--custom-categories`/`--structured-data-schema`, no `--keyword`, no pagination/category/date filters on `list`, no events, and entity types are limited to `user` / `agent` / `run`. Commands that use an unsupported feature fail with a clear message rather than silently degrading.

## Common Edge Cases

- **Missing API key:** every command calls `memgo init` or sets `MEMGO_API_KEY`. A missing or invalid key produces `No API key configured.` / `Invalid or expired API key.`
- **OSS `search` requires a scope:** the self-hosted contract rejects a query with no `user_id` / `agent_id` / `run_id` filter. Pass at least one scope flag.
- **Delete modes are mutually exclusive:** `memgo delete <id>` (single), `memgo delete --all` (bulk), and `memgo delete --entity` (cascade) cannot be combined. Confirmation prompts `[y/N]` unless `--force`; non-TTY and agent mode always deny.
- **`--project` is platform-only:** on the OSS backend, `memgo delete --all --project` errors and points at the server admin `POST /reset` endpoint.
- **`config get` prints raw values:** unlike `config show`, `memgo config get platform.api_key` prints the full key, not a redacted form. Handle output carefully in shared logs.
- **Entity ID resolution:** if you pass any explicit scope flag (`--user-id`, `--agent-id`, `--app-id`, `--run-id`), the CLI uses ONLY the explicit IDs and ignores config defaults. If none are given, all configured defaults apply.
- **Stdin detection:** when no text argument is provided and input is piped (not a TTY), the CLI reads one line from stdin. Works with `add`, `search`, and `update`.

## References

Load these on demand for deeper detail:

| Topic | File |
|-------|------|
| Command reference (all commands, flags, options, examples) | [references/command-reference.md](references/command-reference.md) |
| Configuration (config file, env vars, precedence, init wizard) | [references/configuration.md](references/configuration.md) |
| Workflows (piping, scripting, CI/CD, agent mode recipes) | [references/workflows.md](references/workflows.md) |

## Related MemGo Skills

| Skill | When to use |
|-------|-------------|
| memgo | SDK / programmatic integration in Go/Python/TS, REST API |
| memgo-integrate | Self-hosted server wiring: Claude Code / Codex / DeepSeek plugins, shared MCP server |
