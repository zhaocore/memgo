package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newImportCmd (对齐 utils.cmd_import: JSON 文件逐条 add)。
func newImportCmd() *cobra.Command {
	var scope scopeFlags
	var output string
	var conn connFlags
	cmd := &cobra.Command{
		Use:   "import <file_path>",
		Short: "Import memories from a JSON file.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, backend, err := getBackendAndConfig(conn.apiKey, conn.baseURL)
			if err != nil {
				return err
			}
			fireTelemetry(cfg, "cli.import", map[string]any{"command": "import"})
			imported, err := importFile(backend, args[0], resolveIDs(cfg, scope.ids()))
			if err != nil {
				return backendErr(err)
			}
			if agentMode || output == "agent" {
				agentEmit("import", map[string]any{"imported": imported}, 0, EntityIDs{}, imported)
				return nil
			}
			printSuccess(fmt.Sprintf("Imported %d memories.", imported))
			return nil
		},
	}
	cmd.Flags().StringVarP(&scope.userID, "user-id", "u", "", "Override user ID.")
	cmd.Flags().StringVar(&scope.agentID, "agent-id", "", "Override agent ID.")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output: text, json.")
	addConnFlags(cmd, &conn)
	return cmd
}

// newConfigCmd config show/get/set。
func newConfigCmd() *cobra.Command {
	cfgCmd := &cobra.Command{Use: "config", Short: "Manage memgo configuration."}
	cfgCmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Display current configuration (secrets redacted).",
		RunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")
			cfg := LoadConfig()
			data := map[string]any{
				"version":   cfg.Version,
				"defaults":  cfg.Defaults,
				"platform":  map[string]any{"api_key": RedactKey(cfg.Platform.APIKey), "base_url": cfg.Platform.BaseURL, "user_email": cfg.Platform.UserEmail},
				"telemetry": cfg.Telemetry,
			}
			if agentMode || output == "json" {
				agentEmit("config.show", data, -1, EntityIDs{}, -1)
				return nil
			}
			printJSON(data)
			return nil
		},
	})
	cfgCmd.Commands()[0].Flags().StringVarP(new(string), "output", "o", "text", "Output: text, json.")
	cfgCmd.AddCommand(&cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := LoadConfig()
			v, ok := configLookup(cfg, args[0])
			if !ok {
				printError(fmt.Sprintf("Unknown config key: %s", args[0]), "Try platform.api_key / defaults.user_id / api_key (short).")
				return errExit
			}
			fmt.Println(v)
			return nil
		},
	})
	cfgCmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := LoadConfig()
			if !configSet(cfg, args[0], args[1]) {
				printError(fmt.Sprintf("Unknown config key: %s", args[0]), "")
				return errExit
			}
			if err := SaveConfig(cfg); err != nil {
				return backendErr(&APIError{Kind: "api", Msg: err.Error()})
			}
			printSuccess(fmt.Sprintf("Set %s = %s", args[0], args[1]))
			return nil
		},
	})
	return cfgCmd
}

// configLookup 对齐 get_nested_value + SHORT_KEY_ALIASES。
func configLookup(cfg *Config, key string) (string, bool) {
	switch key {
	case "api_key", "platform.api_key":
		return cfg.Platform.APIKey, true
	case "base_url", "platform.base_url":
		return cfg.Platform.BaseURL, true
	case "user_email", "platform.user_email":
		return cfg.Platform.UserEmail, true
	case "user_id", "defaults.user_id":
		return cfg.Defaults.UserID, true
	case "agent_id", "defaults.agent_id":
		return cfg.Defaults.AgentID, true
	case "app_id", "defaults.app_id":
		return cfg.Defaults.AppID, true
	case "run_id", "defaults.run_id":
		return cfg.Defaults.RunID, true
	}
	return "", false
}

// configSet 对齐 set_nested_value。
func configSet(cfg *Config, key, value string) bool {
	switch key {
	case "api_key", "platform.api_key":
		cfg.Platform.APIKey = value
	case "base_url", "platform.base_url":
		cfg.Platform.BaseURL = value
	case "user_email", "platform.user_email":
		cfg.Platform.UserEmail = value
	case "user_id", "defaults.user_id":
		cfg.Defaults.UserID = value
	case "agent_id", "defaults.agent_id":
		cfg.Defaults.AgentID = value
	case "app_id", "defaults.app_id":
		cfg.Defaults.AppID = value
	case "run_id", "defaults.run_id":
		cfg.Defaults.RunID = value
	default:
		return false
	}
	return true
}

// newEntityCmd entity list/delete。
func newEntityCmd() *cobra.Command {
	entCmd := &cobra.Command{Use: "entity", Short: "Manage entities."}
	listCmd := &cobra.Command{
		Use:   "list <entity_type>",
		Short: "List all entities of a given type.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")
			apiKey, _ := cmd.Flags().GetString("api-key")
			baseURL, _ := cmd.Flags().GetString("base-url")
			_, backend, err := getBackendAndConfig(apiKey, baseURL)
			if err != nil {
				return err
			}
			fireTelemetry(LoadConfig(), "cli.entity.list", map[string]any{"command": "entity.list"})
			items, err := backend.Entities(args[0])
			if err != nil {
				return backendErr(err)
			}
			if agentMode || output == "json" {
				agentEmit("entity.list", items, 0, EntityIDs{}, len(items))
				return nil
			}
			for _, e := range items {
				fmt.Printf("%s  %s  (memories: %v)\n", strOf(e["type"]), strOf(e["id"]), e["total_memories"])
			}
			return nil
		},
	}
	listCmd.Flags().StringVarP(new(string), "output", "o", "table", "Output: table, json.")
	addConnFlags(listCmd, &connFlags{})
	delCmd := newEntityDeleteCmd()
	entCmd.AddCommand(listCmd, delCmd)
	return entCmd
}

// newEntityDeleteCmd 供 entity delete 与 delete --entity 复用。
func newEntityDeleteCmd() *cobra.Command {
	var scope scopeFlags
	var force, dryRun bool
	var output string
	var conn connFlags
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an entity and ALL its memories (cascade).",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, backend, err := getBackendAndConfig(conn.apiKey, conn.baseURL)
			if err != nil {
				return err
			}
			fireTelemetry(LoadConfig(), "cli.entity.delete", map[string]any{"command": "entity.delete"})
			if dryRun {
				printInfo("Would delete the entity and all its memories (dry run).")
				return nil
			}
			if !force && !confirmPrompt("Delete the entity and ALL its memories (cascade)?") {
				printInfo("Aborted.")
				return nil
			}
			res, err := backend.DeleteEntities(scope.ids())
			if err != nil {
				return backendErr(err)
			}
			emitResult("entity.delete", res, output, 0, EntityIDs{}, -1)
			return nil
		},
	}
	addScopeFlags(cmd, &scope)
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation.")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deleted without deleting.")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output: text, json, quiet.")
	addConnFlags(cmd, &conn)
	return cmd
}

// newEventCmd event list/status (OSS 显式不支持)。
func newEventCmd() *cobra.Command {
	evCmd := &cobra.Command{Use: "event", Short: "Inspect background processing events."}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List recent background processing events.",
		RunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")
			apiKey, _ := cmd.Flags().GetString("api-key")
			baseURL, _ := cmd.Flags().GetString("base-url")
			_, backend, err := getBackendAndConfig(apiKey, baseURL)
			if err != nil {
				return err
			}
			fireTelemetry(LoadConfig(), "cli.event.list", map[string]any{"command": "event.list"})
			items, err := backend.ListEvents()
			if err != nil {
				return backendErr(err)
			}
			if agentMode || output == "json" {
				agentEmit("event.list", items, 0, EntityIDs{}, len(items))
				return nil
			}
			for _, e := range items {
				fmt.Printf("%s  %s  %s\n", strOf(e["event_id"]), strOf(e["status"]), strOf(e["created_at"]))
			}
			return nil
		},
	}
	listCmd.Flags().StringVarP(new(string), "output", "o", "table", "Output: table, json.")
	addConnFlags(listCmd, &connFlags{})
	statusCmd := &cobra.Command{
		Use:   "status <event_id>",
		Short: "Check the status of a specific background event.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			output, _ := cmd.Flags().GetString("output")
			apiKey, _ := cmd.Flags().GetString("api-key")
			baseURL, _ := cmd.Flags().GetString("base-url")
			_, backend, err := getBackendAndConfig(apiKey, baseURL)
			if err != nil {
				return err
			}
			fireTelemetry(LoadConfig(), "cli.event.status", map[string]any{"command": "event.status"})
			res, err := backend.GetEvent(args[0])
			if err != nil {
				return backendErr(err)
			}
			if agentMode || output == "json" {
				agentEmit("event.status", res, 0, EntityIDs{}, -1)
				return nil
			}
			printJSON(res)
			return nil
		},
	}
	statusCmd.Flags().StringVarP(new(string), "output", "o", "text", "Output: text, json.")
	addConnFlags(statusCmd, &connFlags{})
	evCmd.AddCommand(listCmd, statusCmd)
	return evCmd
}
