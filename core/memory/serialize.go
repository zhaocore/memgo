package memory

// serializeMemoryItem 对齐 MemoryItem.model_dump() 形状 (get 用, 含 metadata/score null 键)。
// payload 为向量行 payload; promoted 键提升, 其余进 metadata (非空才置)。
func serializeMemoryItem(id string, payload map[string]any, score *float64, includeScore bool) map[string]any {
	item := map[string]any{
		"id":         id,
		"memory":     payload["data"],
		"hash":       payload["hash"],
		"metadata":   nil,
		"score":      nil,
		"created_at": payload["created_at"],
		"updated_at": payload["updated_at"],
	}
	if !includeScore {
		delete(item, "score")
	} else if score != nil {
		item["score"] = *score
	}
	for _, k := range promotedPayloadKeys {
		if v, ok := payload[k]; ok {
			item[k] = v
		}
	}
	additional := map[string]any{}
	for k, v := range payload {
		if !isCoreOrPromotedKey(k) {
			additional[k] = v
		}
	}
	if len(additional) > 0 {
		if existing, ok := item["metadata"].(map[string]any); ok {
			for k, v := range additional {
				existing[k] = v
			}
		} else {
			item["metadata"] = additional
		}
	}
	return item
}
