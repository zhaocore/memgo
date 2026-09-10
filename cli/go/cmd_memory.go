package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// resolveIDs 对齐 _resolve_ids: 显式 id 或配置默认。
func resolveIDs(cfg *Config, ids EntityIDs) EntityIDs {
	if ids.UserID != "" || ids.AgentID != "" || ids.AppID != "" || ids.RunID != "" {
		return ids
	}
	return EntityIDs{
		UserID: cfg.Defaults.UserID, AgentID: cfg.Defaults.AgentID,
		AppID: cfg.Defaults.AppID, RunID: cfg.Defaults.RunID,
	}
}

// runAdd 对齐 cmd_add (text/messages/file/stdin 四输入源)。
func runAdd(cfg *Config, backend Backend, text string, ids EntityIDs, messagesJSON, file, metadataJSON string, immutable, noInfer bool, expires, customInstructions, structuredDataSchema string, timestamp int64, output, categories, agentCustomInstructions, customCategories string) error {
	var messages []map[string]any
	switch {
	case messagesJSON != "":
		if err := json.Unmarshal([]byte(messagesJSON), &messages); err != nil {
			printError("--messages 必须是 JSON 数组", "")
			return errExit
		}
	case file != "":
		raw, err := os.ReadFile(file)
		if err != nil {
			printError(fmt.Sprintf("读取文件失败: %v", err), "")
			return errExit
		}
		if err := json.Unmarshal(raw, &messages); err != nil {
			printError("--file 内容必须是消息 JSON 数组", "")
			return errExit
		}
	}
	if text == "" && len(messages) == 0 {
		if line := readStdinLine(); line != "" {
			text = line
		}
	}
	if text == "" && len(messages) == 0 {
		printError("Provide text, --messages, --file, or pipe to stdin.", "")
		return errExit
	}
	var metadata map[string]any
	if metadataJSON != "" {
		if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
			printError("--metadata 必须是 JSON 对象", "")
			return errExit
		}
	}
	if categories != "" {
		printError("--categories is not supported on add, use --custom-categories instead.", "")
		return errExit
	}
	p := AddParams{
		Content: text, Messages: messages, IDs: ids, Metadata: metadata,
		Immutable: immutable, Infer: !noInfer, Expires: expires,
		CustomInstructions: customInstructions, StructuredDataSchema: structuredDataSchema,
		AgentCustomInstructions: agentCustomInstructions, CustomCategories: customCategories,
	}
	if timestamp > 0 {
		p.Timestamp = &timestamp
	}
	started := nowMillis()
	res, err := backend.Add(p)
	if err != nil {
		return backendErr(err)
	}
	emitResult("add", res, output, started, ids, -1)
	return nil
}

// runSearch 对齐 cmd_search (stdin query 回退)。
func runSearch(cfg *Config, backend Backend, query string, ids EntityIDs, topK int, threshold float64, rerank, keyword bool, filterJSON, fields string, showExpired, latestOnly bool, referenceDate, output string) error {
	if query == "" {
		query = readStdinLine()
	}
	if strings.TrimSpace(query) == "" {
		printError("Search query cannot be empty.", "")
		return errExit
	}
	var filterMap map[string]any
	if filterJSON != "" {
		if err := json.Unmarshal([]byte(filterJSON), &filterMap); err != nil {
			printError("--filter 必须是 JSON 对象", "")
			return errExit
		}
	}
	var fieldList []string
	if fields != "" {
		for _, f := range strings.Split(fields, ",") {
			if strings.TrimSpace(f) != "" {
				fieldList = append(fieldList, strings.TrimSpace(f))
			}
		}
	}
	started := nowMillis()
	res, err := backend.Search(SearchParams{
		Query: query, IDs: ids, TopK: topK, Threshold: threshold,
		Rerank: rerank, Keyword: keyword, FilterJSON: filterMap,
		Fields: fieldList, ShowExpired: showExpired, ReferenceDate: referenceDate, LatestOnly: latestOnly,
	})
	if err != nil {
		return backendErr(err)
	}
	emitResult("search", res, output, started, ids, len(res))
	return nil
}

var errExit = fmt.Errorf("exit")

// readStdinLine 管道输入 (非 TTY)。
func readStdinLine() string {
	stat, err := os.Stdin.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) != 0 {
		return ""
	}
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func nowMillis() int64 { return int64(timenow().UnixMilli()) }

// emitResult 统一输出分发 (text/json/table/quiet/agent)。
func emitResult(command string, data any, output string, started int64, ids EntityIDs, count int) {
	if agentMode || output == "agent" {
		agentEmit(command, data, started, ids, count)
		return
	}
	switch output {
	case "json":
		printJSON(data)
	case "quiet":
		// 仅 id 列表
		if l, ok := data.([]map[string]any); ok {
			for _, m := range l {
				fmt.Println(strOf(m["id"]))
			}
		}
	case "table":
		if l, ok := data.([]map[string]any); ok {
			formatMemoriesTable(l, command == "search")
		}
		printSuccess("")
	default:
		if l, ok := data.([]map[string]any); ok {
			formatMemoriesText(l, "memories", command == "search")
		} else {
			printJSON(data)
		}
	}
}

// agentEmit agent 信封 (sanitize_agent_data 的简化等价: 原样透传)。
func agentEmit(command string, data any, started int64, ids EntityIDs, count int) {
	scope := map[string]any{}
	if ids.UserID != "" {
		scope["user_id"] = ids.UserID
	}
	if ids.AgentID != "" {
		scope["agent_id"] = ids.AgentID
	}
	if ids.RunID != "" {
		scope["run_id"] = ids.RunID
	}
	if len(scope) == 0 {
		scope = nil
	}
	env := map[string]any{"status": "success", "command": command}
	if scope != nil {
		env["scope"] = scope
	}
	if count >= 0 {
		env["count"] = count
	}
	if n := TakeNotice(); n != "" {
		env["mem0_notice"] = n
	}
	env["data"] = data
	printJSON(env)
}

// atoiDefault 安全转 int。
func atoiDefault(s string, def int) int {
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
