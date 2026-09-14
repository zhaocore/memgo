package memory

import (
	"fmt"
	"math"
	"strings"

	"github.com/zhao-core/memgo/core/entity"
	"github.com/zhao-core/memgo/core/jsonx"
)

// SearchParams 对齐 Memory.search 参数面 (rerank 由 reranker 未实现而恒 false, 合同未暴露)。
type SearchParams struct {
	Query       string
	TopK        int
	Filters     map[string]any
	Threshold   *float64
	Explain     bool
	ShowExpired bool
}

// Search 对齐 Memory.search。
func (m *Memory) Search(p SearchParams) (map[string]any, error) {
	trimmedQuery, err := validateAndTrimSearchQuery(p.Query)
	if err != nil {
		return nil, err
	}
	effective, err := trimEntityFilters(p.Filters)
	if err != nil {
		return nil, err
	}
	if !hasEntityFilter(effective) {
		return nil, fmt.Errorf("filters must contain at least one of: user_id, agent_id, run_id. Example: filters={'user_id': 'u1'}")
	}
	threshold := 0.1
	if p.Threshold != nil {
		if err := validateSearchParams(p.Threshold, &p.TopK); err != nil {
			return nil, err
		}
		threshold = *p.Threshold
	} else if err := validateSearchParams(nil, &p.TopK); err != nil {
		return nil, err
	}
	if hasAdvancedOperators(effective) {
		processed := processMetadataFilters(effective)
		for _, lk := range []string{"AND", "OR", "NOT"} {
			delete(effective, lk)
		}
		for fk := range effective {
			if fk == "user_id" || fk == "agent_id" || fk == "run_id" {
				continue
			}
			if _, ok := effective[fk].(map[string]any); ok {
				delete(effective, fk)
			}
		}
		for k, v := range processed {
			effective[k] = v
		}
	}
	results, err := m.searchVectorStore(trimmedQuery, effective, p.TopK, threshold, p.Explain, p.ShowExpired)
	if err != nil {
		return nil, err
	}
	return map[string]any{"results": results}, nil
}

// hasAdvancedOperators 对齐 _has_advanced_operators。
func hasAdvancedOperators(filters map[string]any) bool {
	for key, value := range filters {
		switch key {
		case "AND", "OR", "NOT":
			return true
		}
		if m, ok := value.(map[string]any); ok {
			for op := range m {
				switch op {
				case "eq", "ne", "gt", "gte", "lt", "lte", "in", "nin", "contains", "icontains":
					return true
				}
			}
		}
		if value == "*" {
			return true
		}
	}
	return false
}

// processMetadataFilters 对齐 _process_metadata_filters (AND 展开合并, OR/NOT → $or/$not)。
func processMetadataFilters(metadataFilters map[string]any) map[string]any {
	processed := map[string]any{}
	for key, value := range metadataFilters {
		switch key {
		case "AND":
			list, ok := value.([]any)
			if !ok {
				continue
			}
			for _, cond := range list {
				cm, ok := cond.(map[string]any)
				if !ok {
					continue
				}
				for sk, sv := range cm {
					mergeFilterConds(processed, processCondition(sk, sv))
				}
			}
		case "OR":
			list, ok := value.([]any)
			if !ok || len(list) == 0 {
				continue
			}
			var ors []any
			for _, cond := range list {
				orCond := map[string]any{}
				if cm, ok := cond.(map[string]any); ok {
					for sk, sv := range cm {
						mergeFilterConds(orCond, processCondition(sk, sv))
					}
				}
				ors = append(ors, orCond)
			}
			processed["$or"] = ors
		case "NOT":
			list, ok := value.([]any)
			if !ok || len(list) == 0 {
				continue
			}
			var nots []any
			for _, cond := range list {
				notCond := map[string]any{}
				if cm, ok := cond.(map[string]any); ok {
					for sk, sv := range cm {
						mergeFilterConds(notCond, processCondition(sk, sv))
					}
				}
				nots = append(nots, notCond)
			}
			processed["$not"] = nots
		default:
			mergeFilterConds(processed, processCondition(key, value))
		}
	}
	return processed
}

// processCondition 对齐内联同名函数。
func processCondition(key string, condition any) map[string]any {
	if m, ok := condition.(map[string]any); ok {
		result := map[string]any{}
		inner := map[string]any{}
		for op, v := range m {
			switch op {
			case "eq", "ne", "gt", "gte", "lt", "lte", "in", "nin", "contains", "icontains":
				inner[op] = v
			default:
				return result // Python: raise ValueError; server 层已限 schema, 此处保守
			}
		}
		if len(inner) > 0 {
			result[key] = inner
		}
		return result
	}
	if condition == "*" {
		return map[string]any{key: "*"}
	}
	return map[string]any{key: condition}
}

// mergeFilterConds 对齐 merge_filters (嵌套操作符 dict 深合并)。
func mergeFilterConds(target, source map[string]any) {
	for key, value := range source {
		if existing, ok := target[key].(map[string]any); ok {
			if incoming, ok := value.(map[string]any); ok {
				for k, v := range incoming {
					existing[k] = v
				}
				continue
			}
		}
		target[key] = value
	}
}

// searchVectorStore 对齐 _search_vector_store: 语义检索 + BM25 + 实体加成 → score_and_rank。
func (m *Memory) searchVectorStore(query string, filters map[string]any, limit int, threshold float64, explain, showExpired bool) ([]map[string]any, error) {
	// Step 1-2: 查询预处理 + 嵌入 (无 spaCy 基线: lemmatize=恒等, 实体抽取=恒空)
	queryLemmatized := query
	queryEntities := entity.ExtractEntities(query)
	embeddings, err := m.EmbeddingModel.Embed(query, "search")
	if err != nil {
		return nil, err
	}
	internalLimit := limit * 4
	if internalLimit < 60 {
		internalLimit = 60
	}
	// Step 3: 语义检索
	semanticResults, err := m.VectorStore.Search(query, embeddings, internalLimit, filters)
	if err != nil {
		return nil, err
	}
	// Step 4: 关键词检索 (store 不支持 → nil)
	keywordResults, _ := m.VectorStore.KeywordSearch(queryLemmatized, internalLimit, filters)
	// Step 5: BM25 归一化
	bm25Scores := map[string]float64{}
	if keywordResults != nil {
		midpoint, steepness := bm25Params(queryLemmatized)
		for _, mem := range keywordResults {
			if mem.Score != nil && *mem.Score > 0 {
				bm25Scores[mem.ID] = normalizeBM25(*mem.Score, midpoint, steepness)
			}
		}
	}
	// Step 6: 实体加成 + 多跳图遍历
	entityBoosts := map[string]float64{}
	multiHopBoosts := map[string]float64{}
	if len(queryEntities) > 0 && m.entityStore != nil {
		store := newEntityStore(m)
		if m.graphIndex != nil && m.graphCfg.EnableMultiHop {
			entityBoosts, multiHopBoosts = store.ComputeGraphBoosts(queryEntities, filters, m.graphCfg.MaxHops, m.graphCfg.DecayFactor)
		} else {
			entityBoosts = store.ComputeBoosts(queryEntities, filters)
		}
	}
	// Step 6.5: 合并多跳加成到 entityBoosts (同记忆取 max)
	for mid, boost := range multiHopBoosts {
		if existing, ok := entityBoosts[mid]; !ok || boost > existing {
			entityBoosts[mid] = boost
		}
	}
	// Step 7: 候选集 (过期过滤 + 多跳记忆补充到候选集)
	var candidates []rankCandidate
	candidateIDs := map[string]bool{}
	for _, mem := range semanticResults {
		if !showExpired && payloadIsExpired(mem.Payload) {
			continue
		}
		score := 0.0
		if mem.Score != nil {
			score = *mem.Score
		}
		candidates = append(candidates, rankCandidate{id: mem.ID, score: score, payload: mem.Payload})
		candidateIDs[mem.ID] = true
	}
	// 多跳发现但不在语义结果中的记忆: 从向量库拉取
	for mid := range multiHopBoosts {
		if candidateIDs[mid] {
			continue
		}
		row, err := m.VectorStore.Get(mid)
		if err != nil || row == nil {
			continue
		}
		if !showExpired && payloadIsExpired(row.Payload) {
			continue
		}
		candidates = append(candidates, rankCandidate{id: row.ID, score: 0, payload: row.Payload})
	}
	// Step 8: 评分排序
	scored := scoreAndRank(candidates, bm25Scores, entityBoosts, threshold, limit, explain)
	// Step 9: 序列化
	results := make([]map[string]any, 0, len(scored))
	for _, s := range scored {
		payload := s.payload
		if strOr(payload["data"]) == "" {
			continue
		}
		score := s.score
		pyScore := jsonx.PyFloat(score)
		item := serializeMemoryItem(s.id, payload, nil, true)
		item["score"] = pyScore
		if explain {
			item["score_details"] = s.details
		}
		results = append(results, item)
	}
	return results, nil
}

// bm25Params 对齐 get_bm25_params (查询长度自适应 sigmoid 参数)。
func bm25Params(lemmatizedQuery string) (float64, float64) {
	numTerms := len(splitWords(lemmatizedQuery))
	if numTerms == 0 {
		numTerms = 1
	}
	switch {
	case numTerms <= 3:
		return 5.0, 0.7
	case numTerms <= 6:
		return 7.0, 0.6
	case numTerms <= 9:
		return 9.0, 0.5
	case numTerms <= 15:
		return 10.0, 0.5
	default:
		return 12.0, 0.5
	}
}

// splitWords 等价 Python str.split() (按空白切分)。
func splitWords(s string) []string {
	var out []string
	for _, w := range strings.Fields(s) {
		out = append(out, w)
	}
	return out
}

// normalizeBM25 对齐 normalize_bm25 (logistic sigmoid)。
func normalizeBM25(raw, midpoint, steepness float64) float64 {
	return 1.0 / (1.0 + math.Exp(-steepness*(raw-midpoint)))
}
