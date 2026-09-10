package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// pyErr 复刻 pydantic v2 错误项 (golden 锁定形状: type/loc/msg/input[/ctx])。
type pyErr struct {
	Type  string         `json:"type"`
	Loc   []any          `json:"loc"`
	Msg   string         `json:"msg"`
	Input any            `json:"input"`
	Ctx   map[string]any `json:"ctx,omitempty"`
}

// write422 输出 {"detail": [...]}。
func write422(w http.ResponseWriter, errs []pyErr) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"detail": errs})
}

func missingErr(loc []any, input any) pyErr {
	return pyErr{Type: "missing", Loc: loc, Msg: "Field required", Input: input}
}

// readBody 读取并解析 JSON; 空体视为 {}; 非法 JSON → 422 json_invalid (近似)。
func readBody(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, bool) {
	raw, err := io.ReadAll(r.Body)
	if err != nil || len(strings.TrimSpace(string(raw))) == 0 {
		return map[string]json.RawMessage{}, true
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		// 根非对象
		var anyVal any
		if json.Unmarshal(raw, &anyVal) == nil {
			write422(w, []pyErr{{Type: "model_attributes_type", Loc: []any{"body"},
				Msg: "Input should be a valid dictionary or object to extract fields from", Input: anyVal}})
			return nil, false
		}
		write422(w, []pyErr{{Type: "json_invalid", Loc: []any{"body"},
			Msg: "JSON decode error", Input: string(raw)}})
		return nil, false
	}
	return fields, true
}

// requireObject 确认 body 是 JSON 对象 (根类型错误)。
func requireObject(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, bool) {
	raw, _ := io.ReadAll(r.Body)
	if len(strings.TrimSpace(string(raw))) == 0 {
		return map[string]json.RawMessage{}, true
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		var anyVal any
		if json.Unmarshal(raw, &anyVal) == nil {
			write422(w, []pyErr{{Type: "model_attributes_type", Loc: []any{"body"},
				Msg: "Input should be a valid dictionary or object to extract fields from", Input: anyVal}})
			return nil, false
		}
		write422(w, []pyErr{{Type: "json_invalid", Loc: []any{"body"}, Msg: "JSON decode error", Input: string(raw)}})
		return nil, false
	}
	return fields, true
}

// validateStringField 校验必填/可选字符串 (pydantic string_type/missing 语义)。
func validateStringField(errs *[]pyErr, fields map[string]json.RawMessage, body any, name string, required bool) *string {
	raw, ok := fields[name]
	if !ok || string(raw) == "null" {
		if required {
			*errs = append(*errs, missingErr([]any{"body", name}, body))
		}
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		*errs = append(*errs, pyErr{Type: "string_type", Loc: []any{"body", name},
			Msg: "Input should be a valid string", Input: json.RawMessage(raw)})
		return nil
	}
	return &s
}

// validateQueryInt 校验 query int 边界 (pydantic ge/le; input 为字符串)。
func validateQueryInt(w http.ResponseWriter, r *http.Request, name string, def, ge, le int, hasGE, hasLE bool) (int, bool) {
	q := r.URL.Query()
	rawVals, present := q[name]
	if !present || rawVals[0] == "" {
		return def, true
	}
	rawVal := rawVals[0]
	v, err := strconv.Atoi(rawVal)
	if err != nil {
		write422(w, []pyErr{{Type: "int_parsing", Loc: []any{"query", name},
			Msg: "Input should be a valid integer, unable to parse string as an integer", Input: rawVal}})
		return 0, false
	}
	if hasGE && v < ge {
		write422(w, []pyErr{{Type: "greater_than_equal", Loc: []any{"query", name},
			Msg: fmt.Sprintf("Input should be greater than or equal to %d", ge), Input: rawVal, Ctx: map[string]any{"ge": ge}}})
		return 0, false
	}
	if hasLE && v > le {
		write422(w, []pyErr{{Type: "less_than_equal", Loc: []any{"query", name},
			Msg: fmt.Sprintf("Input should be less than or equal to %d", le), Input: rawVal, Ctx: map[string]any{"le": le}}})
		return 0, false
	}
	return v, true
}

// isNotFoundErr 对齐 _client_error: ValueError 且含 "not found" → 404。
func isNotFoundErr(err error) bool {
	return containsFold(err.Error(), "not found")
}

// isClientErr 其余 ValueError 类 → 400。
func isClientErr(err error) bool {
	msg := err.Error()
	for _, prefix := range []string{
		"At least one", "Invalid ", "expiration_date must be", "messages must be",
		"filters must contain", "Unsupported filter operator", "Filter operator",
		"Invalid query", "Invalid threshold", "Invalid top_k",
	} {
		if strings.HasPrefix(msg, prefix) {
			return true
		}
	}
	return false
}

func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}
