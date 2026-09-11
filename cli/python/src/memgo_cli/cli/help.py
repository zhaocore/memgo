"""机器可读命令帮助。"""

from __future__ import annotations

from memgo_cli import __version__
from memgo_cli.backend.json import JsonObject


def _build_help_json() -> JsonObject:
    """构造描述全部命令的机器可读帮助。"""
    commands: JsonObject = {
        "add": {
            "description": "Add a memory from text, messages, file, or stdin.",
            "usage": "memgo add <text> [OPTIONS]",
            "arguments": {
                "text": {"description": "Text content to add as a memory.", "required": False}
            },
            "options": {
                "--user-id, -u": "Scope to user.",
                "--agent-id": "Scope to agent.",
                "--app-id": "Scope to app.",
                "--run-id": "Scope to run.",
                "--messages": "Conversation messages as JSON.",
                "--file, -f": "Read messages from JSON file.",
                "--metadata, -m": "Custom metadata as JSON.",
                "--immutable": "Prevent future updates.",
                "--no-infer": "Skip inference, store raw.",
                "--expires": "Expiration date (YYYY-MM-DD).",
                "--categories": "Not supported on add, use --custom-categories instead.",
                "--custom-instructions": "Custom instructions for fact extraction.",
                "--agent-custom-instructions": "Extraction instructions for agent-scoped memories, overriding the project setting.",
                "--custom-categories": "Custom categories as a JSON array of {name: description} objects.",
                "--structured-data-schema": "Schema for structured data extraction, as JSON.",
                "--timestamp": "Unix timestamp for the memory.",
                "--graph": "Enable graph memory extraction.",
                "--no-graph": "Disable graph memory extraction.",
                "--output, -o": "Output format: text, json, quiet.",
            },
        },
        "search": {
            "description": "Query your memory store — semantic, keyword, or hybrid retrieval.",
            "usage": "memgo search <query> [OPTIONS]",
            "arguments": {"query": {"description": "Search query.", "required": False}},
            "options": {
                "--user-id, -u": "Filter by user.",
                "--agent-id": "Filter by agent.",
                "--top-k, -k, --limit": "Number of results (default: 10).",
                "--threshold": "Minimum similarity score (default: 0.3).",
                "--rerank": "Enable reranking (Platform only).",
                "--keyword": "Use keyword search instead of semantic.",
                "--filter": (
                    'Advanced filter as JSON: {"AND": [...]} or {"OR": [...]}, '
                    'e.g. {"AND": [{"categories": {"in": ["work"]}}]}.'
                ),
                "--fields": "Specific fields to return (comma-separated).",
                "--show-expired": "Include expired memories.",
                "--reference-date": "Reference date for relative queries (YYYY-MM-DD or unix timestamp).",
                "--latest-only": "Only return the latest version of each memory.",
                "--graph": "Enable graph in search.",
                "--no-graph": "Disable graph in search.",
                "--output, -o": "Output format: text, json, table.",
            },
        },
        "get": {
            "description": "Get a specific memory by ID.",
            "usage": "memgo get <memory_id> [OPTIONS]",
            "arguments": {"memory_id": {"description": "Memory ID to retrieve.", "required": True}},
            "options": {"--output, -o": "Output format: text, json."},
        },
        "list": {
            "description": "List memories with optional filters.",
            "usage": "memgo list [OPTIONS]",
            "arguments": {},
            "options": {
                "--user-id, -u": "Filter by user.",
                "--agent-id": "Filter by agent.",
                "--page": "Page number (default: 1).",
                "--page-size": "Results per page (default: 100).",
                "--category": "Filter by category.",
                "--after": "Created after (YYYY-MM-DD).",
                "--before": "Created before (YYYY-MM-DD).",
                "--show-expired": "Include expired memories.",
                "--latest-only": "Only return the latest version of each memory.",
                "--graph": "Enable graph in listing.",
                "--no-graph": "Disable graph in listing.",
                "--output, -o": "Output format: text, json, table.",
            },
        },
        "update": {
            "description": "Update a memory's text or metadata.",
            "usage": "memgo update <memory_id> [text] [OPTIONS]",
            "arguments": {
                "memory_id": {"description": "Memory ID to update.", "required": True},
                "text": {"description": "New memory text.", "required": False},
            },
            "options": {
                "--metadata, -m": "Update metadata (JSON).",
                "--expires": "Expiration date (YYYY-MM-DD).",
                "--timestamp": "Unix timestamp for the memory.",
                "--output, -o": "Output format: text, json, quiet.",
            },
        },
        "delete": {
            "description": "Delete a memory, all memories, or an entity.",
            "usage": "memgo delete [memory_id] [OPTIONS]",
            "arguments": {
                "memory_id": {
                    "description": "Memory ID to delete (omit when using --all or --entity).",
                    "required": False,
                }
            },
            "options": {
                "--all": "Delete all memories matching scope filters.",
                "--entity": "Delete the entity itself and all its memories (cascade).",
                "--project": "With --all: delete ALL memories project-wide.",
                "--delete-linked": "Also delete memories linked to this memory.",
                "--dry-run": "Show what would be deleted without deleting.",
                "--force": "Skip confirmation.",
                "--user-id, -u": "Scope to user.",
                "--agent-id": "Scope to agent.",
                "--app-id": "Scope to app.",
                "--run-id": "Scope to run.",
                "--output, -o": "Output format: text, json, quiet.",
            },
        },
        "import": {
            "description": "Import memories from a JSON file.",
            "usage": "memgo import <file_path> [OPTIONS]",
            "arguments": {"file_path": {"description": "JSON file to import.", "required": True}},
            "options": {
                "--user-id, -u": "Override user ID.",
                "--agent-id": "Override agent ID.",
                "--output, -o": "Output format: text, json.",
            },
        },
        "config show": {
            "description": "Display current configuration (secrets redacted).",
            "usage": "memgo config show",
            "options": {"--output, -o": "Output format: text, json."},
        },
        "config get": {
            "description": "Get a configuration value.",
            "usage": "memgo config get <key>",
            "arguments": {
                "key": {"description": "Config key (e.g. platform.api_key).", "required": True}
            },
        },
        "config set": {
            "description": "Set a configuration value.",
            "usage": "memgo config set <key> <value>",
            "arguments": {
                "key": {"description": "Config key (e.g. platform.api_key).", "required": True},
                "value": {"description": "Value to set.", "required": True},
            },
        },
        "event": {
            "description": "Inspect background processing events.",
            "subcommands": {
                "list": {
                    "description": "List recent background processing events.",
                    "usage": "memgo event list [OPTIONS]",
                    "options": {"--output, -o": "Output format: table, json."},
                },
                "status": {
                    "description": "Check the status of a specific background event.",
                    "usage": "memgo event status <event_id> [OPTIONS]",
                    "arguments": {
                        "event_id": {"description": "Event ID to inspect.", "required": True}
                    },
                    "options": {"--output, -o": "Output format: text, json."},
                },
            },
        },
        "entity": {
            "description": "Manage entities.",
            "subcommands": {
                "list": {
                    "description": "List all entities of a given type.",
                    "usage": "memgo entity list <entity_type> [OPTIONS]",
                    "arguments": {
                        "entity_type": {
                            "description": "Entity type: users, agents, apps, runs.",
                            "required": True,
                        }
                    },
                    "options": {"--output, -o": "Output format: table, json."},
                },
                "delete": {
                    "description": "Delete an entity and ALL its memories (cascade).",
                    "usage": "memgo entity delete [OPTIONS]",
                    "options": {
                        "--user-id, -u": "User ID.",
                        "--agent-id": "Agent ID.",
                        "--app-id": "App ID.",
                        "--run-id": "Run ID.",
                        "--force": "Skip confirmation.",
                        "--dry-run": "Show what would be deleted without deleting.",
                        "--output, -o": "Output format: text, json, quiet.",
                    },
                },
            },
        },
        "init": {
            "description": "Interactive setup wizard for memgo CLI.",
            "usage": "memgo init",
            "options": {
                "--api-key": "API key (skip prompt).",
                "--user-id, -u": "Default user ID (skip prompt).",
            },
        },
        "status": {
            "description": "Check connectivity and authentication.",
            "usage": "memgo status [OPTIONS]",
            "options": {"--output, -o": "Output format: text, json."},
        },
    }
    return {
        "name": "memgo",
        "version": __version__,
        "description": "The Memory Layer for AI Agents",
        "commands": commands,
        "global_options": {
            "--api-key": "Override API key (env: MEMGO_API_KEY).",
            "--base-url": "Override API base URL.",
            "--json / --agent": "Output as JSON for agent/programmatic use.",
            "--help": "Show help for a command.",
            "--version": "Show version and exit.",
        },
        "help": {
            "human": "memgo <command> --help    Get help for a command",
            "machine": "memgo help --json         Machine-readable help (for LLM agents)",
        },
    }
