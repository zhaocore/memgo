// Package graph 提供内存图索引, 在 entity store 的 pgvector 实体数据之上构建
// 双邻接表 (entitiesToMemories, memoriesToEntities), 支持多跳 BFS 遍历以增强记忆检索。
// v3 native Graph Memory 的实体-记忆关联图理念, 不依赖外部图数据库。
//
// 本包不依赖 entity / vectorstore / memory 包, 避免循环引用。
// Build 逻辑由调用方 (entity.Store 或 memory.Memory) 负责编排, 通过 Build 回调注入。
package graph

import (
	"sync"
)

// GraphIndex 维护实体→记忆 (前向) 与记忆→实体 (反向) 双邻接表内存索引。
// 零外部依赖 (无 Neo4j), 启动时从 entity store 全量构建, 运行时通过回调增量维护。
// 线程安全 (sync.RWMutex)。
type GraphIndex struct {
	mu sync.RWMutex
	// entityID (pgvector entity row id) → linked memory IDs
	entitiesToMemories map[string][]string
	// memoryID → linked entity IDs
	memoriesToEntities map[string][]string
}

// NewGraphIndex 返回空图索引, 后续调用 Build 填充。
func NewGraphIndex() *GraphIndex {
	return &GraphIndex{
		entitiesToMemories: make(map[string][]string),
		memoriesToEntities: make(map[string][]string),
	}
}

// Build 从 entity store 全量扫描构建双邻接表。scope 限制了扫描的 entity 作用域
// (user_id/agent_id/run_id), 可传 nil 表示不限域。
// onRow 回调接收每条 entity 的 (entityID, linked_memory_ids), 由调用方提供 List 能力。
func (g *GraphIndex) Build(onRow func(onEntity func(entityID string, linkedMemoryIDs []string)) error) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.entitiesToMemories = make(map[string][]string)
	g.memoriesToEntities = make(map[string][]string)

	return onRow(func(entityID string, linkedMemoryIDs []string) {
		if len(linkedMemoryIDs) == 0 {
			return
		}
		// 前向: entity → memories
		copied := make([]string, len(linkedMemoryIDs))
		copy(copied, linkedMemoryIDs)
		g.entitiesToMemories[entityID] = copied

		// 反向: memory → entities
		for _, mid := range linkedMemoryIDs {
			g.memoriesToEntities[mid] = appendUnique(g.memoriesToEntities[mid], entityID)
		}
	})
}

// OnEntityUpsert 实体变更时增量同步双邻接表。
// oldLinkedMemoryIDs 是变更前的旧关联 (初次创建时传 nil)。
func (g *GraphIndex) OnEntityUpsert(entityID string, linkedMemoryIDs, oldLinkedMemoryIDs []string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 清旧关联: 从反向索引中移除 → 从前向索引删除
	if len(oldLinkedMemoryIDs) > 0 {
		for _, mid := range oldLinkedMemoryIDs {
			g.memoriesToEntities[mid] = removeUnique(g.memoriesToEntities[mid], entityID)
		}
	}

	// 写新关联
	if len(linkedMemoryIDs) > 0 {
		copied := make([]string, len(linkedMemoryIDs))
		copy(copied, linkedMemoryIDs)
		g.entitiesToMemories[entityID] = copied
		for _, mid := range linkedMemoryIDs {
			g.memoriesToEntities[mid] = appendUnique(g.memoriesToEntities[mid], entityID)
		}
	} else {
		delete(g.entitiesToMemories, entityID)
	}
}

// OnMemoryRemove 删除记忆时清理反向索引中该记忆的所有实体引用,
// 并从前向索引中逐实体移除该记忆。
func (g *GraphIndex) OnMemoryRemove(memoryID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	entityIDs, ok := g.memoriesToEntities[memoryID]
	if !ok {
		return
	}
	delete(g.memoriesToEntities, memoryID)

	for _, eid := range entityIDs {
		g.entitiesToMemories[eid] = removeUnique(g.entitiesToMemories[eid], memoryID)
		if len(g.entitiesToMemories[eid]) == 0 {
			delete(g.entitiesToMemories, eid)
		}
	}
}

// MultiHopResult 多跳遍历的一条命中结果。
type MultiHopResult struct {
	MemoryID string
	Boost    float64 // 累计 boost (同记忆多次到达取最大值)
	Hop      int     // 首次发现该记忆的跳数
}

// MultiHopSearch 执行 BFS 多跳遍历。
//   - entityIDs: 查询中匹配到的实体 ID 集合 (已在 entity store 中语义匹配过)
//   - maxHops: 最大跳数 (1=仅直达, 2=直达+一跳邻接, …)
//   - decayFactor: 逐跳衰减因子 (如 0.5, 跳数 n 的 boost = decayFactor^(n-1))
//
// 算法:
//
//	Hop 0: 从 entityIDs 出发 → 获取直达记忆集合, boost = 1.0
//	Hop 1: 从 hop 0 的记忆出发 → 查反向索引得到关联实体 → 排除已访问实体 →
//	       查这些实体的 linked_memory_ids → hop 1 记忆, boost = decay
//	Hop N: 重复
//
// 返回 map[memoryID]MultiHopResult, 同记忆多次到达取 max boost。
func (g *GraphIndex) MultiHopSearch(entityIDs []string, maxHops int, decayFactor float64) map[string]MultiHopResult {
	if len(entityIDs) == 0 || maxHops < 1 {
		return nil
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	result := make(map[string]MultiHopResult)
	visitedEntities := make(map[string]bool)
	visitedMemories := make(map[string]bool)

	// Hop 0: 直达记忆
	currentEntities := make([]string, 0, len(entityIDs))
	for _, eid := range entityIDs {
		if visitedEntities[eid] {
			continue
		}
		visitedEntities[eid] = true
		currentEntities = append(currentEntities, eid)

		for _, mid := range g.entitiesToMemories[eid] {
			if visitedMemories[mid] {
				continue
			}
			visitedMemories[mid] = true
			result[mid] = MultiHopResult{MemoryID: mid, Boost: 1.0, Hop: 0}
		}
	}

	// Hop 1..N
	for hop := 1; hop < maxHops; hop++ {
		// 收集当前层所有记忆的关联实体
		var nextEntities []string
		for mid := range visitedMemories {
			for _, eid := range g.memoriesToEntities[mid] {
				if visitedEntities[eid] {
					continue
				}
				visitedEntities[eid] = true
				nextEntities = append(nextEntities, eid)
			}
		}

		if len(nextEntities) == 0 {
			break
		}

		boost := 1.0
		for i := 0; i < hop; i++ {
			boost *= decayFactor
		}

		for _, eid := range nextEntities {
			for _, mid := range g.entitiesToMemories[eid] {
				if visitedMemories[mid] {
					// 已访问: 只在 boost 更高时更新
					if existing, ok := result[mid]; ok && boost > existing.Boost {
						result[mid] = MultiHopResult{MemoryID: mid, Boost: boost, Hop: hop}
					}
					continue
				}
				visitedMemories[mid] = true
				result[mid] = MultiHopResult{MemoryID: mid, Boost: boost, Hop: hop}
			}
		}
	}

	return result
}

// Stats 返回索引统计 (调试用)。
func (g *GraphIndex) Stats() map[string]int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return map[string]int{
		"entities":    len(g.entitiesToMemories),
		"memories":    len(g.memoriesToEntities),
		"total_edges": g.countEdges(),
	}
}

func (g *GraphIndex) countEdges() int {
	c := 0
	for _, ids := range g.entitiesToMemories {
		c += len(ids)
	}
	return c
}

// GetMemoriesForEntity 前向查询: entityID → linked memory IDs (调试/测试用)。
func (g *GraphIndex) GetMemoriesForEntity(entityID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := g.entitiesToMemories[entityID]
	out := make([]string, len(ids))
	copy(out, ids)
	return out
}

// GetEntitiesForMemory 反向查询: memoryID → linked entity IDs (调试/测试用)。
func (g *GraphIndex) GetEntitiesForMemory(memoryID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	ids := g.memoriesToEntities[memoryID]
	out := make([]string, len(ids))
	copy(out, ids)
	return out
}

// appendUnique 追加去重。
func appendUnique(slice []string, v string) []string {
	for _, s := range slice {
		if s == v {
			return slice
		}
	}
	return append(slice, v)
}

// removeUnique 删除单元素 (不保序, 如有多个仅删首个)。
func removeUnique(slice []string, v string) []string {
	for i, s := range slice {
		if s == v {
			slice[i] = slice[len(slice)-1]
			return slice[:len(slice)-1]
		}
	}
	return slice
}

// MemoryBoosts 将 MultiHopSearch 的结果转为 map[memoryID]boost (兼容现有 entity boost 接口)。
func MemoryBoosts(results map[string]MultiHopResult) map[string]float64 {
	boosts := make(map[string]float64, len(results))
	for mid, r := range results {
		boosts[mid] = r.Boost
	}
	return boosts
}
