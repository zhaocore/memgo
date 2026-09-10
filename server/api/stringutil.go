package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/zhao-core/memgo/core/llm"
)

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func errStr(msg string) error { return errors.New(msg) }

// readAllJSON 读体并解为 map (空体 → 空 map)。
func readAllJSON(r *http.Request) (map[string]any, error) {
	raw, err := io.ReadAll(r.Body)
	if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := jsonUnmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func jsonUnmarshal(raw []byte, v any) error { return json.Unmarshal(raw, v) }

func splitOnce(s, sep string) []string {
	idx := strings.Index(s, sep)
	if idx < 0 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+len(sep):]}
}

func trimAll(s string) string { return strings.TrimSpace(s) }

func replaceAll(s, old, new string) string { return strings.ReplaceAll(s, old, new) }

type llmMessage = llm.Message

func genOptsNone() llm.GenerateOptions { return llm.GenerateOptions{} }
