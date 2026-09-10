package embedder

import "strings"

// replaceNewlines 对齐 Python embed 的 text.replace("\n", " ")。
func replaceNewlines(text string) string {
	return strings.ReplaceAll(text, "\n", " ")
}
