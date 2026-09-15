#!/usr/bin/env bash
# PreToolUse hook for MemGo MCP tools.
# Injects identity (user_id, agent_id) and metadata defaults when the agent
# omits them. Uses the hookSpecificOutput.updatedInput contract to modify
# tool call parameters before execution.
#
# Handles (OSS 面, 实体范围 user_id/agent_id/run_id):
#   add_memory          — 顶层 user_id, agent_id, run_id, metadata 默认值
#   search_memories     — user_id/agent_id 注入扁平 filters
#   get_memories        — user_id/agent_id 注入扁平 filters
#   delete_all_memories — 顶层 user_id, agent_id
#
# Hook contract:
#   exit 0 = allow. If stdout contains {"hookSpecificOutput": {"updatedInput": ...}},
#            the updatedInput replaces the tool's input parameters.
#   exit 2 = block (stderr shown as rejection reason).

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "$SCRIPT_DIR/_identity.sh" 2>/dev/null || true

INPUT=$(cat)

TOOL_NAME=$(echo "$INPUT" | jq -r '.tool_name // ""' 2>/dev/null)

# Determine which handler to use based on tool name
HANDLER=""
case "$TOOL_NAME" in
  mcp__memgo__add_memory)
    HANDLER="add_memory" ;;
  mcp__memgo__search_memories)
    HANDLER="search_memories" ;;
  mcp__memgo__get_memories)
    HANDLER="get_memories" ;;
  mcp__memgo__delete_all_memories)
    HANDLER="delete_all" ;;
  *) exit 0 ;;
esac

TOOL_INPUT=$(echo "$INPUT" | jq -r '.tool_input // "{}"' 2>/dev/null)

_PATCH_OUT="/tmp/memgo_enforce_$$"
trap 'rm -f "$_PATCH_OUT"' EXIT
_MEMGO_TOOL_INPUT="$TOOL_INPUT" \
_MEMGO_USER_ID="${MEMGO_RESOLVED_USER_ID:-}" \
_MEMGO_AGENT_ID="${MEMGO_AGENT_ID:-}" \
_MEMGO_HANDLER="$HANDLER" \
python3 <<'PYEOF' > "$_PATCH_OUT" 2>/dev/null || true
import json, os, sys

raw = os.environ.get("_MEMGO_TOOL_INPUT", "{}")
try:
    inp = json.loads(raw)
except Exception:
    sys.exit(0)

handler = os.environ.get("_MEMGO_HANDLER", "")
resolved_uid = os.environ.get("_MEMGO_USER_ID", "")
resolved_aid = os.environ.get("_MEMGO_AGENT_ID", "")
changed = False


def inject_top_level_identity(inp, uid, aid):
    """Inject user_id/agent_id as top-level params (add_memory, delete_all)."""
    changed = False
    if uid and not inp.get("user_id"):
        inp["user_id"] = uid
        changed = True
    if aid and not inp.get("agent_id"):
        inp["agent_id"] = aid
        changed = True
    return changed


def inject_filter_identity(inp, uid, aid):
    """Inject user_id/agent_id into the flat filters dict (OSS 搜索/列表)."""
    changed = False
    if not uid and not aid:
        return False

    filters = inp.get("filters")
    if filters is None:
        filters = {}
        inp["filters"] = filters
    if not isinstance(filters, dict):
        return False

    if uid and not filters.get("user_id"):
        filters["user_id"] = uid
        changed = True
    if aid and not filters.get("agent_id"):
        filters["agent_id"] = aid
        changed = True
    return changed


if handler == "add_memory":
    changed = inject_top_level_identity(inp, resolved_uid, resolved_aid)

    meta = inp.get("metadata") or {}

    if "confidence" not in meta:
        meta["confidence"] = 0.7
        changed = True
    if "source" not in meta:
        meta["source"] = "auto_capture"
        changed = True
    if "type" not in meta:
        meta["type"] = "task_learning"
        changed = True

    if meta.get("confidence", 0) >= 1.0 and "infer" not in inp:
        inp["infer"] = False
        changed = True

    # 会话范围用 run_id 表达(OSS 支持 run_id 实体维度, 可随 filters 检索)。
    if not inp.get("run_id"):
        sid = os.environ.get("MEMGO_SESSION_ID", "")
        if not sid:
            session_file = "/tmp/memgo_session_id_" + os.environ.get("USER", "default")
            if os.path.isfile(session_file):
                try:
                    with open(session_file) as f:
                        sid = f.read().strip()
                except OSError:
                    pass
        if sid:
            inp["run_id"] = sid
            changed = True

    if changed:
        inp["metadata"] = meta

elif handler in ("search_memories", "get_memories"):
    changed = inject_filter_identity(inp, resolved_uid, resolved_aid)

elif handler == "delete_all":
    changed = inject_top_level_identity(inp, resolved_uid, resolved_aid)

if changed:
    print(json.dumps(inp))
PYEOF
PATCHED=$(cat "$_PATCH_OUT" 2>/dev/null)
rm -f "$_PATCH_OUT"

if [ -n "$PATCHED" ] && echo "$PATCHED" | jq empty 2>/dev/null; then
  jq -n --argjson updated "$PATCHED" '{
    "hookSpecificOutput": {
      "hookEventName": "PreToolUse",
      "permissionDecision": "allow",
      "updatedInput": $updated
    }
  }' 2>/dev/null || true
fi

# Track session stats here because PostToolUse hooks don't fire for plugin MCP tools.
case "$HANDLER" in
  add_memory)
    _CAT=$(echo "$TOOL_INPUT" | jq -r '.metadata.type // .metadata.category // ""' 2>/dev/null || echo "")
    python3 "$SCRIPT_DIR/session_stats.py" add "$_CAT" 2>/dev/null &
    ;;
  search_memories|get_memories)
    python3 "$SCRIPT_DIR/session_stats.py" search 2>/dev/null &
    ;;
esac

exit 0
