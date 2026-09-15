# MemGo CLI Configuration

Everything about configuring the memgo CLI: config file format, environment variables, the init wizard, and precedence rules. The Go CLI (`bin/memgo`) and the Python/Node ports share the same config location and schema.

---

## Config File Location

| Path | Permissions | Description |
|------|-------------|-------------|
| `~/.memgo/` | `0700` (owner rwx) | Config directory. Created automatically by `memgo init`. |
| `~/.memgo/config.json` | `0600` (owner rw) | Config file. Contains the API key, defaults, and platform settings. |

The restricted permissions ensure the API key is not world-readable.

---

## Config File Schema

```json
{
  "version": 1,
  "defaults": {
    "user_id": "",
    "agent_id": "",
    "app_id": "",
    "run_id": ""
  },
  "platform": {
    "api_key": "",
    "base_url": "https://api.memgo.ai",
    "user_email": "",
    "agent_mode": false,
    "created_via": "",
    "agent_caller": "",
    "claimed_at": "",
    "default_user_id": ""
  },
  "telemetry": {
    "anonymous_id": ""
  },
  "agent_rush": {
    "acknowledged_at": ""
  }
}
```

### Field Reference

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `version` | integer | `1` | Config schema version. |
| `defaults.user_id` | string | `""` | Default user ID for scoping commands. |
| `defaults.agent_id` | string | `""` | Default agent ID for scoping commands. |
| `defaults.app_id` | string | `""` | Default app ID for scoping commands. |
| `defaults.run_id` | string | `""` | Default run ID for scoping commands. |
| `platform.api_key` | string | `""` | API key for the MemGo platform. |
| `platform.base_url` | string | `"https://api.memgo.ai"` | Base URL for API requests. Selects the backend (see below). |
| `platform.user_email` | string | `""` | Account email, discovered from the server on ping. |
| `platform.agent_mode` | boolean | `false` | True while an unattended Agent Mode key is unclaimed. |
| `platform.created_via` | string | `""` | Signup channel: `agent_mode`, `email`, `api_key`, `existing_key`. |
| `platform.agent_caller` | string | `""` | Agent identity recorded during Agent Mode signup (e.g. `claude-code`). |
| `platform.claimed_at` | string | `""` | ISO timestamp when the user claimed the key. |
| `platform.default_user_id` | string | `""` | `user_<slug>` returned at init, used as the default scope. |
| `telemetry.anonymous_id` | string | `""` | Persistent anonymous telemetry id. |
| `agent_rush.acknowledged_at` | string | `""` | AgentRush public-memory consent timestamp. |

---

## `memgo init` Wizard

The Go CLI implements one flow: API key. The email-login and Agent Mode flows (`--email`, `--code`, `--agent`, `--agent-caller`) are declared but **not migrated** -- using them prints:

```
✗ Error: --email/--code/--agent/--agent-caller flows are not migrated to the Go CLI yet.
  Use --api-key from https://app.memgo.ai
```

Those flows exist in the Python and Node ports.

### API Key Flow (default)

```bash
# Fully interactive:
memgo init

# Fully non-interactive:
memgo init --api-key mgsk-xxx --user-id alice --force
```

**Interactive mode steps:**

1. Checks for existing config. If found with an API key and no `--force`, errors: `Config already exists. Use --force to overwrite.`
2. Prompts for the API key (from `https://app.memgo.ai`).
3. Prompts for a default user ID (optional; empty to skip).
4. Saves config to `~/.memgo/config.json` with `0600` permissions.
5. Prints `Config saved to ~/.memgo/config.json`.

**Non-interactive mode:** with `--api-key` (and optionally `--user-id`) the wizard skips all prompts. When stdin is not a TTY and `--api-key` is missing, the prompt read returns empty and it errors with `API key is required.`

### Force Overwrite

If `~/.memgo/config.json` already exists with an API key, `memgo init` errors. Use `--force` to skip:

```bash
memgo init --api-key mgsk-new-key --user-id alice --force
```

---

## `memgo config` Subcommands

### `memgo config show`

Displays the current configuration. Text mode prints the raw JSON object with the API key redacted; `-o json` wraps it in the agent envelope (`command: "config.show"`).

```bash
memgo config show
memgo config show -o json
```

### `memgo config get <key>`

Reads a single configuration value by dotted key or short alias. Prints the value as-is -- the API key is NOT redacted here.

```bash
memgo config get platform.api_key     # 完整 key, 未脱敏
memgo config get user_id              # 短别名 = defaults.user_id
```

**Valid keys (dotted or short alias):**
- `platform.api_key` / `api_key`
- `platform.base_url` / `base_url`
- `platform.user_email` / `user_email`
- `defaults.user_id` / `user_id`
- `defaults.agent_id` / `agent_id`
- `defaults.app_id` / `app_id`
- `defaults.run_id` / `run_id`

Unknown keys print `Unknown config key: <key>` and exit non-zero.

### `memgo config set <key> <value>`

Sets a configuration value and saves the config file.

```bash
memgo config set user_id alice
memgo config set platform.base_url https://memgo.wxget.com
```

**Type coercion:** boolean fields accept `true` / `1` / `yes` (case-insensitive) as true, anything else as false; integer fields are parsed with `int`; strings are stored as-is.

### `memgo config clear`

The Go CLI has **no** `config clear` subcommand (present in the upstream Python/Node CLIs). To reset, remove `~/.memgo/config.json` directly.

---

## Environment Variables

Environment variables override config file values but are overridden by CLI flags (`--api-key`, `--base-url`, scope flags).

| Variable | Config Path | Type | Default |
|----------|-------------|------|---------|
| `MEMGO_API_KEY` | `platform.api_key` | string | `""` |
| `MEMGO_BASE_URL` | `platform.base_url` | string | `"https://api.memgo.ai"` |
| `MEMGO_USER_ID` | `defaults.user_id` | string | `""` |
| `MEMGO_AGENT_ID` | `defaults.agent_id` | string | `""` |
| `MEMGO_APP_ID` | `defaults.app_id` | string | `""` |
| `MEMGO_RUN_ID` | `defaults.run_id` | string | `""` |

`MEMGO_TELEMETRY=false` disables telemetry. Telemetry is fire-and-forget and never blocks a command.

---

## Backend Selection (base URL)

The CLI picks its protocol from the configured `base_url` host:

| Host | Backend | Auth header | Notes |
|------|---------|-------------|-------|
| `api.memgo.ai` or empty | platform | `Authorization: Token <key>` | MemGo hosted platform; full feature set, async events. |
| any other host (e.g. `https://memgo.wxget.com`) | OSS | `X-API-Key: <key>` | Self-hosted MemGo server; synchronous writes, reduced surface. |

To talk to the MemGo self-hosted server, set `MEMGO_BASE_URL` (or `platform.base_url`) to the server root, e.g. `https://memgo.wxget.com`:

```bash
export MEMGO_API_KEY="mgsk-xxx"
export MEMGO_BASE_URL="https://memgo.wxget.com"
memgo status
```

---

## Precedence

Configuration values are resolved in this order (highest priority first):

```
1. CLI flags        --api-key, --base-url, --user-id, etc.
2. Environment vars MEMGO_API_KEY, MEMGO_BASE_URL, MEMGO_USER_ID, etc.
3. Config file      ~/.memgo/config.json
4. Defaults         Hardcoded defaults (empty strings, https://api.memgo.ai)
```

**Example:** config file has `user_id: "bob"`, env var `MEMGO_USER_ID=charlie` is set, and you pass `--user-id alice` on the command line -- the effective `user_id` is `alice`.

---

## API Key Redaction Rules

`memgo config show` and status output redact the API key with `RedactKey`:

| Condition | Output |
|-----------|--------|
| Empty string | `(not set)` |
| Length <= 8 | First 2 characters + `***` |
| Length > 8 | First 4 characters + `...` + last 4 characters |

**Examples:**
- `""` -> `(not set)`
- `"mg-abc"` -> `mg***`
- `"mgsk_abcdefghijklmnop"` -> `mgsk...mnop`

Note: `memgo config get platform.api_key` does NOT redact -- it prints the raw value.

---

## Dotted Key Map

The `config get` and `config set` commands accept the full dotted path or a short alias:

| Dotted Key | Short Alias | Section | Field |
|------------|-------------|---------|-------|
| `platform.api_key` | `api_key` | platform | api_key |
| `platform.base_url` | `base_url` | platform | base_url |
| `platform.user_email` | `user_email` | platform | user_email |
| `defaults.user_id` | `user_id` | defaults | user_id |
| `defaults.agent_id` | `agent_id` | defaults | agent_id |
| `defaults.app_id` | `app_id` | defaults | app_id |
| `defaults.run_id` | `run_id` | defaults | run_id |
