package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// 输出模式: text | json | table | quiet | agent (--json 全局)。
var agentMode bool

// SetAgentMode 全局 --json/--agent。
func SetAgentMode(v bool) { agentMode = v }

// IsAgentMode 查询。
func IsAgentMode() bool { return agentMode }

// printError 对齐 branding.print_error (agent 态输出错误信封)。
func printError(msg, hint string) {
	if agentMode {
		env := map[string]any{
			"status": "error", "command": currentCommand,
			"error": msg, "data": nil,
		}
		raw, _ := json.Marshal(env)
		fmt.Println(string(raw))
	} else {
		fmt.Fprintf(os.Stderr, "✗ Error: %s\n", msg)
		if hint != "" {
			fmt.Fprintf(os.Stderr, "  %s\n", hint)
		}
	}
}

var currentCommand string

// setCurrentCommand 记录 (错误信封用)。
func setCurrentCommand(name string) { currentCommand = name }

// envelope 对齐 format_json_envelope。
func envelope(command string, data any, count int, scope map[string]any) {
	env := map[string]any{"status": "success", "command": command}
	if scope != nil {
		env["scope"] = scope
	}
	if count >= 0 {
		env["count"] = count
	}
	if n := TakeNotice(); n != "" {
		env["memgo_notice"] = n
	}
	env["data"] = data
	raw, _ := json.Marshal(env)
	fmt.Println(string(raw))
}

// printJSON 紧凑输出 (json 模式)。
func printJSON(v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		raw = []byte("null")
	}
	fmt.Println(string(raw))
}

// printSuccess ✓ 行 (agent 态静默)。
func printSuccess(msg string) {
	if agentMode {
		return
	}
	fmt.Printf("✓ %s\n", msg)
}

// printInfo ◆ 行。
func printInfo(msg string) {
	if agentMode {
		return
	}
	fmt.Printf("◆ %s\n", msg)
}

// formatMemoriesText 对齐 format_memories_text (人读列表)。
func formatMemoriesText(memories []map[string]any, title string, showScore bool) {
	fmt.Printf("\nFound %d %s:\n\n", len(memories), title)
	for i, mem := range memories {
		memoryText := strOf(mem["memory"])
		if memoryText == "" {
			memoryText = strOf(mem["text"])
		}
		memID := strOf(mem["id"])
		if len(memID) > 8 {
			memID = memID[:8]
		}
		fmt.Printf("  %d. %s\n", i+1, memoryText)
		var details []string
		if showScore {
			if s, ok := mem["score"].(float64); ok {
				details = append(details, fmt.Sprintf("Score: %.2f", s))
			}
		}
		if memID != "" {
			details = append(details, "ID: "+memID)
		}
		if created := strOf(mem["created_at"]); created != "" {
			details = append(details, "Created: "+formatDate(created))
		}
		if len(details) > 0 {
			fmt.Printf("     %s\n", strings.Join(details, " · "))
		}
		fmt.Println()
	}
}

// formatDate 对齐 _format_date (YYYY-MM-DD HH:MM)。
func formatDate(iso string) string {
	if iso == "" {
		return ""
	}
	t := iso
	if idx := strings.IndexAny(t, "Tt"); idx > 0 && len(t) > idx+6 {
		return t[:idx] + " " + t[idx+1:idx+6]
	}
	if len(t) >= 16 {
		return t[:16]
	}
	return t
}

// formatMemoriesTable 对齐 format_memories_table (无 rich, 纯文本表)。
func formatMemoriesTable(memories []map[string]any, showScore bool) {
	fmt.Printf("%-38s  %-50s  %s\n", "ID", "Memory", "Created")
	if showScore {
		fmt.Printf("%-38s  %-7s  %-50s  %s\n", "", "Score", "", "")
	}
	for _, mem := range memories {
		memID := strOf(mem["id"])
		memoryText := strOf(mem["memory"])
		if memoryText == "" {
			memoryText = strOf(mem["text"])
		}
		if len(memoryText) > 60 {
			memoryText = memoryText[:57] + "..."
		}
		created := formatDate(strOf(mem["created_at"]))
		if showScore {
			score := "—"
			if s, ok := mem["score"].(float64); ok {
				score = fmt.Sprintf("%.2f", s)
			}
			fmt.Printf("%-38s  %7s  %-50s  %s\n", memID, score, memoryText, created)
		} else {
			fmt.Printf("%-38s  %-50s  %s\n", memID, memoryText, created)
		}
	}
}
