package vectorstore

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// filterCond 是 _build_filter_conditions 的中间形态: SQL 片段 + 位置参数。
type filterCond struct {
	sql    string
	params []any
}

// buildFilterConditions 逐行对齐 Python pgvector._build_filter_conditions。
// 确定性: map 遍历按 key 排序 (Python dict 保序, 上游按插入序; Go 用字典序保证稳定)。
func buildFilterConditions(filters map[string]any) ([]filterCond, error) {
	var conds []filterCond
	if len(filters) == 0 {
		return conds, nil
	}
	keys := make([]string, 0, len(filters))
	for k := range filters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := filters[key]
		switch {
		case key == "$or":
			orGroups, err := buildGroups(value.([]any))
			if err != nil {
				return nil, err
			}
			if len(orGroups.sqls) > 0 {
				conds = append(conds, filterCond{sql: "(" + strings.Join(orGroups.sqls, " OR ") + ")", params: orGroups.params})
			}
		case key == "$not":
			notGroups, err := buildGroups(value.([]any))
			if err != nil {
				return nil, err
			}
			if len(notGroups.sqls) > 0 {
				conds = append(conds, filterCond{sql: "NOT (" + strings.Join(notGroups.sqls, " OR ") + ")", params: notGroups.params})
			}
		case value == "*":
			conds = append(conds, filterCond{sql: "payload ? %s", params: []any{key}})
		default:
			c, err := buildValueConds(key, value)
			if err != nil {
				return nil, err
			}
			conds = append(conds, c...)
		}
	}
	return conds, nil
}

type groupResult struct {
	sqls   []string
	params []any
}

// buildGroups 展开 $or/$not 的子过滤组。
func buildGroups(value any) (groupResult, error) {
	var res groupResult
	list, ok := value.([]any)
	if !ok {
		return res, fmt.Errorf("$or/$not: 必须是数组")
	}
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		subs, err := buildFilterConditions(m)
		if err != nil {
			return res, err
		}
		if len(subs) == 0 {
			continue
		}
		sqls := make([]string, 0, len(subs))
		for _, s := range subs {
			sqls = append(sqls, s.sql)
			res.params = append(res.params, s.params...)
		}
		res.sqls = append(res.sqls, "("+strings.Join(sqls, " AND ")+")")
	}
	return res, nil
}

// operatorSQLMap 对齐 Python OPERATOR_SQL_MAP (op → SQL 模板, 是否 numeric)。
var operatorSQLMap = map[string]struct {
	sql     string
	numeric bool
	isIn    bool
	isLike  bool
}{
	"eq":        {"payload->>%s = %s", false, false, false},
	"ne":        {"payload->>%s != %s", false, false, false},
	"gt":        {"(payload->>%s)::numeric > %s", true, false, false},
	"gte":       {"(payload->>%s)::numeric >= %s", true, false, false},
	"lt":        {"(payload->>%s)::numeric < %s", true, false, false},
	"lte":       {"(payload->>%s)::numeric <= %s", true, false, false},
	"in":        {"payload->>%s = ANY(%s)", false, true, false},
	"nin":       {"NOT (payload->>%s = ANY(%s))", false, true, false},
	"contains":  {"payload->>%s LIKE %s", false, false, true},
	"icontains": {"payload->>%s ILIKE %s", false, false, true},
}

// buildValueConds 构造单键的条件 (dict 操作符 / list / 标量)。
func buildValueConds(key string, value any) ([]filterCond, error) {
	if m, ok := value.(map[string]any); ok {
		var conds []filterCond
		ops := make([]string, 0, len(m))
		for op := range m {
			ops = append(ops, op)
		}
		sort.Strings(ops)
		for _, op := range ops {
			spec, ok := operatorSQLMap[op]
			if !ok {
				return nil, fmt.Errorf("Unsupported filter operator: %s", op)
			}
			opValue := m[op]
			switch {
			case spec.isIn:
				list, ok := opValue.([]any)
				if !ok {
					return nil, fmt.Errorf("Filter operator %q for key %q requires a list value", op, key)
				}
				strs := make([]string, 0, len(list))
				for _, v := range list {
					strs = append(strs, toStringValue(v))
				}
				conds = append(conds, filterCond{sql: spec.sql, params: []any{key, strs}})
			case spec.isLike:
				escaped := strings.ReplaceAll(toStringValue(opValue), "\\", "\\\\")
				escaped = strings.ReplaceAll(escaped, "%", "\\%")
				escaped = strings.ReplaceAll(escaped, "_", "\\_")
				conds = append(conds, filterCond{sql: spec.sql + " ESCAPE '\\'", params: []any{key, "%" + escaped + "%"}})
			case spec.numeric:
				f, err := toFloat(opValue)
				if err != nil {
					return nil, err
				}
				conds = append(conds, filterCond{sql: spec.sql, params: []any{key, f}})
			default:
				conds = append(conds, filterCond{sql: spec.sql, params: []any{key, toStringValue(opValue)}})
			}
		}
		return conds, nil
	}
	if list, ok := value.([]any); ok {
		strs := make([]string, 0, len(list))
		for _, v := range list {
			strs = append(strs, toStringValue(v))
		}
		return []filterCond{{sql: "payload->>%s = ANY(%s)", params: []any{key, strs}}}, nil
	}
	if b, ok := value.(bool); ok {
		js, _ := json.Marshal(b)
		return []filterCond{{sql: "payload->>%s = %s", params: []any{key, string(js)}}}, nil
	}
	return []filterCond{{sql: "payload->>%s = %s", params: []any{key, toStringValue(value)}}}, nil
}

// toStringValue 对齐 Python str(v) (bool 走 json 序列化的分支已在调用侧处理)。
func toStringValue(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		return fmt.Sprintf("%v", x)
	case int:
		return fmt.Sprintf("%d", x)
	case int64:
		return fmt.Sprintf("%d", x)
	case float64:
		if x == float64(int64(x)) && x < 1e15 && x > -1e15 {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%g", x)
	case nil:
		return "None"
	default:
		return fmt.Sprintf("%v", x)
	}
}

// toFloat 数值操作符参数转换。
func toFloat(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case float32:
		return float64(x), nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case string:
		var f float64
		if _, err := fmt.Sscanf(x, "%g", &f); err != nil {
			return 0, fmt.Errorf("数值过滤参数非法: %q", x)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("数值过滤参数非法: %v", v)
	}
}
