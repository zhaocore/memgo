# MemGo CLI Command Reference

Complete reference for every command, argument, flag, and output mode in the memgo CLI. The shipped binary is the Go implementation (`bin/memgo`, v0.2.12); `cli/python` and `cli/node` are parallel ports with the same command surface. Where the Go CLI deviates from the upstream Python/Node CLIs, this document describes the Go behavior (the binary users actually run).

---

## Global Options

| Flag | Type | Description |
|------|------|-------------|
| `--json` | boolean | Agent mode: wrap all output in a structured JSON envelope on stdout. Spinners/progress go to stderr. |
| `--agent` | boolean | Alias of `--json`. |
| `-h, --help` | boolean | Show help for a command. |

Most commands also accept connection overrides (highest precedence over env and config file):

| Flag | Type | Description |
|------|------|-------------|
| `--api-key <key>` | string | Override the API key for this invocation. |
| `--base-url <url>` | string | Override the API base URL. Also selects the backend (platform vs OSS). |

Version is a subcommand, not a flag: `memgo version`.

---

## Commands

### `memgo add`

Add a memory from text, messages, file, or stdin.

**Usage:** `memgo add [text] [flags]`

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `text` | string | No | Text content to add as a memory. |

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-u, --user-id <id>` | string | - | Scope to user. |
| `--agent-id <id>` | string | - | Scope to agent. |
| `--app-id <id>` | string | - | Scope to app. |
| `--run-id <id>` | string | - | Scope to run. |
| `--messages <json>` | string | - | Conversation messages as JSON array (e.g. `'[{"role":"user","content":"..."}]'`). |
| `-f, --file <path>` | path | - | Read messages from a JSON file. |
| `-m, --metadata <json>` | string | - | Custom metadata as JSON object (e.g. `'{"source":"cli"}'`). |
| `--immutable` | boolean | false | Prevent future updates. |
| `--no-infer` | boolean | false | Skip inference; store the text verbatim. |
| `--expires <date>` | string | - | Expiration date (YYYY-MM-DD). |
| `--custom-instructions <text>` | string | - | Custom instructions for fact extraction. |
| `--agent-custom-instructions <text>` | string | - | Extraction instructions for agent-scoped memories, overriding the project setting. |
| `--custom-categories <json>` | string | - | Custom categories as a JSON array of `{name, description}` objects. |
| `--structured-data-schema <json>` | string | - | Schema for structured data extraction, as JSON. |
| `--timestamp <unix>` | int | - | Unix timestamp for the memory. |
| `--categories <cats>` | string | - | NOT supported on add. Errors; use `--custom-categories` instead. |
| `-o, --output <fmt>` | string | `text` | Output format: `text`, `json`, `quiet`. |
| `--api-key`, `--base-url` | string | - | Connection overrides. |

**Input priority:** `--file` > `--messages` > text argument > stdin (if piped and no text). Text content is wrapped as `[{"role":"user","content":"<text>"}]` before sending to the API; messages from `--messages` or `--file` are sent as-is.

**Output events:** the API returns results with an `event` field per memory:

| Event | Meaning |
|-------|---------|
| `ADD` | New memory created |
| `UPDATE` | Existing memory updated (deduplication) |
| `DELETE` | Existing memory removed (contradiction) |
| `NOOP` | No change needed |
| `PENDING` | Processing asynchronously in background (platform) |

On the OSS backend the write is synchronous: results come back immediately, no `PENDING`, no polling.

**OSS limitations:** `--app-id`, `--immutable`, `--custom-categories`, `--structured-data-schema`, and `--agent-custom-instructions` are rejected with a clear error. `--expires` and `--timestamp` are supported (map to `expiration_date` and `timestamp`).

**Examples:**
```bash
memgo add "I prefer dark mode" --user-id alice
memgo add "allergic to nuts" -u alice -m '{"source":"onboarding"}'
memgo add --messages '[{"role":"user","content":"I like Python"}]' -u alice
memgo add --file conversation.json -u alice -o json
echo "I prefer dark mode" | memgo add -u alice
memgo add "temporary note" -u alice --expires 2025-12-31
memgo add "important fact" -u alice --immutable
```

---

### `memgo search`

Search memories by semantic query (or keyword / hybrid, platform only).

**Usage:** `memgo search [query] [flags]`

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `query` | string | No | The search query. Falls back to stdin if piped, else errors if empty. |

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-u, --user-id <id>` | string | - | Filter by user. |
| `--agent-id <id>` | string | - | Filter by agent. |
| `--app-id <id>` | string | - | Filter by app. |
| `--run-id <id>` | string | - | Filter by run. |
| `-k, --top-k <n>` | integer | 10 | Number of results. |
| `--limit <n>` | integer | 10 | Alias of `--top-k`. |
| `--threshold <score>` | float | 0.3 | Minimum similarity score (0.0 to 1.0). |
| `--rerank` | boolean | false | Enable reranking for improved relevance (Platform only). |
| `--keyword` | boolean | false | Use keyword search instead of semantic. |
| `--filter <json>` | string | - | Advanced filter expression as JSON (`{"AND":[...]}` / `{"OR":[...]}`). |
| `--fields <list>` | string | - | Comma-separated list of fields to return. |
| `--show-expired` | boolean | false | Include expired memories. |
| `--reference-date <date>` | string | - | Reference date for relative queries (YYYY-MM-DD or unix timestamp). |
| `--latest-only` | boolean | false | Only return the latest version of each memory. |
| `-o, --output <fmt>` | string | `text` | Output format: `text`, `json`, `table`. |
| `--api-key`, `--base-url` | string | - | Connection overrides. |

**OSS contract:** the query must carry at least one of `user_id` / `agent_id` / `run_id` in its filter, otherwise the server rejects it with HTTP 400. `--keyword`, `--reference-date`, and `--latest-only` are not supported on OSS; `--rerank` is silently ignored. The scope flags are folded into the `filters` map sent to the server.

**Examples:**
```bash
memgo search "preferences" --user-id alice
memgo search "tools" -u alice -o json -k 5
memgo search "dietary restrictions" -u alice --threshold 0.5
memgo search "project setup" -u alice --rerank
memgo search "preferences" -u alice --filter '{"AND":[{"created_at":{"gte":"2025-01-01"}}]}'
echo "preferences" | memgo search -u alice
```

---

### `memgo get`

Get a specific memory by ID.

**Usage:** `memgo get <memory_id> [flags]`

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `memory_id` | string | Yes | The ID of the memory to retrieve. |

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-o, --output <fmt>` | string | `text` | Output format: `text`, `json`. |
| `--api-key`, `--base-url` | string | - | Connection overrides. |

A missing memory returns a not-found error.

**Examples:**
```bash
memgo get abc-123-def-456
memgo get abc-123-def-456 -o json
```

---

### `memgo list`

List memories with optional filters and pagination.

**Usage:** `memgo list [flags]`

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-u, --user-id <id>` | string | - | Filter by user. |
| `--agent-id <id>` | string | - | Filter by agent. |
| `--app-id <id>` | string | - | Filter by app. |
| `--run-id <id>` | string | - | Filter by run. |
| `--page <n>` | integer | 1 | Page number. |
| `--page-size <n>` | integer | 100 | Results per page. |
| `--category <name>` | string | - | Filter by category. |
| `--after <date>` | string | - | Created after (YYYY-MM-DD). |
| `--before <date>` | string | - | Created before (YYYY-MM-DD). |
| `--show-expired` | boolean | false | Include expired memories. |
| `--latest-only` | boolean | false | Only return the latest version of each memory. |
| `-o, --output <fmt>` | string | `table` | Output format: `text`, `json`, `table`. |
| `--api-key`, `--base-url` | string | - | Connection overrides. |

**OSS limitations:** `--page > 1`, `--category`, `--after`, `--before`, and `--latest-only` are not supported and error out. `--page-size` maps to the server `top_k` parameter.

**Examples:**
```bash
memgo list -u alice
memgo list --category prefs --after 2025-01-01 -o json
memgo list -u alice --page 2 --page-size 50
memgo list --before 2025-06-01 -o table
```

---

### `memgo update`

Update a memory's text or metadata.

**Usage:** `memgo update <memory_id> [text] [flags]`

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `memory_id` | string | Yes | The ID of the memory to update. |
| `text` | string | No | New memory text. Falls back to one line of stdin if piped and no text given. |

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-m, --metadata <json>` | string | - | Update metadata (JSON object). |
| `--expires <date>` | string | - | Expiration date (YYYY-MM-DD). |
| `--timestamp <unix>` | int | - | Unix timestamp for the memory. |
| `-o, --output <fmt>` | string | `text` | Output format: `text`, `json`, `quiet`. |
| `--api-key`, `--base-url` | string | - | Connection overrides. |

The OSS backend supports `text`, `metadata`, and `expiration_date`; `--timestamp` is rejected.

**Examples:**
```bash
memgo update abc-123 "new text"
memgo update abc-123 --metadata '{"priority":"high"}'
memgo update abc-123 "new text" -m '{"priority":"high"}'
echo "new text" | memgo update abc-123
```

---

### `memgo delete`

Delete a memory, all memories matching a scope, or an entity. Three mutually exclusive modes.

**Usage:** `memgo delete [memory_id] [flags]`

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `memory_id` | string | No | Memory ID to delete (omit when using `--all` or `--entity`). |

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--all` | boolean | false | Delete all memories matching scope filters. |
| `--entity` | boolean | false | Delete the entity itself and all its memories (cascade). |
| `--project` | boolean | false | With `--all`: delete ALL memories project-wide (platform only). |
| `--dry-run` | boolean | false | Show what would be deleted without actually deleting. |
| `--force` | boolean | false | Skip confirmation prompt. |
| `--delete-linked` | boolean | false | Also delete memories linked to this memory (platform only). |
| `-u, --user-id <id>` | string | - | Scope to user. |
| `--agent-id <id>` | string | - | Scope to agent. |
| `--app-id <id>` | string | - | Scope to app. |
| `--run-id <id>` | string | - | Scope to run. |
| `-o, --output <fmt>` | string | `text` | Output format: `text`, `json`, `quiet`. |
| `--api-key`, `--base-url` | string | - | Connection overrides. |

**Three modes (mutually exclusive):**

1. **Single memory:** `memgo delete <memory_id>` -- deletes one memory by its ID.
2. **Bulk delete:** `memgo delete --all [scope flags]` -- deletes all memories matching the scope.
3. **Entity cascade:** `memgo delete --entity [scope flags]` -- deletes the entity itself AND all its memories.

You cannot combine `<memory_id>` with `--all` or `--entity`, and you cannot combine `--all` with `--entity`. If none are provided, the command prints a usage hint and exits with an error.

**Confirmation:** without `--force`, every destructive mode prompts `[y/N]`. Non-TTY input and agent mode (`--json`/`--agent`) always deny the prompt.

**OSS behavior:** `--all` requires at least one scope id (user/agent/run). `--project` and `--delete-linked` are platform-only and error on OSS; project-wide reset on OSS goes through the server admin `POST /reset`.

**Examples:**
```bash
memgo delete abc-123-def-456 --force
memgo delete --all -u alice --force
memgo delete --all --project --force
memgo delete --entity -u alice --force
memgo delete abc-123 --dry-run
memgo delete --all -u alice --dry-run
```

---

### `memgo import`

Import memories from a JSON file.

**Usage:** `memgo import <file_path> [flags]`

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `file_path` | string | Yes | Path to a JSON file containing memories. |

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-u, --user-id <id>` | string | - | Override user ID for all imported items. |
| `--agent-id <id>` | string | - | Override agent ID for all imported items. |
| `-o, --output <fmt>` | string | `text` | Output format: `text`, `json`. |
| `--api-key`, `--base-url` | string | - | Connection overrides. |

**File format:** a JSON array where each item carries the memory text in a `memory` field (falling back to `text`). Items without text are skipped. CLI-provided `--user-id`/`--agent-id` apply to every imported item.

**Import format example:**
```json
[
  { "memory": "Prefers dark mode", "user_id": "alice" },
  { "text": "Allergic to nuts", "metadata": { "source": "intake" } }
]
```

**Behavior:** iterates through items, calling the add API for each, and reports the total imported count.

**Examples:**
```bash
memgo import memories.json --user-id alice
memgo import data.json -u alice -o json
```

---

### `memgo config`

Manage memgo configuration.

#### `memgo config show`

Displays the current configuration with the API key redacted. Text mode prints the raw JSON config object; `-o json` wraps it in the agent envelope.

**Usage:** `memgo config show [flags]`

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-o, --output <fmt>` | string | `text` | Output format: `text`, `json`. |

**Example:**
```bash
memgo config show
memgo config show -o json
```

#### `memgo config get <key>`

Reads a single configuration value by dotted key (or short alias). Returns the raw value as-is -- the API key is NOT redacted here.

**Usage:** `memgo config get <key>`

**Valid keys:** `api_key`/`platform.api_key`, `base_url`/`platform.base_url`, `user_email`/`platform.user_email`, `user_id`/`defaults.user_id`, `agent_id`/`defaults.agent_id`, `app_id`/`defaults.app_id`, `run_id`/`defaults.run_id`.

Unknown keys print an error.

**Examples:**
```bash
memgo config get platform.api_key   # 输出完整 key, 未脱敏
memgo config get user_id
```

#### `memgo config set <key> <value>`

Sets a configuration value and saves the config file.

**Usage:** `memgo config set <key> <value>`

**Type coercion:** boolean fields accept `true`/`1`/`yes` (case-insensitive) as true, anything else as false; integer fields are parsed; strings are stored as-is.

**Examples:**
```bash
memgo config set user_id alice
memgo config set platform.base_url https://memgo.wxget.com
```

There is no `config clear` subcommand in the Go CLI (present in the upstream Python/Node CLIs); remove `~/.memgo/config.json` directly if you need to reset.

---

### `memgo entity`

Manage entities.

#### `memgo entity list <entity_type>`

List all entities of a given type.

**Usage:** `memgo entity list <entity_type> [flags]`

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `entity_type` | string | Yes | Entity type: `users`, `agents`, `apps`, `runs`. |

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-o, --output <fmt>` | string | `table` | Output format: `table`, `json`. |
| `--api-key`, `--base-url` | string | - | Connection overrides. |

On the OSS backend the supported types are `user` / `agent` / `run`; `apps` errors.

**Examples:**
```bash
memgo entity list users
memgo entity list agents -o json
```

#### `memgo entity delete`

Delete an entity and ALL its memories (cascade).

**Usage:** `memgo entity delete [flags]`

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-u, --user-id <id>` | string | - | User ID of the entity to delete. |
| `--agent-id <id>` | string | - | Agent ID of the entity to delete. |
| `--app-id <id>` | string | - | App ID of the entity to delete. |
| `--run-id <id>` | string | - | Run ID of the entity to delete. |
| `--dry-run` | boolean | false | Show what would be deleted without deleting. |
| `--force` | boolean | false | Skip confirmation prompt. |
| `-o, --output <fmt>` | string | `text` | Output format: `text`, `json`, `quiet`. |
| `--api-key`, `--base-url` | string | - | Connection overrides. |

At least one entity ID is required. On OSS, `--app-id` is rejected.

**Examples:**
```bash
memgo entity delete --user-id alice --force
memgo entity delete --user-id alice --dry-run
memgo entity delete --agent-id bot1 --force
```

---

### `memgo event`

Inspect background processing events (platform only).

#### `memgo event list`

List recent background processing events.

**Usage:** `memgo event list [flags]`

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-o, --output <fmt>` | string | `table` | Output format: `table`, `json`. |
| `--api-key`, `--base-url` | string | - | Connection overrides. |

#### `memgo event status <event_id>`

Get the status and results of a specific background event.

**Usage:** `memgo event status <event_id> [flags]`

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-o, --output <fmt>` | string | `text` | Output format: `text`, `json`. |
| `--api-key`, `--base-url` | string | - | Connection overrides. |

**OSS behavior:** events are not supported on the OSS backend. Both subcommands error with `events are not supported on the OSS backend` -- OSS writes are synchronous, so there is nothing to poll.

**Examples:**
```bash
memgo event list
memgo event status evt-abc-123 -o json
```

---

### `memgo init`

Interactive setup wizard for memgo CLI.

**Usage:** `memgo init [flags]`

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--api-key <key>` | string | - | API key (skip interactive prompt). |
| `-u, --user-id <id>` | string | - | Default user ID (skip interactive prompt). |
| `--email <addr>` | string | - | Login via email verification code. |
| `--code <code>` | string | - | Verification code (use with `--email`). |
| `--agent` | boolean | false | Bootstrap an unattended Agent Mode account. |
| `--agent-caller <name>` | string | - | Self-declared agent identity. |
| `--source <channel>` | string | - | Channel attribution for signup. |
| `--force` | boolean | false | Overwrite existing config without confirmation. |

**Behavior (Go CLI):**
- The `--email`/`--code`/`--agent`/`--agent-caller` flows are declared but **not migrated** -- they print an error and point at `--api-key`. Those flows exist in the Python/Node ports.
- With `--api-key` (and optionally `--user-id`) the wizard runs non-interactively and saves `~/.memgo/config.json` (dir `0700`, file `0600`).
- Without `--api-key` and stdin is a TTY, it prompts for the key and default user ID.
- If config already exists with an API key and no `--force`, it errors: `Config already exists. Use --force to overwrite.`

**Examples:**
```bash
memgo init
memgo init --api-key mgsk-xxx --user-id alice
memgo init --api-key mgsk-xxx --user-id alice --force
```

---

### `memgo identify`

Tag your active Agent Mode key with the AI agent that's using it.

**Usage:** `memgo identify <name>`

**Arguments:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `name` | string | Yes | The AI agent identity (e.g. `claude-code`, `cursor`, `codex`). |

**Behavior:** works only on unclaimed agent-mode keys (`platform.agent_mode=true` in config). Without one it errors. In the Go CLI it only records the caller locally in config.

**Examples:**
```bash
memgo identify claude-code
memgo identify cursor
```

---

### `memgo status`

Check connectivity and authentication.

**Usage:** `memgo status [flags]`

**Options:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-o, --output <fmt>` | string | `text` | Output format: `text`, `json`. |
| `--api-key`, `--base-url` | string | - | Connection overrides. |

**Behavior:** pings the backend -- on OSS, `GET /auth/setup-status`; on platform, `GET /v1/ping/`. Text mode prints `Connected to <backend> backend at <base_url>` or `Connection failed: <error>`. JSON mode wraps the status object (`connected`, `backend`, `base_url`) in the agent envelope under `data`.

**Examples:**
```bash
memgo status
memgo status -o json
```

---

### `memgo version`

Print the CLI version and exit.

```bash
memgo version
# ◆ memgo CLI v0.2.12 (Go)
```

---

### `memgo whoami`

Print your AGENTRUSH identifier (the `default_user_id` from config). Errors if not set.

```bash
memgo whoami
# ◆ Your AGENTRUSH identifier:  user_<slug>
```

---

### `memgo help`

Show help. Text mode prints the command list; `--json` (or `--json` global) emits a machine-readable command map for LLM agents.

**Usage:** `memgo help [flags]`

**Options:**

| Flag | Type | Description |
|------|------|-------------|
| `--json` | boolean | Output machine-readable JSON. |

---

## Agent Mode Envelope Format

When `--json` or `--agent` is passed, every command wraps its output in a consistent JSON envelope on stdout:

```json
{
  "status": "success",
  "command": "search",
  "scope": { "user_id": "alice" },
  "count": 2,
  "memgo_notice": "optional notice text",
  "data": { ... }
}
```

**Fields:**
- `status`: `"success"` or `"error"`.
- `command`: The command name (e.g. `"search"`, `"add"`, `"config.show"`, `"entity.list"`).
- `scope`: Active entity scope, present only if non-empty.
- `count`: Number of results, where applicable.
- `memgo_notice`: Present only when the server returned a notice.
- `data`: Command-specific response data, or `null` on error.

The Go CLI envelope omits the `duration_ms` field emitted by the upstream Python/Node CLIs. `config show -o json` also uses this envelope with `command: "config.show"`.

**Error envelope:**
```json
{
  "status": "error",
  "command": "search",
  "error": "Authentication failed. Your API key may be invalid or expired.",
  "data": null
}
```

---

## Entity ID Resolution

**Rule:** if **any** explicit entity ID is provided via CLI flags (`--user-id`, `--agent-id`, `--app-id`, `--run-id`), the CLI uses only the explicitly provided IDs. It does NOT mix in defaults from config for the other entity types.

If **no** explicit IDs are given, all configured defaults from config file and env vars apply.

```
if any(user_id, agent_id, app_id, run_id) were passed as flags:
    use only the explicitly provided IDs (others = null)
else:
    use all configured defaults
```

This applies to `add`, `search`, `list`, `delete`, and `import`.

---

## Filter Building

For `search` and `list`, entity IDs and additional filters are composed into the API filter structure:

1. If the user provides a pre-built filter via `--filter` containing `AND` or `OR` keys, it is passed through to the API as-is (merged with the scope IDs on OSS).
2. Otherwise the CLI builds the filter from the scope IDs (`user_id`, `agent_id`, `run_id`) and, on the platform backend, `category`/`after`/`before` conditions.
3. On the OSS backend, `list`'s pagination/category/date filters are rejected -- only scope IDs and `top_k` are sent.

---

## Output Mode Support Matrix

| Command | `text` | `json` | `table` | `quiet` | Default |
|---------|--------|--------|---------|---------|---------|
| `add` | Y | Y | - | Y | `text` |
| `search` | Y | Y | Y | - | `text` |
| `get` | Y | Y | - | - | `text` |
| `list` | Y | Y | Y | - | `table` |
| `update` | Y | Y | - | Y | `text` |
| `delete` | Y | Y | - | Y | `text` |
| `import` | Y | Y | - | - | `text` |
| `config show` | Y (raw JSON) | Y (envelope) | - | - | `text` |
| `config get` | raw | - | - | - | raw |
| `config set` | msg | - | - | - | msg |
| `entity list` | - | Y | Y | - | `table` |
| `entity delete` | Y | Y | - | Y | `text` |
| `event list` | Y | Y | - | - | `table` |
| `event status` | Y | Y | - | - | `text` |
| `status` | Y | Y | - | - | `text` |
| `version` / `whoami` / `identify` / `help` | Y | - | - | - | `text` |

**Output shapes:**
- `text` (memories): `Found N memories:` followed by a numbered list; each item shows the memory text, an 8-char ID prefix, and `Created` date (search also shows `Score`).
- `table`: aligned columns `ID` / `Memory` / `Created` (search adds a `Score` column).
- `quiet`: prints only memory IDs, one per line.

All commands additionally support agent mode (`--json`/`--agent`), which overrides the output format with the JSON envelope.
