#!/usr/bin/env bash
set -uo pipefail

if [ -n "${MEMGO_DEBUG:-}" ]; then
  mkdir -p "$HOME/.memgo" && exec 2>>"$HOME/.memgo/hooks.log"
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
. "$SCRIPT_DIR/_identity.sh" 2>/dev/null || true

INPUT=$(cat)
SOURCE=$(echo "$INPUT" | jq -r '.source // "startup"' 2>/dev/null || echo "startup")

if [ "$SOURCE" = "startup" ]; then
  python3 "$SCRIPT_DIR/session_stats.py" init 2>/dev/null || true
  rm -f /tmp/memgo_recent_reads_${USER:-default}_* 2>/dev/null || true
fi
PYTHONPATH="$SCRIPT_DIR" python3 "$SCRIPT_DIR/load_settings.py" init 2>/dev/null || true
rm -f "/tmp/memgo_msg_count_${USER:-default}" 2>/dev/null || true
MEMGO_SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // ""' 2>/dev/null || echo "")
if [ -z "$MEMGO_SESSION_ID" ]; then
  MEMGO_SESSION_ID="ses_$(date +%s)_$$"
fi
printf '%s' "$MEMGO_SESSION_ID" > "/tmp/memgo_session_id_${USER:-default}"
export MEMGO_SESSION_ID

if [ -z "${MEMGO_API_KEY:-}" ]; then
  _UID="${MEMGO_RESOLVED_USER_ID:-${USER:-default}}"
  _AID="${MEMGO_AGENT_ID:-unknown}"
  _BR="${MEMGO_BRANCH:-unknown}"
  cat <<BANNER
## MemGo — Setup Required

\`user=${_UID} | agent=${_AID} | branch=${_BR} | auth=none\`

IMPORTANT: In your FIRST response, display this exact status line as your opening line:

\`\`\`
MemGo — Setup Required | user=${_UID} | agent=${_AID} | branch=${_BR} | auth=none
\`\`\`

MEMGO_API_KEY is not set. To configure:
- CLI: Add \`export MEMGO_API_KEY=...\` to your shell profile (~/.zshrc or ~/.bashrc)
- Point at your self-hosted server with \`export MEMGO_BASE_URL=...\` (default https://memgo.wxget.com)
BANNER
  exit 0
fi

MEMGO_COUNT="?"
if command -v python3 >/dev/null 2>&1; then
  MEMGO_COUNT=$(MEMGO_RESOLVED_USER_ID="${MEMGO_RESOLVED_USER_ID:-${USER:-default}}" \
    MEMGO_AGENT_ID="${MEMGO_AGENT_ID:-unknown}" \
    PYTHONPATH="$SCRIPT_DIR" python3 -c "
import os, sys
sys.path.insert(0, os.environ.get('PYTHONPATH', '.'))
from _api import list_memories

user = os.environ.get('MEMGO_RESOLVED_USER_ID', 'default')
agent = os.environ.get('MEMGO_AGENT_ID', 'unknown')
try:
    results = list_memories(user_id=user, agent_id=agent)
    print(len(results))
except Exception:
    print('?')
" 2>/dev/null || echo "?")
fi

_UID="${MEMGO_RESOLVED_USER_ID:-${USER:-default}}"
_ANN="${_MEMGO_IDENTITY_ANNOTATION:-}"
_AID="${MEMGO_AGENT_ID:-unknown}"
_BR="${MEMGO_BRANCH:-unknown}"

cat <<BANNER
## MemGo Active

\`user=${_UID}${_ANN} | agent=${_AID} | branch=${_BR} | memories=${MEMGO_COUNT}\`

IMPORTANT: In your FIRST response, display this exact status line as your opening line:

\`\`\`
MemGo Active | user=${_UID}${_ANN} | agent=${_AID} | branch=${_BR} | memories=${MEMGO_COUNT}
\`\`\`

Memories are scoped by user (\`${_UID}\`) and agent (\`${_AID}\`, the repo identity).
Use the memgo MCP tools \`add_memory\` / \`search_memories\` / \`get_memories\` to store and retrieve them.

After completing any task, decision, or meaningful exchange, proactively store learnings via \`add_memory\`. Do NOT wait until the session ends — store memories incrementally as work progresses. Focus on: decisions made, bugs fixed, patterns discovered, user preferences, or task outcomes. Aim for 1–3 memories per substantial interaction.

BANNER

MEMGO_CWD_RESOLVED=$(echo "$INPUT" | jq -r '.cwd // "."' 2>/dev/null || echo ".")
if [ "$SOURCE" = "startup" ]; then
  if [ "${MEMGO_AUTO_SEARCH:-true}" = "true" ]; then
    if [ "$MEMGO_COUNT" = "0" ]; then
      echo "New project with 0 memories. Start storing decisions and learnings via add_memory as you work."
    else
      echo "Search MemGo for recent decisions and task learnings before responding."

      # Inject compact recent activity timeline (non-blocking, 5s timeout)
      # Use perl alarm as portable timeout (macOS lacks GNU timeout)
      _TIMELINE=$(MEMGO_CWD="$MEMGO_CWD_RESOLVED" perl -e 'alarm 5; exec @ARGV' python3 "$SCRIPT_DIR/session_timeline.py" 2>/dev/null || echo "")
      if [ -n "$_TIMELINE" ]; then
        echo ""
        echo "$_TIMELINE"
      fi
    fi
  fi

elif [ "$SOURCE" = "resume" ]; then
  echo "Session resumed. Search MemGo for session_state and decision memories to pick up where you left off."

elif [ "$SOURCE" = "compact" ]; then
  echo "Context compacted. Search MemGo for session_state and decision memories to recover context."
  if [ "${MEMGO_AUTO_SAVE:-true}" != "false" ]; then
    printf '%s' "$INPUT" | python3 "$SCRIPT_DIR/capture_compact_summary.py" 2>/dev/null &
  fi
fi

python3 "$SCRIPT_DIR/telemetry.py" session_start --source="$SOURCE" --memory_count="${MEMGO_COUNT:-0}" 2>/dev/null &

exit 0
