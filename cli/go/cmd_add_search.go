package cli

import (
	"github.com/spf13/cobra"
)

// newAddCmd 对齐 app.py add 选项矩阵。
func newAddCmd() *cobra.Command {
	var (
		scope      scopeFlags
		messages   string
		file       string
		metadata   string
		immutable  bool
		noInfer    bool
		expires    string
		categories string
		custInstr  string
		agentInstr string
		custCats   string
		structSch  string
		timestamp  int64
		output     string
		conn       connFlags
	)
	cmd := &cobra.Command{
		Use:   "add [text]",
		Short: "Add a memory from text, messages, file, or stdin.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, backend, err := getBackendAndConfig(conn.apiKey, conn.baseURL)
			if err != nil {
				return err
			}
			fireTelemetry(cfg, "cli.add", map[string]any{"command": "add"})
			text := ""
			if len(args) > 0 {
				text = args[0]
			}
			return runAdd(cfg, backend, text, resolveIDs(cfg, scope.ids()),
				messages, file, metadata, immutable, noInfer, expires,
				custInstr, structSch, timestamp, output, categories, agentInstr, custCats)
		},
	}
	addScopeFlags(cmd, &scope)
	cmd.Flags().StringVar(&messages, "messages", "", "Conversation messages as JSON.")
	cmd.Flags().StringVarP(&file, "file", "f", "", "Read messages from JSON file.")
	cmd.Flags().StringVarP(&metadata, "metadata", "m", "", "Custom metadata as JSON.")
	cmd.Flags().BoolVar(&immutable, "immutable", false, "Prevent future updates.")
	cmd.Flags().BoolVar(&noInfer, "no-infer", false, "Skip inference, store raw.")
	cmd.Flags().StringVar(&expires, "expires", "", "Expiration date (YYYY-MM-DD).")
	cmd.Flags().StringVar(&categories, "categories", "", "Not supported on add, use --custom-categories instead.")
	cmd.Flags().StringVar(&custInstr, "custom-instructions", "", "Custom instructions for fact extraction.")
	cmd.Flags().StringVar(&agentInstr, "agent-custom-instructions", "", "Extraction instructions for agent-scoped memories, overriding the project setting.")
	cmd.Flags().StringVar(&custCats, "custom-categories", "", "Custom categories as a JSON array of {name: description} objects.")
	cmd.Flags().StringVar(&structSch, "structured-data-schema", "", "Schema for structured data extraction, as JSON.")
	cmd.Flags().Int64Var(&timestamp, "timestamp", 0, "Unix timestamp for the memory.")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output format: text, json, quiet.")
	addConnFlags(cmd, &conn)
	return cmd
}

// newSearchCmd 对齐 app.py search 选项矩阵。
func newSearchCmd() *cobra.Command {
	var (
		scope         scopeFlags
		topK          int
		threshold     float64
		rerank        bool
		keyword       bool
		filterJSON    string
		fields        string
		showExpired   bool
		referenceDate string
		latestOnly    bool
		output        string
		conn          connFlags
	)
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Query your memory store — semantic, keyword, or hybrid retrieval.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, backend, err := getBackendAndConfig(conn.apiKey, conn.baseURL)
			if err != nil {
				return err
			}
			fireTelemetry(cfg, "cli.search", map[string]any{"command": "search"})
			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			return runSearch(cfg, backend, query, resolveIDs(cfg, scope.ids()),
				topK, threshold, rerank, keyword, filterJSON, fields,
				showExpired, latestOnly, referenceDate, output)
		},
	}
	addScopeFlags(cmd, &scope)
	cmd.Flags().IntVarP(&topK, "top-k", "k", 10, "Number of results.")
	cmd.Flags().IntVar(&topK, "limit", 10, "Alias of --top-k.")
	cmd.Flags().Float64Var(&threshold, "threshold", 0.3, "Minimum similarity score.")
	cmd.Flags().BoolVar(&rerank, "rerank", false, "Enable reranking (Platform only).")
	cmd.Flags().BoolVar(&keyword, "keyword", false, "Use keyword search.")
	cmd.Flags().StringVar(&filterJSON, "filter", "", `Advanced filter as JSON: {"AND": [...]} or {"OR": [...]}.`)
	cmd.Flags().StringVar(&fields, "fields", "", "Specific fields to return (comma-separated).")
	cmd.Flags().BoolVar(&showExpired, "show-expired", false, "Include expired memories.")
	cmd.Flags().StringVar(&referenceDate, "reference-date", "", "Reference date for relative queries (YYYY-MM-DD or unix timestamp).")
	cmd.Flags().BoolVar(&latestOnly, "latest-only", false, "Only return the latest version of each memory.")
	cmd.Flags().StringVarP(&output, "output", "o", "text", "Output: text, json, table.")
	addConnFlags(cmd, &conn)
	return cmd
}
