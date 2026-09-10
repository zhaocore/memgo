package entity

import "testing"

// TestExtractEntitiesFallbackParity: 无 spaCy 基线下抽取恒空 —
// 与契约基线 (entity_boost 恒 0) 一致; NER 接入会改变行为, 须连带更新契约基线。
func TestExtractEntitiesFallbackParity(t *testing.T) {
	if got := ExtractEntities("Alice went to Berlin with Bob."); got != nil {
		t.Errorf("fallback 必须恒空: %v", got)
	}
	batch := ExtractEntitiesBatch([]string{"a", "b"})
	if len(batch) != 2 || batch[0] == nil {
		t.Errorf("batch fallback 应为空组: %v", batch)
	}
}

// TestNormalizeText: 实体归一化规则 (strip/lower/空白折叠)。
func TestNormalizeText(t *testing.T) {
	if NormalizeText("  Alice   Smith ") != "alice smith" {
		t.Errorf("归一化不符: %q", NormalizeText("  Alice   Smith "))
	}
	if EntityCollectionName("pgvector", "memories") != "memories_entities" {
		t.Errorf("实体 collection 命名规则不符")
	}
}
