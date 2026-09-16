package memory

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/zhao-core/memgo/core/config"
	"github.com/zhao-core/memgo/core/entity"
	"github.com/zhao-core/memgo/core/history"
	"github.com/zhao-core/memgo/core/llm"
	"github.com/zhao-core/memgo/core/prompts"
	"github.com/zhao-core/memgo/core/vectorstore"
)

// AddParams 对齐 Memory.add 的关键字参数面 (server 端点逐一映射)。
type AddParams struct {
	UserID         string
	AgentID        string
	RunID          string
	Metadata       map[string]any
	ExpirationDate any    // string 或 nil
	Infer          bool   // 默认 true 由调用方填
	MemoryType     string // 仅 "procedural_memory" 特判
	Prompt         string
	// CustomCategories per-call 分类目录 (非空时整体替换 config 级, 不合并 —— memgo 扩展)。
	CustomCategories []config.Category
}

// AddResultItem 是 add 返回 results 数组的元素。
type AddResultItem = map[string]any

// Add 对齐 Memory.add: 校验 → procedural 分支 / infer 双分支。
func (m *Memory) Add(messages []map[string]any, p AddParams) ([]AddResultItem, error) {
	expiration, err := normalizeExpirationDate(p.ExpirationDate)
	if err != nil {
		return nil, err
	}
	metadata, filters, err := buildFiltersAndMetadata(p.UserID, p.AgentID, p.RunID, p.Metadata, nil, m.warn)
	if err != nil {
		return nil, err
	}
	if expiration != "" {
		metadata["expiration_date"] = expiration
	}
	if p.MemoryType != "" && p.MemoryType != "procedural_memory" {
		return nil, fmt.Errorf("Invalid 'memory_type'. Please pass procedural_memory to create procedural memories.")
	}
	if p.AgentID != "" && p.MemoryType == "procedural_memory" {
		result, err := m.createProceduralMemory(messages, metadata, p.Prompt)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	messages = parseVisionMessages(messages)
	// 分类目录解析 (memgo 扩展): per-call 整体替换 config 级; 都空 = 功能关闭。
	activeCategories := resolveCategories(p.CustomCategories, m.CustomCategories)
	return m.addToVectorStore(messages, metadata, filters, p.Infer, p.Prompt, activeCategories)
}

func (m *Memory) warn(msg string) {}

// parseVisionMessages 对齐 parse_vision_messages 的无 LLM 分支 (enable_vision=false 默认):
// system 原样; content 为数组时拼接 text 分段; 其余原样; 无 content 跳过。
func parseVisionMessages(messages []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		role, _ := msg["role"].(string)
		content, hasContent := msg["content"]
		if role == "system" {
			out = append(out, msg)
			continue
		}
		if !hasContent || content == nil {
			continue
		}
		if parts, ok := content.([]any); ok {
			var texts []string
			for _, part := range parts {
				if pm, ok := part.(map[string]any); ok && pm["type"] == "text" {
					if t, ok := pm["text"].(string); ok {
						texts = append(texts, t)
					}
				}
			}
			if len(texts) == 0 {
				continue
			}
			out = append(out, map[string]any{"role": role, "content": strings.Join(texts, " ")})
			continue
		}
		out = append(out, msg)
	}
	return out
}

// addToVectorStore 对齐 _add_to_vector_store: infer=false 直存分支 + infer=true 阶段化流水线。
func (m *Memory) addToVectorStore(messages []map[string]any, metadata, filters map[string]any, infer bool, prompt string, activeCategories []config.Category) ([]AddResultItem, error) {
	if !infer {
		var returned []AddResultItem
		for _, msg := range messages {
			role, _ := msg["role"].(string)
			content, ok := msg["content"].(string)
			if !ok || role == "" || msg["content"] == nil {
				continue
			}
			if role == "system" {
				continue
			}
			perMsgMeta := map[string]any{}
			for k, v := range metadata {
				perMsgMeta[k] = v
			}
			perMsgMeta["role"] = role
			actorName, hasActor := msg["name"].(string)
			if hasActor && actorName != "" {
				perMsgMeta["actor_id"] = actorName
			}
			vec, err := m.EmbeddingModel.Embed(content, "add")
			if err != nil {
				return nil, err
			}
			memID, err := m.createMemory(content, vec, perMsgMeta)
			if err != nil {
				return nil, err
			}
			var actor any
			if hasActor && actorName != "" {
				actor = actorName
			}
			returned = append(returned, map[string]any{
				"id": memID, "memory": content, "event": "ADD", "actor_id": actor, "role": role,
			})
		}
		return returned, nil
	}
	return m.addInferred(messages, metadata, filters, prompt, activeCategories)
}

// addInferred 对齐 _add_to_vector_store 的 V3 阶段化批量流水线 (Phase 0-8)。
func (m *Memory) addInferred(messages []map[string]any, metadata, filters map[string]any, prompt string, activeCategories []config.Category) ([]AddResultItem, error) {
	sessionScope := buildSessionScope(filters)
	parsedMessages := parseMessages(messages)

	searchFilters := scopeFiltersOnly(filters)
	queryEmbedding, err := m.EmbeddingModel.Embed(parsedMessages, "search")
	if err != nil {
		return nil, err
	}
	existingResults, err := m.VectorStore.Search(parsedMessages, queryEmbedding, 10, searchFilters)
	if err != nil {
		return nil, err
	}

	// Phase 2: 单次 LLM 抽取 (UUID→整数映射防幻觉)
	existingMemories := make([]map[string]any, 0, len(existingResults))
	for idx, mem := range existingResults {
		text, _ := mem.Payload["data"].(string)
		existingMemories = append(existingMemories, map[string]any{"id": fmt.Sprintf("%d", idx), "text": text})
	}
	isAgentScoped := false
	if a, ok := filters["agent_id"].(string); ok && a != "" {
		if u, ok := filters["user_id"].(string); !ok || u == "" {
			isAgentScoped = true
		}
	}
	systemPrompt := prompts.ADDITIVE_EXTRACTION_PROMPT
	if isAgentScoped {
		systemPrompt += prompts.AGENT_CONTEXT_SUFFIX
	}
	customInstr := prompt
	if customInstr == "" {
		customInstr = m.CustomInstructions
	}
	lastMessages, err := m.DB.GetLastMessages(sessionScope, 10)
	if err != nil {
		return nil, err
	}
	lastK := messageRowsToMaps(lastMessages)
	userPrompt := prompts.GenerateAdditiveExtractionPrompt(prompts.GenerateAdditiveExtractionParams{
		ExistingMemories:   existingMemories,
		NewMessages:        parsedMessages,
		LastKMessages:      lastK,
		CustomInstructions: customInstr,
	})
	response, err := m.LLM.GenerateResponse([]llm.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, llm.GenerateOptions{ResponseFormat: llm.JSONFormat()})
	if err != nil {
		return nil, fmt.Errorf("LLM extraction failed: %w", err)
	}
	extractedMemories := parseExtractionResponse(response)
	if len(extractedMemories) == 0 {
		if err := m.DB.SaveMessages(messages, sessionScope); err != nil {
			return nil, err
		}
		return []AddResultItem{}, nil
	}
	return m.persistExtracted(extractedMemories, existingResults, metadata, searchFilters, messages, sessionScope, activeCategories)
}

// parseMessages 对齐 memory/utils.parse_messages: "role: content\n" 拼接, 跳过无 content。
func parseMessages(messages []map[string]any) string {
	var b strings.Builder
	for _, msg := range messages {
		role, _ := msg["role"].(string)
		content, has := msg["content"]
		if !has || content == nil {
			continue
		}
		c, _ := content.(string)
		switch role {
		case "system":
			b.WriteString("system: " + c + "\n")
		case "user":
			b.WriteString("user: " + c + "\n")
		case "assistant":
			b.WriteString("assistant: " + c + "\n")
		}
	}
	return b.String()
}

// messageRowsToMaps 将 history 消息行转为 prompt 用的 dict (role/message/content 键兼容)。
func messageRowsToMaps(rows []history.MessageRow) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		m := map[string]any{"role": r.Role, "content": ""}
		if r.Content != nil {
			m["content"] = *r.Content
		}
		out = append(out, m)
	}
	return out
}

// parseExtractionResponse 对齐 Phase 2 解析: remove_code_blocks → json.loads → extract_json 回退;
// 全失败返回空 (错误仅日志)。
func parseExtractionResponse(response string) []map[string]any {
	cleaned := removeCodeBlocks(response)
	if strings.TrimSpace(cleaned) == "" {
		return nil
	}
	if mem := loadsMemoryKey(cleaned); mem != nil {
		return mem
	}
	if mem := loadsMemoryKey(extractJSON(cleaned)); mem != nil {
		return mem
	}
	return nil
}

// loadsMemoryKey 尝试 json 解析并取 "memory" 键 (strict=False 近似: 宽松反序列化)。
func loadsMemoryKey(text string) []map[string]any {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil
	}
	raw, ok := parsed["memory"].([]any)
	if !ok {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// removeCodeBlocks 对齐 memory/utils.remove_code_blocks。
func removeCodeBlocks(content string) string {
	trimmed := strings.TrimSpace(content)
	re := regexp.MustCompile(`^\x60\x60\x60[a-zA-Z0-9]*\n([\s\S]*?)\n\x60\x60\x60$`)
	if m := re.FindStringSubmatch(trimmed); m != nil {
		trimmed = strings.TrimSpace(m[1])
	}
	think := regexp.MustCompile(`(?s)<think>.*?</think>`)
	return strings.TrimSpace(think.ReplaceAllString(trimmed, ""))
}

// extractJSON 对齐 memory/utils.extract_json: 代码块优先, 否则首 "{" 至末 "}"。
func extractJSON(text string) string {
	text = strings.TrimSpace(text)
	re := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")
	if m := re.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start != -1 && end != -1 && end > start {
		return text[start : end+1]
	}
	return text
}

// memoryRecord 是流水线中间记录。
type memoryRecord struct {
	id   string
	text string
	vec  []float64
	meta map[string]any
}

// persistExtracted 对齐 Phase 3-8: 批量嵌入 → hash 去重 → 插入 → 历史 → 实体联动 → 存消息。
func (m *Memory) persistExtracted(extractedMemories []map[string]any, existingResults []vectorstore.OutputData, metadata, searchFilters map[string]any, messages []map[string]any, sessionScope string, activeCategories []config.Category) ([]AddResultItem, error) {
	// Phase 3: 批量嵌入
	var memTexts []string
	for _, mm := range extractedMemories {
		if t, ok := mm["text"].(string); ok && t != "" {
			memTexts = append(memTexts, t)
		}
	}
	embedMap := map[string][]float64{}
	if len(memTexts) > 0 {
		vecs, err := m.EmbeddingModel.EmbedBatch(memTexts, "add")
		if err == nil {
			for i, t := range memTexts {
				embedMap[t] = vecs[i]
			}
		} else {
			for _, t := range memTexts {
				if v, err := m.EmbeddingModel.Embed(t, "add"); err == nil {
					embedMap[t] = v
				}
			}
		}
	}
	// Phase 4+5: hash 去重 (批内 + 既有)
	existingHashes := map[string]bool{}
	for _, mem := range existingResults {
		if h, ok := mem.Payload["hash"].(string); ok && h != "" {
			existingHashes[h] = true
		}
	}
	var records []memoryRecord
	seenHashes := map[string]bool{}
	for _, mm := range extractedMemories {
		text, ok := mm["text"].(string)
		if !ok || text == "" {
			continue
		}
		if _, ok := embedMap[text]; !ok {
			continue
		}
		h := memHash(text)
		if existingHashes[h] || seenHashes[h] {
			continue
		}
		seenHashes[h] = true
		memID := newUUID()
		memMeta := map[string]any{}
		for k, v := range metadata {
			memMeta[k] = v
		}
		memMeta["data"] = text
		memMeta["text_lemmatized"] = text
		memMeta["hash"] = h
		if _, ok := memMeta["created_at"]; !ok {
			memMeta["created_at"] = nowUTC()
		}
		memMeta["updated_at"] = memMeta["created_at"]
		if at, ok := mm["attributed_to"]; ok {
			memMeta["attributed_to"] = at
		}
		records = append(records, memoryRecord{id: memID, text: text, vec: embedMap[text], meta: memMeta})
	}
	if len(records) == 0 {
		if err := m.DB.SaveMessages(messages, sessionScope); err != nil {
			return nil, err
		}
		return []AddResultItem{}, nil
	}
	// Phase 5.5 (memgo 扩展): 分类打标 — 生效目录非空时对新记忆单次 LLM 打标, 失败显式报错。
	if len(activeCategories) > 0 {
		texts := make([]string, 0, len(records))
		for _, r := range records {
			texts = append(texts, r.text)
		}
		assigned, err := m.classifyCategories(texts, activeCategories)
		if err != nil {
			// LLM 输出不可解析/缺项/未知分类 → 整个 add 显式失败, 不静默回退。
			return nil, err
		}
		for i := range records {
			records[i].meta["category"] = assigned[i]
		}
	}
	return m.persistRecords(records, searchFilters, messages, sessionScope)
}

// persistRecords 对齐 Phase 6-8: 批量插入 + 历史 + 实体联动 + 存消息 + 返回结果。
func (m *Memory) persistRecords(records []memoryRecord, searchFilters map[string]any, messages []map[string]any, sessionScope string) ([]AddResultItem, error) {
	vectors := make([][]float64, 0, len(records))
	ids := make([]string, 0, len(records))
	payloads := make([]map[string]any, 0, len(records))
	for _, r := range records {
		vectors = append(vectors, r.vec)
		ids = append(ids, r.id)
		payloads = append(payloads, r.meta)
	}
	if err := m.VectorStore.Insert(vectors, ids, payloads); err != nil {
		for _, r := range records {
			_ = m.VectorStore.Insert([][]float64{r.vec}, []string{r.id}, []map[string]any{r.meta})
		}
	}
	historyRecords := make([]history.AddHistoryRecord, 0, len(records))
	for _, r := range records {
		ca, _ := r.meta["created_at"].(string)
		historyRecords = append(historyRecords, history.AddHistoryRecord{
			MemoryID: r.id, NewMemory: &r.text, Event: "ADD", CreatedAt: &ca, IsDeleted: 0,
		})
	}
	if err := m.DB.BatchAddHistory(historyRecords); err != nil {
		for _, hr := range historyRecords {
			_ = m.DB.AddHistory(hr)
		}
	}
	// Phase 7: 实体联动 (无 spaCy 基线恒空 → 空操作, 逻辑保留以对齐行为面)
	if m.entityStore != nil {
		m.linkEntitiesBatch(records, searchFilters)
	}
	if err := m.DB.SaveMessages(messages, sessionScope); err != nil {
		return nil, err
	}
	returned := make([]AddResultItem, 0, len(records))
	for _, r := range records {
		item := map[string]any{"id": r.id, "memory": r.text, "event": "ADD"}
		// 分类目录生效时 (memgo 扩展) 返回打标结果; 功能关闭与 Python 基线形状一致。
		if c, ok := r.meta["category"]; ok {
			item["category"] = c
		}
		returned = append(returned, item)
	}
	return returned, nil
}

// linkEntitiesBatch 对齐 Phase 7 (批量实体 upsert; 抽取恒空时无操作)。
func (m *Memory) linkEntitiesBatch(records []memoryRecord, searchFilters map[string]any) {
	var allTexts []string
	for _, r := range records {
		allTexts = append(allTexts, r.text)
	}
	allEntities := entity.ExtractEntitiesBatch(allTexts)
	for idx, r := range records {
		if idx >= len(allEntities) {
			break
		}
		for _, e := range allEntities[idx] {
			m.entityUpsert(e.Text, e.Type, r.id, searchFilters)
		}
	}
}

// entityUpsert 委托实体包 (精简路径, 单条)。
func (m *Memory) entityUpsert(entityText, entityType, memoryID string, filters map[string]any) {
	store := entity.NewStore(m.EntityStore(), m.EmbeddingModel)
	store.Upsert(entityText, entityType, memoryID, filters, m.warn)
}

// createProceduralMemory 对齐 _create_procedural_memory。
func (m *Memory) createProceduralMemory(messages []map[string]any, metadata map[string]any, prompt string) ([]AddResultItem, error) {
	system := prompt
	if system == "" {
		system = prompts.PROCEDURAL_MEMORY_SYSTEM_PROMPT
	}
	parsed := []map[string]any{{"role": "system", "content": system}}
	parsed = append(parsed, messages...)
	parsed = append(parsed, map[string]any{"role": "user", "content": "Create procedural memory of the above conversation."})
	llmMessages := make([]llm.Message, 0, len(parsed))
	for _, msg := range parsed {
		role, _ := msg["role"].(string)
		llmMessages = append(llmMessages, llm.Message{Role: role, Content: msg["content"]})
	}
	response, err := m.LLM.GenerateResponse(llmMessages, llm.GenerateOptions{})
	if err != nil {
		return nil, err
	}
	procedural := removeCodeBlocks(response)
	if procedural == "" {
		return nil, fmt.Errorf("The LLM returned no content for the procedural memory summary. The model may have declined the request or returned an empty response.")
	}
	if metadata == nil {
		return nil, fmt.Errorf("Metadata cannot be done for procedural memory.")
	}
	meta := map[string]any{}
	for k, v := range metadata {
		meta[k] = v
	}
	meta["memory_type"] = "procedural_memory"
	vec, err := m.EmbeddingModel.Embed(procedural, "add")
	if err != nil {
		return nil, err
	}
	memID, err := m.createMemory(procedural, vec, meta)
	if err != nil {
		return nil, err
	}
	return []AddResultItem{{"id": memID, "memory": procedural, "event": "ADD"}}, nil
}

// createMemory 对齐 _create_memory: 插入向量 + ADD 历史。
func (m *Memory) createMemory(data string, embedding []float64, metadata map[string]any) (string, error) {
	memID := newUUID()
	newMeta := map[string]any{}
	for k, v := range metadata {
		newMeta[k] = v
	}
	newMeta["data"] = data
	newMeta["hash"] = memHash(data)
	if _, ok := newMeta["created_at"]; !ok {
		newMeta["created_at"] = nowUTC()
	}
	newMeta["updated_at"] = newMeta["created_at"]
	newMeta["text_lemmatized"] = data
	if err := m.VectorStore.Insert([][]float64{embedding}, []string{memID}, []map[string]any{newMeta}); err != nil {
		return "", err
	}
	var actorID, role *string
	if v, ok := newMeta["actor_id"].(string); ok {
		actorID = &v
	}
	if v, ok := newMeta["role"].(string); ok {
		role = &v
	}
	ca, _ := newMeta["created_at"].(string)
	ua, _ := newMeta["updated_at"].(string)
	if err := m.DB.AddHistory(history.AddHistoryRecord{
		MemoryID: memID, NewMemory: &data, Event: "ADD", CreatedAt: &ca, UpdatedAt: &ua,
		ActorID: actorID, Role: role,
	}); err != nil {
		return "", err
	}
	return memID, nil
}
