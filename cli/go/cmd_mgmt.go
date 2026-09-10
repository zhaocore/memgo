package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// importFile 对齐 cmd_import: JSON 文件数组逐条 add。
func importFile(backend Backend, path string, ids EntityIDs) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, errAPI(fmt.Sprintf("读取文件失败: %v", err))
	}
	var items []map[string]any
	if err := json.Unmarshal(raw, &items); err != nil {
		return 0, errAPI("import 文件必须是记忆 JSON 数组")
	}
	imported := 0
	for _, item := range items {
		text := strOf(item["memory"])
		if text == "" {
			text = strOf(item["text"])
		}
		if text == "" {
			continue
		}
		p := AddParams{Content: text, IDs: ids}
		if _, err := backend.Add(p); err != nil {
			return imported, err
		}
		imported++
	}
	return imported, nil
}

// newInitCmd init (非交互 --api-key/--user-id; 交互提示; email/agent 流程未迁移)。
func newInitCmd() *cobra.Command {
	var apiKey, userID, email, code, source, agentCaller string
	var force, agentSignal bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive setup wizard for memgo CLI.",
		RunE: func(cmd *cobra.Command, args []string) error {
			fireTelemetry(LoadConfig(), "cli.init", map[string]any{"command": "init"})
			if email != "" || agentSignal || agentCaller != "" {
				printError("--email/--code/--agent/--agent-caller flows are not migrated to the Go CLI yet.", "Use --api-key from https://app.memgo.ai")
				return errExit
			}
			cfg := LoadConfig()
			if cfg.Platform.APIKey != "" && !force {
				printError("Config already exists. Use --force to overwrite.", "")
				return errExit
			}
			if apiKey == "" {
				apiKey = promptLine("Enter your memgo API key (https://app.memgo.ai): ")
				if apiKey == "" {
					printError("API key is required.", "")
					return errExit
				}
			}
			if userID == "" {
				userID = promptLine("Default user ID (optional, Enter to skip): ")
			}
			cfg.Platform.APIKey = apiKey
			cfg.Platform.CreatedVia = "api_key"
			if userID != "" {
				cfg.Defaults.UserID = userID
			}
			if err := SaveConfig(cfg); err != nil {
				return backendErr(&APIError{Kind: "api", Msg: err.Error()})
			}
			printSuccess(fmt.Sprintf("Config saved to %s", ConfigFile()))
			return nil
		},
	}
	cmd.Flags().StringVar(&apiKey, "api-key", "", "API key (skip prompt).")
	cmd.Flags().StringVarP(&userID, "user-id", "u", "", "Default user ID (skip prompt).")
	cmd.Flags().StringVar(&email, "email", "", "Login via email verification code.")
	cmd.Flags().StringVar(&code, "code", "", "Verification code (use with --email).")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config without confirmation.")
	cmd.Flags().BoolVar(&agentSignal, "agent", false, "Bootstrap an unattended Agent Mode account (no email required).")
	cmd.Flags().StringVar(&source, "source", "", "Channel attribution for signup.")
	cmd.Flags().StringVar(&agentCaller, "agent-caller", "", "Self-declared agent identity.")
	return cmd
}

// promptLine TTY 行输入。
func promptLine(prompt string) string {
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) == 0 {
		return ""
	}
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

// newIdentifyCmd (agent-mode key 专属)。
func newIdentifyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "identify <name>",
		Short: "Tag your active Agent Mode key with the AI agent that's using it.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := LoadConfig()
			if cfg.Platform.APIKey == "" {
				printError("No API key configured. Run `memgo init --agent` first.", "")
				return errExit
			}
			if !cfg.Platform.AgentMode {
				printError("This command only works on unclaimed agent-mode keys.", "")
				return errExit
			}
			cfg.Platform.AgentCaller = args[0]
			if err := SaveConfig(cfg); err != nil {
				return backendErr(&APIError{Kind: "api", Msg: err.Error()})
			}
			printSuccess(fmt.Sprintf("Identified as %s.", args[0]))
			return nil
		},
	}
}

// newWhoamiCmd。
func newWhoamiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Print your AGENTRUSH identifier (default_user_id).",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := LoadConfig()
			if cfg.Platform.DefaultUserID == "" {
				printError("No default_user_id found. Run `memgo init --agent` first.", "")
				return errExit
			}
			printInfo("Your AGENTRUSH identifier:  " + cfg.Platform.DefaultUserID)
			return nil
		},
	}
}

// newStatusCmd。
func newStatusCmd() *cobra.Command {
	var output string
	var conn connFlags
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check connectivity and authentication.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, backend, err := getBackendAndConfig(conn.apiKey, conn.baseURL)
			if err != nil {
				return err
			}
			fireTelemetry(cfg, "cli.status", map[string]any{"command": "status"})
			res := backend.Status()
			if agentMode || output == "json" {
				agentEmit("status", res, 0, EntityIDs{}, -1)
				return nil
			}
			if res["connected"] == true {
				printSuccess(fmt.Sprintf("Connected to %s backend at %s", strOf(res["backend"]), strOf(res["base_url"])))
			} else {
				printError(fmt.Sprintf("Connection failed: %v", res["error"]), "")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output: text, json.")
	addConnFlags(cmd, &conn)
	return cmd
}

// newVersionCmd。
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version and exit.",
		RunE: func(cmd *cobra.Command, args []string) error {
			printInfo("memgo CLI v" + Version + " (Go)")
			return nil
		},
	}
}

// newHelpCmd (help --json 机器可读)。
func newHelpCmd() *cobra.Command {
	var asJSONFlag bool
	cmd := &cobra.Command{
		Use:   "help",
		Short: "Show help. Use --json for machine-readable output (for LLM agents).",
		RunE: func(cmd *cobra.Command, args []string) error {
			asJSON := asJSONFlag
			if asJSON || agentMode {
				printJSON(buildHelpJSON())
				return nil
			}
			printInfo("memgo CLI v" + Version + " (Go) — The Memory Layer for AI Agents")
			fmt.Println()
			fmt.Println("Usage: memgo <command> [OPTIONS]")
			fmt.Println()
			fmt.Println("Commands:")
			for _, line := range []string{
				"  add              Add a memory from text, messages, file, or stdin",
				"  search           Query your memory store (semantic, keyword, hybrid)",
				"  get              Get a specific memory by ID",
				"  list             List memories with optional filters",
				"  update           Update a memory's text or metadata",
				"  delete           Delete a memory, all memories, or an entity",
				"  import           Import memories from a JSON file",
				"  config           Manage configuration (show, get, set)",
				"  entity           Manage entities (list, delete)",
				"  event            Inspect background events (list, status)",
				"  init             Interactive setup wizard",
				"  status           Check connectivity and authentication",
			} {
				fmt.Println(line)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSONFlag, "json", false, "Output machine-readable JSON for LLM agents.")
	return cmd
}

// buildHelpJSON 对齐 _build_help_json (commands 面)。
func buildHelpJSON() map[string]any {
	return map[string]any{
		"name":        "memgo",
		"version":     Version,
		"description": "The Memory Layer for AI Agents",
		"commands": map[string]any{
			"add":    map[string]any{"description": "Add a memory from text, messages, file, or stdin.", "usage": "memgo add <text> [OPTIONS]"},
			"search": map[string]any{"description": "Query your memory store — semantic, keyword, or hybrid retrieval.", "usage": "memgo search <query> [OPTIONS]"},
			"get":    map[string]any{"description": "Get a specific memory by ID.", "usage": "memgo get <memory_id> [OPTIONS]"},
			"list":   map[string]any{"description": "List memories with optional filters.", "usage": "memgo list [OPTIONS]"},
			"update": map[string]any{"description": "Update a memory's text or metadata.", "usage": "memgo update <memory_id> [text] [OPTIONS]"},
			"delete": map[string]any{"description": "Delete a memory, all memories, or an entity.", "usage": "memgo delete [memory_id] [OPTIONS]"},
			"import": map[string]any{"description": "Import memories from a JSON file.", "usage": "memgo import <file_path> [OPTIONS]"},
			"config": map[string]any{"description": "Manage configuration (show, get, set)", "subcommands": map[string]any{
				"show": map[string]any{"usage": "memgo config show"},
				"get":  map[string]any{"usage": "memgo config get <key>"},
				"set":  map[string]any{"usage": "memgo config set <key> <value>"},
			}},
			"entity": map[string]any{"description": "Manage entities.", "subcommands": map[string]any{
				"list":   map[string]any{"usage": "memgo entity list <entity_type> [OPTIONS]"},
				"delete": map[string]any{"usage": "memgo entity delete [OPTIONS]"},
			}},
			"event": map[string]any{"description": "Inspect background processing events.", "subcommands": map[string]any{
				"list":   map[string]any{"usage": "memgo event list [OPTIONS]"},
				"status": map[string]any{"usage": "memgo event status <event_id> [OPTIONS]"},
			}},
			"init":   map[string]any{"description": "Interactive setup wizard.", "usage": "memgo init"},
			"status": map[string]any{"description": "Check connectivity and authentication.", "usage": "memgo status [OPTIONS]"},
		},
		"global_options": map[string]any{
			"--api-key":      "Override API key (env: MEMGO_API_KEY).",
			"--base-url":     "Override API base URL.",
			"--json/--agent": "Output as JSON for agent/programmatic use.",
			"--help":         "Show help for a command.",
			"--version":      "Show version and exit.",
		},
	}
}
