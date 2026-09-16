package prompts

import (
	"fmt"
	"strings"
	"time"

	"github.com/zhao-core/memgo/core/config"
)

// PastMessageTruncationLimit: Python PAST_MESSAGE_TRUNCATION_LIMIT = 300。
const PastMessageTruncationLimit = 300

// truncateContent 对齐 Python _truncate_content: 超限截断并追加 "..."。
func truncateContent(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	return text[:limit] + "..."
}

// formatConversationHistory 对齐 Python _format_conversation_history:
// role 取 "role" 键(默认空), content 优先 "message" 后 "content"(默认空), 两者非空才输出。
func formatConversationHistory(messages []map[string]any) string {
	if len(messages) == 0 {
		return ""
	}
	var b strings.Builder
	for _, msg := range messages {
		role, _ := msg["role"].(string)
		content := msg["message"]
		if content == nil {
			content = msg["content"]
		}
		contentStr, _ := content.(string)
		if role != "" && contentStr != "" {
			b.WriteString(role)
			b.WriteString(": ")
			b.WriteString(truncateContent(contentStr, PastMessageTruncationLimit))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// serializeMemories 对齐 Python _serialize_memories: json.dumps(v or [], ensure_ascii=False)。
// 注意 Python json.dumps 默认分隔符为 ", " / ": ", 与 Go json.Marshal 紧凑输出不同,
// 故用 pythonJSON 复刻。
func serializeMemories(memories []map[string]any) string {
	if memories == nil {
		memories = []map[string]any{}
	}
	return pythonJSON(memories)
}

// formatNewMessages 对齐 Python _format_new_messages: 字符串透传, 否则 JSON 序列化。
func formatNewMessages(newMessages any) string {
	if s, ok := newMessages.(string); ok {
		return s
	}
	return pythonJSON(newMessages)
}

// pythonJSON 复刻 Python json.dumps(v, ensure_ascii=False) 的输出形状:
// 分隔符 ", "/": ", 非 ASCII 不转义, HTML 字符不转义。
func pythonJSON(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return pyString(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	case float64:
		return pyFloat(x)
	case []any:
		parts := make([]string, 0, len(x))
		for _, item := range x {
			parts = append(parts, pythonJSON(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case []map[string]any:
		parts := make([]string, 0, len(x))
		for _, item := range x {
			parts = append(parts, pythonJSON(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sortStrings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, pyString(k)+": "+pythonJSON(x[k]))
		}
		return "{" + strings.Join(parts, ", ") + "}"
	default:
		return pyString(fmt.Sprintf("%v", x))
	}
}

// pyString 复刻 Python str 的 JSON 转义 (ensure_ascii=False): 仅转义 \ " 与控制字符。
func pyString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// pyFloat 复刻 Python float 的 JSON 表示 (repr 规则, 整数值不带小数点)。
func pyFloat(f float64) string {
	if f == float64(int64(f)) && f < 1e15 && f > -1e15 {
		return fmt.Sprintf("%d.0", int64(f))
	}
	return fmt.Sprintf("%g", f)
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// GenerateAdditiveExtractionParams 对齐 Python generate_additive_extraction_prompt 的关键字参数面。
type GenerateAdditiveExtractionParams struct {
	Summary                   any // string 或 map(含 "summary" 键); nil = 无
	RecentlyExtractedMemories []map[string]any
	ExistingMemories          []map[string]any
	NewMessages               any // string 透传或 []map[string]any
	LastKMessages             []map[string]any
	CurrentDate               string // YYYY-MM-DD; 空 = 今天(UTC)
	Timestamp                 string // observation date 回退用; 空 = CurrentDate
	CustomInstructions        string
}

// GenerateAdditiveExtractionPrompt 对齐 Python generate_additive_extraction_prompt
// (use_input_language 恒 false —— OSS add 路径不传该参)。
func GenerateAdditiveExtractionPrompt(p GenerateAdditiveExtractionParams) string {
	currentDate := p.CurrentDate
	if currentDate == "" {
		currentDate = time.Now().UTC().Format("2006-01-02")
	}
	observationDate := p.Timestamp
	if observationDate == "" {
		observationDate = currentDate
	}

	sections := []string{
		"## Summary\n" + formatSummary(p.Summary),
		"## Last k Messages\n" + formatConversationHistory(p.LastKMessages),
		"## Recently Extracted Memories\n" + serializeMemories(p.RecentlyExtractedMemories),
		"## Existing Memories\n" + serializeMemories(p.ExistingMemories),
		"## New Messages\n" + formatNewMessages(p.NewMessages),
		"## Observation Date\n" + observationDate,
		"## Current Date\n" + currentDate,
	}
	if p.CustomInstructions != "" {
		sections = append(sections, "## Custom Instructions\n"+p.CustomInstructions)
	}
	sections = append(sections, "# Output:")
	return strings.Join(sections, "\n\n")
}

// formatSummary 对齐 Python _format_summary: dict 取 "summary" 键, 其余原样(nil → "")。
func formatSummary(summary any) string {
	if summary == nil {
		return ""
	}
	if m, ok := summary.(map[string]any); ok {
		s, _ := m["summary"].(string)
		return s
	}
	s, _ := summary.(string)
	return s
}

// CATEGORY_CLASSIFICATION_SYSTEM_PROMPT 分类打标 system prompt。
// memgo 扩展 (custom-categories 分类打标), 非 parity 锁定。
const CATEGORY_CLASSIFICATION_SYSTEM_PROMPT = `You classify memories into categories.

You will receive a category catalog (name and description) and a numbered list of memories. For each memory, pick the single closest matching category from the catalog. Every memory must be classified: always choose the closest category even if the match is imperfect.

Respond with JSON only — no markdown, no extra text — in exactly this shape:
{"categories": [{"id": 1, "category": "<category name>"}]}

"id" is the memory number from the input. Output exactly one entry per memory, in input order, using category names exactly as given in the catalog.`

// GenerateCategoryClassificationParams 分类打标 user prompt 参数。
type GenerateCategoryClassificationParams struct {
	Memories   []string          // 待打标记忆文本 (按序, 编号 1..n)
	Categories []config.Category // 生效分类目录
}

// GenerateCategoryClassificationPrompt 拼装分类打标 user prompt (目录 + 编号记忆列表)。
func GenerateCategoryClassificationPrompt(p GenerateCategoryClassificationParams) string {
	var b strings.Builder
	b.WriteString("## Categories\n")
	for _, c := range p.Categories {
		b.WriteString("- " + c.Name)
		if c.Description != "" {
			b.WriteString(": " + c.Description)
		}
		b.WriteString("\n")
	}
	b.WriteString("\n## Memories\n")
	for i, text := range p.Memories {
		fmt.Fprintf(&b, "%d. %s\n", i+1, text)
	}
	return strings.TrimRight(b.String(), "\n")
}
