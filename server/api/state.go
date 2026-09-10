// Package api: HTTP 层 (doc-02 合同实现)。AppState 对齐 server_state.py。
package api

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/zhao-core/memgo/core/config"
	corehistory "github.com/zhao-core/memgo/core/history"
	"github.com/zhao-core/memgo/core/memory"
	"github.com/zhao-core/memgo/core/vectorstore"
)

// AppState 全局单例 (config 深合并 + Memory 重建)。
type AppState struct {
	mu       sync.Mutex
	cfgMap   map[string]any
	memory   *memory.Memory
	history  *corehistory.Manager
	mainVec  vectorstore.VectorStore
	entVec   vectorstore.VectorStore
	closeOld func()
}

// NewAppState 初始化: default + overrides 深合并 + 建 Memory。
func NewAppState(defaultConfig map[string]any, loadOverrides func() (map[string]any, error)) (*AppState, error) {
	cfgMap := deepCopyMap(defaultConfig)
	overrides, err := loadOverrides()
	if err != nil {
		overrides = nil // 对齐 _load_overrides: 任何异常吞掉返回 {}
	}
	if len(overrides) > 0 {
		cfgMap = config.DeepMerge(cfgMap, overrides)
	}
	st := &AppState{cfgMap: cfgMap}
	if err := st.rebuild(); err != nil {
		return nil, err
	}
	return st, nil
}

// Memory 读取 (锁内; Python get_memory_instance 等价)。
func (s *AppState) Memory() *memory.Memory {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.memory
}

// CurrentConfig 深拷贝。
func (s *AppState) CurrentConfig() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return deepCopyMap(s.cfgMap)
}

// UpdateConfig 对齐 update_config: 深合并 → 重建 → overrides 落库 → 返回深拷贝。
func (s *AppState) UpdateConfig(updates map[string]any, persist func(merged map[string]any) error) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := config.DeepMerge(s.cfgMap, updates)
	if err := s.rebuildFrom(next); err != nil {
		return nil, err
	}
	s.cfgMap = next
	if persist != nil {
		if err := persist(deepCopyMap(next)); err != nil {
			log.Printf("config overrides 落库失败: %v", err)
		}
	}
	return deepCopyMap(s.cfgMap), nil
}

// rebuild 用当前 cfgMap 重建实例。
func (s *AppState) rebuild() error { return s.rebuildFrom(s.cfgMap) }

// rebuildFrom 按配置 map 构建全套依赖 (对齐 Memory.from_config: 失败即崩)。
func (s *AppState) rebuildFrom(cfgMap map[string]any) error {
	parsed, err := config.ParseMemoryConfig(cfgMap)
	if err != nil {
		return fmt.Errorf("配置解析失败: %w", err)
	}
	mainVec, err := buildVectorStore(parsed.VectorStore)
	if err != nil {
		return err
	}
	entityCollection := parseStr(parsed.VectorStore.Config, "collection_name", "memgo") + "_entities"
	entCfg := deepCopyMap(parsed.VectorStore.Config)
	entCfg["collection_name"] = entityCollection
	entVec, err := buildVectorStore(config.ProviderConfig{Provider: parsed.VectorStore.Provider, Config: entCfg})
	if err != nil {
		return err
	}
	language, err := buildLLM(parsed.LLM)
	if err != nil {
		return err
	}
	emb, err := buildEmbedder(parsed.Embedder)
	if err != nil {
		return err
	}
	db, err := corehistory.NewManager(parsed.HistoryDBPath)
	if err != nil {
		return err
	}
	mem := memory.New(parsed, mainVec, emb, language, db, nil)
	oldClose := s.closeOld
	s.mainVec, s.entVec, s.history, s.memory = mainVec, entVec, db, mem
	s.closeOld = func() {
		mainVec.Close()
		entVec.Close()
		_ = db.Close()
	}
	if oldClose != nil {
		oldClose()
	}
	return nil
}

// parseStr 取 config 键 (带默认)。
func parseStr(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok && v != "" {
		return v
	}
	return def
}

// deepCopyMap JSON 往返深拷贝。
func deepCopyMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	raw, _ := json.Marshal(src)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	return out
}
