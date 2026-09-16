package memory

// 分类打标: add 抽取出的新记忆按生效分类目录打最近分类标。
// 目录解析: per-call 非空整体替换 > config 级 > 未配置 (功能关闭)。
// 打标在 ingest 阶段 (抽取+去重后、持久化前) 单次 LLM 调用完成; 改目录不影响存量记忆。

import (
	"encoding/json"
	"fmt"

	"github.com/zhao-core/memgo/core/config"
	"github.com/zhao-core/memgo/core/llm"
	"github.com/zhao-core/memgo/core/prompts"
)

// resolveCategories 解析生效分类目录: per-call 非空整体替换, 否则 config 级, 都空 = 功能关闭。
func resolveCategories(perCall, configLevel []config.Category) []config.Category {
	if len(perCall) > 0 {
		return perCall
	}
	return configLevel
}

// classifyCategories 批量打标 (单次 LLM 调用), 返回与 texts 等长的分类名。
// LLM 调用失败或输出不可解析/缺项/未知分类 → 显式报错, 不静默回退。
func (m *Memory) classifyCategories(texts []string, categories []config.Category) ([]string, error) {
	userPrompt := prompts.GenerateCategoryClassificationPrompt(prompts.GenerateCategoryClassificationParams{
		Memories:   texts,
		Categories: categories,
	})
	response, err := m.LLM.GenerateResponse([]llm.Message{
		{Role: "system", Content: prompts.CATEGORY_CLASSIFICATION_SYSTEM_PROMPT},
		{Role: "user", Content: userPrompt},
	}, llm.GenerateOptions{ResponseFormat: llm.JSONFormat()})
	if err != nil {
		return nil, fmt.Errorf("LLM category classification failed: %w", err)
	}
	return parseCategoryAssignments(response, texts, categories)
}

// parseCategoryAssignments 解析 LLM 分类输出并按记忆序回填, 严格校验 (缺项/越界/重复/未知分类报错)。
func parseCategoryAssignments(response string, texts []string, categories []config.Category) ([]string, error) {
	valid := make(map[string]bool, len(categories))
	for _, c := range categories {
		valid[c.Name] = true
	}
	var parsed struct {
		Categories []struct {
			ID       float64 `json:"id"`
			Category string  `json:"category"`
		} `json:"categories"`
	}
	cleaned := removeCodeBlocks(response)
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, fmt.Errorf("LLM category classification returned unparseable output: %s", response)
	}
	out := make([]string, len(texts))
	filled := make([]bool, len(texts))
	for _, item := range parsed.Categories {
		if item.ID < 1 || item.ID > float64(len(texts)) {
			return nil, fmt.Errorf("LLM category classification returned out-of-range memory id %v (expected 1..%d)", item.ID, len(texts))
		}
		if !valid[item.Category] {
			return nil, fmt.Errorf("LLM category classification returned unknown category %q for memory %v", item.Category, item.ID)
		}
		idx := int(item.ID) - 1
		if filled[idx] {
			return nil, fmt.Errorf("LLM category classification returned duplicate entry for memory %v", item.ID)
		}
		out[idx] = item.Category
		filled[idx] = true
	}
	for i, ok := range filled {
		if !ok {
			return nil, fmt.Errorf("LLM category classification missing entry for memory %d", i+1)
		}
	}
	return out, nil
}
