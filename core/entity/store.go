package entity

import (
	"fmt"
	"strings"

	"github.com/zhao-core/memgo/core/embedder"
	"github.com/zhao-core/memgo/core/vectorstore"
)

// Store 封装实体 collection 上的 upsert / 解链 / 加成逻辑。
// 对齐 Memory._upsert_entity / _remove_memory_from_entity_store / _compute_entity_boosts。
type Store struct {
	vec   vectorstore.VectorStore
	emb   embedder.Embedder
	scope func(filters map[string]any) map[string]any
}

// NewStore 构造; scope 抽取 user_id/agent_id/run_id 非空键 (对齐 Python 内联推导)。
func NewStore(vec vectorstore.VectorStore, emb embedder.Embedder) *Store {
	return &Store{vec: vec, emb: emb, scope: ScopeKeys}
}

// ScopeKeys 抽取实体查询作用域键 (仅非空 user_id/agent_id/run_id)。
func ScopeKeys(filters map[string]any) map[string]any {
	out := map[string]any{}
	for _, k := range []string{"user_id", "agent_id", "run_id"} {
		if v, ok := filters[k]; ok {
			if s, ok := v.(string); ok && s != "" {
				out[k] = s
			}
		}
	}
	return out
}

// NormalizeText 对齐 _normalize_entity_text: strip + lower + 内部空白折叠。
func NormalizeText(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

// EntityCollectionName 对齐 _entity_collection_name (pgvector 分隔符 "_")。
func EntityCollectionName(provider, collectionName string) string {
	sep := "_"
	if provider == "s3_vectors" {
		sep = "-"
	}
	return collectionName + sep + "entities"
}

// Upsert 对齐 _upsert_entity: 精确文本匹配优先, 次语义匹配(≥0.95), 否则新建。
// 任何失败仅告警级吞掉 (Python: logger.warning), 不破坏主链路。
func (s *Store) Upsert(entityText, entityType, memoryID string, filters map[string]any, warn func(string)) {
	defer func() {
		if r := recover(); r != nil && warn != nil {
			warn(fmt.Sprintf("entity upsert panic: %v", r))
		}
	}()
	searchFilters := s.scope(filters)
	embedding, err := s.emb.Embed(entityText, "add")
	if err != nil {
		if warn != nil {
			warn(fmt.Sprintf("Entity upsert failed for %q: %v", entityText, err))
		}
		return
	}
	exact := s.existingByText(searchFilters)[NormalizeText(entityText)]
	var match *vectorstore.OutputData
	if exact != nil {
		match = exact
	} else {
		results, err := s.vec.Search(entityText, embedding, 1, searchFilters)
		if err == nil && len(results) > 0 && results[0].Score != nil && *results[0].Score >= 0.95 {
			match = &results[0]
		}
	}
	if match != nil {
		payload := match.Payload
		linked := linkedIDs(payload)
		if !contains(linked, memoryID) {
			linked = append(linked, memoryID)
			payload["linked_memory_ids"] = linked
			_ = s.vec.Update(match.ID, nil, payload)
		}
		return
	}
	newPayload := map[string]any{
		"data":              entityText,
		"entity_type":       entityType,
		"linked_memory_ids": []string{memoryID},
	}
	for k, v := range searchFilters {
		newPayload[k] = v
	}
	_ = s.vec.Insert([][]float64{embedding}, []string{newUUID()}, []map[string]any{newPayload})
}

// existingByText 对齐 _existing_entities_by_text: list 全量后按归一化文本索引。
func (s *Store) existingByText(filters map[string]any) map[string]*vectorstore.OutputData {
	out := map[string]*vectorstore.OutputData{}
	listed, err := s.vec.List(filters, 10000)
	if err != nil {
		return out
	}
	for _, rows := range listed {
		for i := range rows {
			text, _ := rows[i].Payload["data"].(string)
			key := NormalizeText(text)
			if key != "" && out[key] == nil {
				row := rows[i]
				out[key] = &row
			}
		}
	}
	return out
}

// linkedIDs 读 payload.linked_memory_ids (容错非列表)。
func linkedIDs(payload map[string]any) []string {
	raw, ok := payload["linked_memory_ids"].([]any)
	if !ok {
		return []string{}
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

// RemoveMemoryFromStore 对齐 _remove_memory_from_entity_store:
// 从所有关联实体剥离 memory_id, 清空则删实体, 否则重嵌并更新 payload。
func (s *Store) RemoveMemoryFromStore(memoryID string, filters map[string]any) {
	searchFilters := s.scope(filters)
	listed, err := s.vec.List(searchFilters, 10000)
	if err != nil {
		return
	}
	for _, rows := range listed {
		for i := range rows {
			row := rows[i]
			linked := linkedIDs(row.Payload)
			if !contains(linked, memoryID) {
				continue
			}
			remaining := make([]string, 0, len(linked))
			for _, id := range linked {
				if id != memoryID {
					remaining = append(remaining, id)
				}
			}
			if len(remaining) == 0 {
				_ = s.vec.Delete(row.ID)
				continue
			}
			text, _ := row.Payload["data"].(string)
			if text == "" {
				continue
			}
			vec, err := s.emb.Embed(text, "update")
			if err != nil {
				continue
			}
			newPayload := map[string]any{}
			for k, v := range row.Payload {
				newPayload[k] = v
			}
			newPayload["linked_memory_ids"] = remaining
			_ = s.vec.Update(row.ID, vec, newPayload)
		}
	}
}

// EntityBoostWeight 对齐 scoring.ENTITY_BOOST_WEIGHT。
const EntityBoostWeight = 0.5

// ComputeBoosts 对齐 _compute_entity_boosts: 查询实体嵌入 → 实体库检索(≥0.5) →
// 关联记忆加成 = similarity * 0.5 * 记忆数衰减权重, 取逐记忆最大值。
func (s *Store) ComputeBoosts(queryEntities []Extracted, filters map[string]any) map[string]float64 {
	boosts := map[string]float64{}
	seen := map[string]bool{}
	var deduped []Extracted
	for _, e := range queryEntities {
		if len(deduped) >= 8 {
			break
		}
		key := NormalizeText(e.Text)
		if key != "" && !seen[key] {
			seen[key] = true
			deduped = append(deduped, e)
		}
	}
	if len(deduped) == 0 {
		return boosts
	}
	searchFilters := s.scope(filters)
	texts := make([]string, 0, len(deduped))
	for _, e := range deduped {
		texts = append(texts, e.Text)
	}
	embeddings, err := s.emb.EmbedBatch(texts, "search")
	if err != nil || len(embeddings) != len(texts) {
		return boosts
	}
	for i, e := range deduped {
		matches, err := s.vec.Search(e.Text, embeddings[i], 500, searchFilters)
		if err != nil {
			continue
		}
		for _, match := range matches {
			if match.Score == nil || *match.Score < 0.5 {
				continue
			}
			linked := linkedIDs(match.Payload)
			n := len(linked)
			if n < 1 {
				n = 1
			}
			weight := 1.0 / (1.0 + 0.001*float64((n-1)*(n-1)))
			boost := *match.Score * EntityBoostWeight * weight
			for _, mid := range linked {
				if mid == "" {
					continue
				}
				if boost > boosts[mid] {
					boosts[mid] = boost
				}
			}
		}
	}
	return boosts
}
