package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// newGetCmd。
func newGetCmd() *cobra.Command {
	var output string
	var conn connFlags
	cmd := &cobra.Command{
		Use:   "get <memory_id>",
		Short: "Get a specific memory by ID.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, backend, err := getBackendAndConfig(conn.apiKey, conn.baseURL)
			if err != nil {
				return err
			}
			fireTelemetry(LoadConfig(), "cli.get", map[string]any{"command": "get"})
			res, err := backend.Get(args[0])
			if err != nil {
				return backendErr(err)
			}
			emitResult("get", res, output, 0, EntityIDs{}, -1)
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output: text, json.")
	addConnFlags(cmd, &conn)
	return cmd
}

// newListCmd。
func newListCmd() *cobra.Command {
	var scope scopeFlags
	var page, pageSize int
	var category, after, before string
	var showExpired, latestOnly bool
	var output string
	var conn connFlags
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List memories with optional filters.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, backend, err := getBackendAndConfig(conn.apiKey, conn.baseURL)
			if err != nil {
				return err
			}
			fireTelemetry(cfg, "cli.list", map[string]any{"command": "list"})
			res, err := backend.ListMemories(ListParams{
				IDs: resolveIDs(cfg, scope.ids()), Page: page, PageSize: pageSize,
				Category: category, After: after, Before: before,
				ShowExpired: showExpired, LatestOnly: latestOnly,
			})
			if err != nil {
				return backendErr(err)
			}
			emitResult("list", res, output, 0, EntityIDs{}, len(res))
			return nil
		},
	}
	addScopeFlags(cmd, &scope)
	cmd.Flags().IntVar(&page, "page", 1, "Page number.")
	cmd.Flags().IntVar(&pageSize, "page-size", 100, "Results per page.")
	cmd.Flags().StringVar(&category, "category", "", "Filter by category.")
	cmd.Flags().StringVar(&after, "after", "", "Created after (YYYY-MM-DD).")
	cmd.Flags().StringVar(&before, "before", "", "Created before (YYYY-MM-DD).")
	cmd.Flags().BoolVar(&showExpired, "show-expired", false, "Include expired memories.")
	cmd.Flags().BoolVar(&latestOnly, "latest-only", false, "Only return the latest version of each memory.")
	cmd.Flags().StringVarP(&output, "output", "o", "table", "Output: text, json, table.")
	addConnFlags(cmd, &conn)
	return cmd
}

// newUpdateCmd。
func newUpdateCmd() *cobra.Command {
	var metadata, expires string
	var timestamp int64
	var output string
	var conn connFlags
	cmd := &cobra.Command{
		Use:   "update <memory_id> [text]",
		Short: "Update a memory's text or metadata.",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, backend, err := getBackendAndConfig(conn.apiKey, conn.baseURL)
			if err != nil {
				return err
			}
			fireTelemetry(LoadConfig(), "cli.update", map[string]any{"command": "update"})
			text := ""
			if len(args) > 1 {
				text = args[1]
			} else if line := readStdinLine(); line != "" {
				text = line
			}
			var meta map[string]any
			if metadata != "" {
				if err := json.Unmarshal([]byte(metadata), &meta); err != nil {
					printError("--metadata 必须是 JSON 对象", "")
					return errExit
				}
			}
			p := UpdateParams{MemoryID: args[0], Content: text, Metadata: meta, ExpirationDate: expires}
			if timestamp > 0 {
				p.Timestamp = &timestamp
			}
			res, err := backend.Update(p)
			if err != nil {
				return backendErr(err)
			}
			emitResult("update", res, output, 0, EntityIDs{}, -1)
			return nil
		},
	}
	cmd.Flags().StringVarP(&metadata, "metadata", "m", "", "Update metadata (JSON).")
	cmd.Flags().StringVar(&expires, "expires", "", "Expiration date (YYYY-MM-DD).")
	cmd.Flags().Int64Var(&timestamp, "timestamp", 0, "Unix timestamp for the memory.")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output: text, json, quiet.")
	addConnFlags(cmd, &conn)
	return cmd
}

// newDeleteCmd (单条/--all/--entity 三模式互斥)。
func newDeleteCmd() *cobra.Command {
	var allFlag, entityFlag, projectFlag, dryRun, force, deleteLinked bool
	var scope scopeFlags
	var output string
	var conn connFlags
	cmd := &cobra.Command{
		Use:   "delete [memory_id]",
		Short: "Delete a memory, all memories, or an entity.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := LoadConfig()
			memoryID := ""
			if len(args) > 0 {
				memoryID = args[0]
			}
			modes := 0
			if memoryID != "" {
				modes++
			}
			if allFlag {
				modes++
			}
			if entityFlag {
				modes++
			}
			if modes > 1 {
				printError("Only one of memory ID, --all, or --entity may be used at a time.", "")
				return errExit
			}
			if modes == 0 {
				printError("Provide a memory ID, --all, or --entity.", "Run 'memgo delete --help' for usage.")
				return errExit
			}
			if memoryID != "" {
				fireTelemetry(cfg, "cli.delete", map[string]any{"command": "delete", "delete_mode": "single"})
				_, backend, err := getBackendAndConfig(conn.apiKey, conn.baseURL)
				if err != nil {
					return err
				}
				if dryRun {
					printInfo(fmt.Sprintf("Would delete memory %s (dry run).", memoryID))
					return nil
				}
				if !force && !confirmPrompt(fmt.Sprintf("Delete memory %s?", memoryID)) {
					printInfo("Aborted.")
					return nil
				}
				res, err := backend.Delete(DeleteParams{MemoryID: memoryID, DeleteLinked: deleteLinked})
				if err != nil {
					return backendErr(err)
				}
				emitResult("delete", res, output, 0, EntityIDs{}, -1)
				return nil
			}
			if allFlag {
				fireTelemetry(cfg, "cli.delete", map[string]any{"command": "delete", "delete_mode": "all"})
				_, backend, err := getBackendAndConfig(conn.apiKey, conn.baseURL)
				if err != nil {
					return err
				}
				if projectFlag {
					return backendErr(&APIError{Kind: "api", Msg: "--project is a platform-only concept; on OSS use server admin POST /reset"})
				}
				if dryRun {
					printInfo("Would delete all memories matching scope filters (dry run).")
					return nil
				}
				if !force && !confirmPrompt("Delete ALL memories matching scope filters?") {
					printInfo("Aborted.")
					return nil
				}
				res, err := backend.Delete(DeleteParams{All: true, IDs: resolveIDs(cfg, scope.ids())})
				if err != nil {
					return backendErr(err)
				}
				emitResult("delete", res, output, 0, EntityIDs{}, -1)
				return nil
			}
			// --entity
			fireTelemetry(cfg, "cli.delete", map[string]any{"command": "delete", "delete_mode": "entity"})
			_, backend, err := getBackendAndConfig(conn.apiKey, conn.baseURL)
			if err != nil {
				return err
			}
			if dryRun {
				printInfo("Would delete the entity and all its memories (dry run).")
				return nil
			}
			if !force && !confirmPrompt("Delete the entity and ALL its memories (cascade)?") {
				printInfo("Aborted.")
				return nil
			}
			res, err := backend.DeleteEntities(resolveIDs(cfg, scope.ids()))
			if err != nil {
				return backendErr(err)
			}
			emitResult("delete", res, output, 0, EntityIDs{}, -1)
			return nil
		},
	}
	cmd.Flags().BoolVar(&allFlag, "all", false, "Delete all memories matching scope filters.")
	cmd.Flags().BoolVar(&entityFlag, "entity", false, "Delete the entity itself and all its memories (cascade).")
	cmd.Flags().BoolVar(&projectFlag, "project", false, "With --all: delete ALL memories project-wide.")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be deleted without deleting.")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation.")
	cmd.Flags().BoolVar(&deleteLinked, "delete-linked", false, "Also delete memories linked to this memory.")
	addScopeFlags(cmd, &scope)
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output: text, json, quiet.")
	addConnFlags(cmd, &conn)
	return cmd
}

// confirmPrompt TTY 确认; 非 TTY 默认拒绝 (安全)。
func confirmPrompt(question string) bool {
	if IsAgentMode() {
		return false
	}
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) == 0 {
		return false
	}
	fmt.Printf("%s [y/N]: ", question)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	}
	return false
}
