package memory

import (
	"fmt"

	"github.com/zhao-core/memgo/core/entity"
	"github.com/zhao-core/memgo/core/history"
	"github.com/zhao-core/memgo/core/vectorstore"
)

// Get 对齐 Memory.get: 不存在返回 nil。
func (m *Memory) Get(memoryID string) (map[string]any, error) {
	row, err := m.VectorStore.Get(memoryID)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, nil
	}
	return serializeMemoryItem(row.ID, row.Payload, row.Score, true), nil
}

// GetAllParams 对齐 get_all 参数。
type GetAllParams struct {
	Filters     map[string]any
	TopK        int
	ShowExpired bool
}

// GetAll 对齐 Memory.get_all。
func (m *Memory) GetAll(p GetAllParams) (map[string]any, error) {
	effective, err := trimEntityFilters(p.Filters)
	if err != nil {
		return nil, err
	}
	if !hasEntityFilter(effective) {
		return nil, fmt.Errorf("filters must contain at least one of: user_id, agent_id, run_id. Example: filters={'user_id': 'u1'}")
	}
	if err := validateSearchParams(nil, &p.TopK); err != nil {
		return nil, err
	}
	fetchLimit := p.TopK * 4
	if fetchLimit < 60 {
		fetchLimit = 60
	}
	if p.ShowExpired {
		fetchLimit = p.TopK
	}
	rows, err := m.listRows(effective, fetchLimit)
	if err != nil {
		return nil, err
	}
	results := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		if !p.ShowExpired && payloadIsExpired(row.Payload) {
			continue
		}
		if len(results) >= p.TopK {
			break
		}
		results = append(results, serializeMemoryItem(row.ID, row.Payload, row.Score, false))
	}
	return map[string]any{"results": results}, nil
}

// listRows 归一 list 嵌套返回 (复刻 _vector_store_list_rows)。
func (m *Memory) listRows(filters map[string]any, limit int) ([]vectorstore.OutputData, error) {
	listed, err := m.VectorStore.List(filters, limit)
	if err != nil {
		return nil, err
	}
	if len(listed) > 0 && len(listed[0]) >= 0 {
		return listed[0], nil
	}
	return nil, nil
}

// trimEntityFilters 校验并裁剪 filters 中的实体 id。
func trimEntityFilters(filters map[string]any) (map[string]any, error) {
	effective := map[string]any{}
	for k, v := range filters {
		effective[k] = v
	}
	for _, key := range []string{"user_id", "agent_id", "run_id"} {
		if v, ok := effective[key]; ok && v != nil {
			trimmed, err := validateAndTrimEntityID(v, key)
			if err != nil {
				return nil, err
			}
			effective[key] = trimmed
		}
	}
	return effective, nil
}

func hasEntityFilter(filters map[string]any) bool {
	for _, key := range []string{"user_id", "agent_id", "run_id"} {
		if v, ok := filters[key]; ok {
			if s, ok := v.(string); ok && s != "" {
				return true
			}
		}
	}
	return false
}

// History 对齐 Memory.history。
func (m *Memory) History(memoryID string) ([]history.HistoryRow, error) {
	return m.DB.GetHistory(memoryID)
}

// UpdateParams 对齐 Memory.update 参数面 (expirationSet 区分显式 null 与未传)。
type UpdateParams struct {
	MemoryID       string
	Text           *string
	Metadata       map[string]any
	ExpirationDate any
	ExpirationSet  bool
}

// Update 对齐 Memory.update + _update_memory。
func (m *Memory) Update(p UpdateParams) (map[string]any, error) {
	text := p.Text
	if text == nil && p.Metadata == nil && !p.ExpirationSet {
		return nil, fmt.Errorf("At least one of text, metadata, or expiration_date must be provided.")
	}
	var updateMetadata map[string]any
	if p.Metadata != nil {
		updateMetadata = map[string]any{}
		for k, v := range p.Metadata {
			updateMetadata[k] = v
		}
	}
	if p.ExpirationSet {
		if updateMetadata == nil {
			updateMetadata = map[string]any{}
		}
		expiration, err := normalizeExpirationDate(p.ExpirationDate)
		if err != nil {
			return nil, err
		}
		if expiration == "" {
			updateMetadata["expiration_date"] = nil
		} else {
			updateMetadata["expiration_date"] = expiration
		}
	}
	existingEmbeddings := map[string][]float64{}
	if text != nil {
		vec, err := m.EmbeddingModel.Embed(*text, "update")
		if err != nil {
			return nil, err
		}
		existingEmbeddings[*text] = vec
	}
	if err := m.updateMemory(p.MemoryID, text, existingEmbeddings, updateMetadata); err != nil {
		return nil, err
	}
	return map[string]any{"message": "Memory updated successfully!"}, nil
}

// updateMemory 对齐 _update_memory。
func (m *Memory) updateMemory(memoryID string, data *string, existingEmbeddings map[string][]float64, metadata map[string]any) error {
	existing, err := m.VectorStore.Get(memoryID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("Memory with id %s not found. Please provide a valid 'memory_id'", memoryID)
	}
	prevValue, _ := existing.Payload["data"].(string)
	if data == nil {
		d := prevValue
		data = &d
	}
	textChanged := *data != prevValue

	newMetadata := map[string]any{}
	for k, v := range existing.Payload {
		newMetadata[k] = v
	}
	if metadata != nil {
		for k, v := range stripIdentityKeys(metadata, existing.Payload, m.warn) {
			newMetadata[k] = v
		}
	}
	newMetadata["data"] = *data
	newMetadata["hash"] = memHash(*data)
	newMetadata["text_lemmatized"] = *data
	if ca, ok := existing.Payload["created_at"]; ok {
		newMetadata["created_at"] = ca
	}
	newMetadata["updated_at"] = nowUTC()

	embedding, ok := existingEmbeddings[*data]
	if !ok {
		embedding, err = m.EmbeddingModel.Embed(*data, "update")
		if err != nil {
			return err
		}
	}
	if err := m.VectorStore.Update(memoryID, embedding, newMetadata); err != nil {
		return err
	}
	var actorID, role *string
	if v, ok := newMetadata["actor_id"].(string); ok {
		actorID = &v
	}
	if v, ok := newMetadata["role"].(string); ok {
		role = &v
	}
	ca, _ := newMetadata["created_at"].(string)
	ua, _ := newMetadata["updated_at"].(string)
	prev := prevValue
	if err := m.DB.AddHistory(history.AddHistoryRecord{
		MemoryID: memoryID, OldMemory: &prev, NewMemory: data, Event: "UPDATE",
		CreatedAt: &ca, UpdatedAt: &ua, ActorID: actorID, Role: role,
	}); err != nil {
		return err
	}
	sessionFilters := map[string]any{}
	for _, k := range []string{"user_id", "agent_id", "run_id"} {
		if v, ok := newMetadata[k].(string); ok && v != "" {
			sessionFilters[k] = v
		}
	}
	if textChanged && m.entityStore != nil {
		store := newEntityStore(m)
		store.RemoveMemoryFromStore(memoryID, sessionFilters)
		m.linkEntitiesForMemory(memoryID, *data, sessionFilters)
	}
	m.emitUpdate(memoryID, *data)
	return nil
}

// newEntityStore 以实体 collection 构建实体包 Store (惰性; 未注入则退主库)。
// 同时传播当前图索引 (如果已构建)。
func newEntityStore(m *Memory) *entity.Store {
	store := entity.NewStore(m.EntityStore(), m.EmbeddingModel)
	if m.graphIndex != nil {
		store.SetGraphIndex(m.graphIndex)
	}
	return store
}

// linkEntitiesForMemory 对齐 _link_entities_for_memory (抽取恒空 → 无操作)。
func (m *Memory) linkEntitiesForMemory(memoryID, text string, filters map[string]any) {
	entities := entity.ExtractEntities(text)
	seen := map[string]bool{}
	for _, e := range entities {
		key := entity.NormalizeText(e.Text)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		store := newEntityStore(m)
		store.Upsert(e.Text, e.Type, memoryID, filters, m.warn)
	}
}

// Delete 对齐 Memory.delete。
func (m *Memory) Delete(memoryID string) (map[string]any, error) {
	existing, err := m.VectorStore.Get(memoryID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("Memory with id %s not found", memoryID)
	}
	if err := m.deleteMemory(memoryID, existing); err != nil {
		return nil, err
	}
	return map[string]any{"message": "Memory deleted successfully!"}, nil
}

// DeleteAllParams 对齐 delete_all。
type DeleteAllParams struct {
	UserID  string
	AgentID string
	RunID   string
}

// DeleteAll 对齐 Memory.delete_all (分批 list + 删除 + 防死循环)。
func (m *Memory) DeleteAll(p DeleteAllParams) (map[string]any, error) {
	filters := map[string]any{}
	for key, val := range map[string]string{"user_id": p.UserID, "agent_id": p.AgentID, "run_id": p.RunID} {
		if val == "" {
			continue
		}
		trimmed, err := validateAndTrimEntityID(val, key)
		if err != nil {
			return nil, err
		}
		filters[key] = trimmed
	}
	if len(filters) == 0 {
		return nil, fmt.Errorf("At least one filter is required to delete all memories. If you want to delete all memories, use the `reset()` method.")
	}
	seenBatches := map[string]bool{}
	for {
		batch, err := m.listRows(filters, 1000)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		batchKey := ""
		for _, row := range batch {
			batchKey += row.ID + ","
		}
		if seenBatches[batchKey] {
			break
		}
		seenBatches[batchKey] = true
		for _, row := range batch {
			if err := m.deleteMemory(row.ID, nil); err != nil {
				return nil, err
			}
		}
	}
	return map[string]any{"message": "Memories deleted successfully!"}, nil
}

// deleteMemory 对齐 _delete_memory。
func (m *Memory) deleteMemory(memoryID string, existing *vectorstore.OutputData) error {
	if existing == nil {
		row, err := m.VectorStore.Get(memoryID)
		if err != nil {
			return err
		}
		if row == nil {
			return fmt.Errorf("Memory with id %s not found. Please provide a valid 'memory_id'", memoryID)
		}
		existing = row
	}
	prevValue, _ := existing.Payload["data"].(string)
	createdAt := normalizeISOTimestampToUTC(strOr(existing.Payload["created_at"]))
	updatedAt := nowUTC()
	payload := existing.Payload
	sessionFilters := map[string]any{}
	for _, k := range []string{"user_id", "agent_id", "run_id"} {
		if v, ok := payload[k].(string); ok && v != "" {
			sessionFilters[k] = v
		}
	}
	if err := m.VectorStore.Delete(memoryID); err != nil {
		return err
	}
	var actorID, role *string
	if v, ok := payload["actor_id"].(string); ok {
		actorID = &v
	}
	if v, ok := payload["role"].(string); ok {
		role = &v
	}
	if err := m.DB.AddHistory(history.AddHistoryRecord{
		MemoryID: memoryID, OldMemory: &prevValue, Event: "DELETE",
		CreatedAt: &createdAt, UpdatedAt: &updatedAt, ActorID: actorID, Role: role, IsDeleted: 1,
	}); err != nil {
		return err
	}
	if m.entityStore != nil {
		newEntityStore(m).RemoveMemoryFromStore(memoryID, sessionFilters)
	}
	m.emitDelete(memoryID, prevValue)
	return nil
}

// Reset 对齐 Memory.reset。
func (m *Memory) Reset() error {
	if err := m.DB.Reset(); err != nil {
		return err
	}
	if err := m.DB.Close(); err != nil {
		return err
	}
	db, err := history.NewManager(m.Config.HistoryDBPath)
	if err != nil {
		return err
	}
	m.DB = db
	if err := m.VectorStore.Reset(); err != nil {
		return err
	}
	if m.entityStore != nil {
		if err := m.entityStore.Reset(); err != nil {
			m.entityStore = nil
		}
	}
	return nil
}

func strOr(v any) string {
	s, _ := v.(string)
	return s
}

// DeleteAllParamsByField 按单字段构造 (entity 删除路径用)。
func DeleteAllParamsByField(field, value string) DeleteAllParams {
	switch field {
	case "user_id":
		return DeleteAllParams{UserID: value}
	case "agent_id":
		return DeleteAllParams{AgentID: value}
	default:
		return DeleteAllParams{RunID: value}
	}
}
