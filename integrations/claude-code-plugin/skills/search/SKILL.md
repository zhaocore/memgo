---
name: search
description: Search memories from earlier Claude Code sessions in this repository. Use it when earlier work may already explain the code, error, decision, or command you need, so you can avoid repeating file reads, searches, or experiments.
argument-hint: "[question] [--top-k number] [--scope repo|dir|mine] [--run-id session-id]"
disable-model-invocation: true
---

# Search memories

Call `search_memories` with the user's question. Treat `--top-k`, `--scope`,
and `--run-id` as tool arguments instead of including them in the query.

Omit `top_k` to use MemGo's configured default. Omit `scope` to use the
configured default, normally `repo`: this repository's shared memory, which
everyone who works in it contributes to, plus your own preferences. `dir`
behaves the same as `repo` because the OSS server has no directory filtering.

Pass `scope` when the question needs something else: `mine` for your own
preferences alone. Pass `run_id` with a Claude Code session ID to look at what
one earlier session recorded, for example to pick up where a compacted or
closed session left off. Return the tool's result directly.
