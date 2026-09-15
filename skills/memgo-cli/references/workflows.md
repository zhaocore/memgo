# MemGo CLI Workflows

Practical recipes for using the memgo CLI in scripts, pipelines, and agent loops. These work against both backends unless a section notes an OSS (`MEMGO_BASE_URL` → self-hosted server) difference.

---

## Piping Content via Stdin

The CLI reads one line from stdin when no text argument is provided and input is piped (not a TTY). This works with `add`, `search`, and `update`.

**Stdin detection:** stdin is piped when `os.Stdin.Stat()` reports it is not a character device.

### Add from pipe

```bash
echo "I prefer dark mode" | memgo add --user-id alice
```

### Pipe multi-line content

```bash
cat <<EOF | memgo add --user-id alice
The user prefers dark mode in all applications.
They also like monospace fonts for code editing.
EOF
```

### Pipe from another command

```bash
git log --oneline -5 | memgo add --user-id ci-bot --metadata '{"source":"git"}'
```

### Search from pipe

```bash
echo "preferences" | memgo search --user-id alice
```

### Update from pipe

```bash
echo "Updated: prefers dark mode AND high contrast" | memgo update abc-123-def-456
```

---

## File Import

Use `memgo import` to bulk-load memories from a JSON file.

### Basic import

```bash
memgo import memories.json --user-id alice
```

### File format

The file must be a JSON array. Each item carries its text in a `memory` field (falling back to `text`); items without text are skipped. `--user-id` / `--agent-id` apply to every item:

```json
[
  { "memory": "Prefers dark mode" },
  { "text": "Allergic to nuts", "metadata": { "source": "intake-form" } }
]
```

### Import with JSON output

```bash
memgo import data.json --user-id alice -o json
```

---

## Agent Mode for LLM Consumption

Use `--json` (or its alias `--agent`) to get structured JSON output suitable for LLM tool calling or agent frameworks. Spinners and progress never pollute stdout.

### Search with agent mode

```bash
memgo search "preferences" --user-id alice --json
```

Output (stdout):

```json
{
  "status": "success",
  "command": "search",
  "scope": { "user_id": "alice" },
  "count": 2,
  "data": [
    { "id": "mem-abc", "memory": "User prefers dark mode", "score": 0.95, "created_at": "2025-01-15T10:00:00Z" },
    { "id": "mem-def", "memory": "User likes monospace fonts", "score": 0.82, "created_at": "2025-01-15T10:01:00Z" }
  ]
}
```

Note the Go CLI envelope omits `duration_ms` and may include a `memgo_notice` field when the server returns a notice.

### Add with agent mode

```bash
memgo add "Uses Python 3.12" --user-id alice --json
```

### Error handling in agent mode

Errors also return valid JSON with `"status": "error"`:

```bash
memgo search "test" --user-id alice --api-key invalid --json
```

Output:

```json
{
  "status": "error",
  "command": "search",
  "error": "Authentication failed. Your API key may be invalid or expired.",
  "data": null
}
```

---

## JSON Output + jq

Use `--output json` (or `-o json`) for raw JSON output, then pipe to `jq`. On memory commands, `-o json` prints the result array directly (unwrapped).

### Extract just memory text

```bash
memgo list --user-id alice --output json | jq '.[] | .memory'
```

### Get memory IDs

```bash
memgo list --user-id alice -o json | jq -r '.[].id'
```

### Count memories

```bash
memgo list --user-id alice -o json | jq 'length'
```

### Extract search scores

```bash
memgo search "tools" --user-id alice -o json | jq '.[] | {memory, score}'
```

---

## Bulk Operations

### Delete multiple memories by ID

```bash
# Get IDs, then delete each one
memgo list --user-id alice -o json | jq -r '.[].id' | while read id; do
  memgo delete "$id" --force
done
```

### Bulk add from a text file (one memory per line)

```bash
while IFS= read -r line; do
  memgo add "$line" --user-id alice
done < memories.txt
```

### Copy memories between users

```bash
memgo list --user-id alice -o json | jq -r '.[].memory' | while IFS= read -r mem; do
  memgo add "$mem" --user-id bob
done
```

### Export all memories to a file

```bash
memgo list --user-id alice -o json > alice_memories.json
```

---

## CI/CD Patterns

### Store build context as a memory

```bash
memgo add "Build #${BUILD_NUMBER} deployed ${APP_VERSION} to ${ENVIRONMENT} at $(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  --agent-id "ci-bot" \
  --metadata "{\"build_number\":\"${BUILD_NUMBER}\",\"version\":\"${APP_VERSION}\",\"env\":\"${ENVIRONMENT}\"}"
```

### Retrieve deployment history

```bash
memgo search "deployment to production" --agent-id ci-bot -o json -k 10
```

### Check CLI connectivity in CI

```bash
if memgo status -o json | jq -e '.data.connected' > /dev/null 2>&1; then
  echo "memgo is connected"
else
  echo "memgo connection failed" >&2
  exit 1
fi
```

(`memgo status -o json` wraps the status object in the agent envelope, so the connectivity flag lives at `.data.connected`.)

### Non-interactive init in CI

```bash
memgo init --api-key "$MEMGO_API_KEY" --user-id ci-bot --force
```

Or simply use the environment variable (no init needed):

```bash
export MEMGO_API_KEY="$MEMGO_API_KEY"
export MEMGO_BASE_URL="https://memgo.wxget.com"
memgo add "CI run started" --user-id ci-bot
```

---

## Stdin Detection Details

The CLI reads from stdin only when ALL of these conditions are met:

1. No text argument was provided on the command line.
2. For `add`: no `--messages` and no `--file` flag.
3. For `update`: no text argument.
4. stdin is piped (not a TTY).

**This means:**

- `memgo add --user-id alice` in an interactive terminal will NOT hang waiting for input. It prints a usage error.
- `echo "text" | memgo add --user-id alice` reads "text" from stdin.
- `memgo add "explicit text" --user-id alice` uses the explicit text, even if stdin is piped.

The reader consumes one line and trims surrounding whitespace.

---

## Common Shell Patterns

### Error handling with exit codes

```bash
set -e  # Exit on error

# This will exit the script if the API key is invalid
memgo status > /dev/null 2>&1

# Add with error check
if memgo add "test memory" --user-id alice 2>/dev/null; then
  echo "Memory added successfully"
else
  echo "Failed to add memory" >&2
  exit 1
fi
```

### Capture memory ID from add

```bash
# Use agent mode to get structured output
result=$(memgo add "new fact" --user-id alice --json 2>/dev/null)
memory_id=$(echo "$result" | jq -r '.data[0].id // empty')
if [ -n "$memory_id" ]; then
  echo "Created memory: $memory_id"
fi
```

### Conditional memory addition

```bash
# Only add if search returns no results
count=$(memgo search "dark mode" --user-id alice --json 2>/dev/null | jq '.count // 0')
if [ "$count" -eq 0 ]; then
  memgo add "User prefers dark mode" --user-id alice
fi
```

### Quiet mode for scripts

```bash
# Suppress all output except errors (prints only memory IDs)
memgo add "background note" --user-id alice --output quiet 2>/dev/null
memgo delete --all --user-id temp-user --force --output quiet 2>/dev/null
```

### Using environment variables for scope

```bash
export MEMGO_USER_ID="alice"
export MEMGO_API_KEY="mgsk-xxx"
export MEMGO_BASE_URL="https://memgo.wxget.com"

# All commands now default to user alice, no --user-id needed
memgo add "prefers dark mode"
memgo search "preferences"
memgo list
```

### Timeout handling

The Go CLI uses a 600s HTTP client timeout on the OSS backend. For long-running scripts, handle failures:

```bash
if ! memgo search "query" --user-id alice -o json 2>/dev/null; then
  echo "Request failed or timed out" >&2
fi
```

---

## OSS Backend Notes

When `MEMGO_BASE_URL` points at a self-hosted server (e.g. `https://memgo.wxget.com`), the CLI uses the synchronous OSS contract. Differences to remember in scripts:

- **Writes are synchronous.** `memgo add` returns the extracted memory immediately. There is no `PENDING` event and no event-polling loop -- delete the "wait 2-3 seconds then poll `memgo event status`" pattern from the platform workflow.
- **`search` needs a scope.** The OSS server rejects a query with no `user_id` / `agent_id` / `run_id`. Always pass at least one scope flag or set a default via `MEMGO_USER_ID` / `config set user_id`.
- **No app_id, no events, no categories.** Avoid `--app-id`, `--custom-categories`, `--keyword`, and the `event` subcommand against OSS; they error out. `--project` project-wide delete is not available -- use the server admin `POST /reset`.
- **`list` pagination/date filters unsupported.** `--page > 1`, `--category`, `--after`, `--before` error on OSS; use `--page-size` (maps to the server `top_k`) to cap result count.

---

## Multi-User Agent Pattern

For AI agents managing memories across multiple users:

```bash
#!/bin/bash
# agent_memory.sh -- manage memories for the current conversation

USER_ID="$1"
ACTION="$2"
shift 2

case "$ACTION" in
  recall)
    memgo search "$*" --user-id "$USER_ID" --json 2>/dev/null
    ;;
  remember)
    memgo add "$*" --user-id "$USER_ID" --json 2>/dev/null
    ;;
  forget)
    memgo delete --all --user-id "$USER_ID" --force --json 2>/dev/null
    ;;
  history)
    memgo list --user-id "$USER_ID" --json 2>/dev/null
    ;;
  *)
    echo '{"status":"error","error":"Unknown action: '"$ACTION"'"}' >&2
    exit 1
    ;;
esac
```

Usage:

```bash
./agent_memory.sh alice recall "dietary preferences"
./agent_memory.sh alice remember "allergic to shellfish"
./agent_memory.sh alice history
```
