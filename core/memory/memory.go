// Package memory: 记忆引擎 (memgo/memory/main.py 同步 Memory 的 Go 等价, 无 HTTP 依赖)。
// 行为对齐以契约基线 (tests/contract goldens) 为准。
package memory

import (
	"fmt"
	"time"

	"github.com/zhao-core/memgo/core/config"
	"github.com/zhao-core/memgo/core/embedder"
	"github.com/zhao-core/memgo/core/entity"
	"github.com/zhao-core/memgo/core/graph"
	"github.com/zhao-core/memgo/core/history"
	"github.com/zhao-core/memgo/core/llm"
	"github.com/zhao-core/memgo/core/vectorstore"
)

// EntityExtractor 实体抽取接口 (entity 包实现; 测试可注入桩)。
type EntityExtractor interface {
	ExtractEntities(text string) []entity.Extracted
}

// Memory 汇聚四大依赖 (计划 D3: core 包无 HTTP)。
type Memory struct {
	Config             *config.MemoryConfig
	VectorStore        vectorstore.VectorStore
	EmbeddingModel     embedder.Embedder
	LLM                llm.LLM
	DB                 *history.Manager
	CollectionName     string
	CustomInstructions string

	entityStore     vectorstore.VectorStore // pgvector entity collection 实例
	entityExtractor EntityExtractor
	graphIndex      *graph.GraphIndex // 内存图索引 (Build 后非 nil)
	graphCfg        config.GraphMemoryConfig
}

// New 按配置装配 Memory (对齐 Memory.__init__; provider 工厂由调用方注入,
// pgvector/openai 组合由 server 与 cli 构造)。
func New(cfg *config.MemoryConfig, vec vectorstore.VectorStore, emb embedder.Embedder, language llm.LLM, db *history.Manager, extractor EntityExtractor) *Memory {
	gmCfg := cfg.GraphMemory
	if gmCfg.MaxHops == 0 {
		gmCfg = config.DefaultGraphMemoryConfig()
	}
	return &Memory{
		Config:             cfg,
		VectorStore:        vec,
		EmbeddingModel:     emb,
		LLM:                language,
		DB:                 db,
		CollectionName:     strOrCfg(cfg.VectorStore.Config["collection_name"]),
		CustomInstructions: cfg.CustomInstructions,
		entityExtractor:    extractor,
		graphCfg:           gmCfg,
	}
}

// BuildGraphIndex 在 entity store 注入后调用, 从 pgvector 全量构建图索引。
// 幂等: 多次调用会重建索引。需在 SetEntityStore 之后调用。
func (m *Memory) BuildGraphIndex() error {
	if !m.graphCfg.EnableMultiHop {
		return nil
	}
	m.graphIndex = graph.NewGraphIndex()
	store := newEntityStore(m)
	if err := store.BuildIndex(nil); err != nil {
		m.graphIndex = nil
		return err
	}
	// 同时将图索引注入到 entity store (供 Upsert/Remove 回调同步)
	store.SetGraphIndex(m.graphIndex)
	return nil
}

// EntityStore 惰性初始化实体 collection (对齐 entity_store 属性; collection 命名规则对齐)。
func (m *Memory) EntityStore() vectorstore.VectorStore {
	if m.entityStore == nil {
		return m.VectorStore // 由调用方以实体 collection 配置注入时替换; 此处保守返回主库
	}
	return m.entityStore
}

// SetEntityStore 注入实体 collection 实例 (工厂按 EntityCollectionName 建好)。
func (m *Memory) SetEntityStore(vs vectorstore.VectorStore) {
	m.entityStore = vs
}

// nowUTC 对齐 Python datetime.now(timezone.utc).isoformat(): 微秒 6 位 + "+00:00"。
func nowUTC() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000000") + "+00:00"
}

// newUUID RFC4122 v4。
func newUUID() string {
	b := make([]byte, 16)
	_, _ = randRead(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func strOrCfg(v any) string {
	s, _ := v.(string)
	return s
}
