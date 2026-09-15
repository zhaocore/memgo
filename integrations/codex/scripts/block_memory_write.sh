#!/usr/bin/env bash
# Hook: PreToolUse (matcher: Write|Edit|MultiEdit)
#
# Blocks writes to MEMORY.md / auto-memory files, redirecting the agent
# to use the MemGo MCP add_memory tool instead.
#
# Input:  JSON on stdin with tool_name, tool_input
# Output: stderr message (exit 2 = block)
#
# Exit codes:
#   0 = allow the tool call
#   2 = block the tool call (stderr is shown to the agent as feedback)

set -euo pipefail

if [ -n "${MEMGO_DEBUG:-}" ]; then
  mkdir -p "$HOME/.memgo" && exec 2>>"$HOME/.memgo/hooks.log"
fi

INPUT=$(cat)

FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // .tool_input.path // ""' 2>/dev/null || echo "")

if [ -z "$FILE_PATH" ]; then
  exit 0
fi

case "$FILE_PATH" in
  */.claude/*/MEMORY.md|*/.claude/memory/*|*/.codex/memory/*)
    echo "BLOCKED: Do not write to $FILE_PATH. Use the MemGo MCP \`add_memory\` tool instead to persist memories. This project uses MemGo for all memory storage." >&2
    exit 2
    ;;
  *)
    exit 0
    ;;
esac
