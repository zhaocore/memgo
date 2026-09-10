// Package entity: 实体抽取与实体存储联动。
// 抽取对齐上游 memgo/utils/entity_extraction.py 的**无 spaCy 回退路径**:
// Python get_nlp_full() 返回 None 时 extract_entities 恒返回 []。
// 契约基线容器未装 spaCy (已实测), 故 Go 恒空即与基线逐字节对齐;
// NER 等价物属后续增强, 接入会改变 search 实体加成行为, 须连带更新契约基线。
package entity

// Extracted 是 (entity_type, entity_text) 二元组。
type Extracted struct {
	Type string
	Text string
}

// ExtractEntities 对齐 extract_entities 的无 spaCy 分支: 恒空。
func ExtractEntities(text string) []Extracted {
	return nil
}

// ExtractEntitiesBatch 对齐 extract_entities_batch 的无 spaCy 分支: 恒空组。
func ExtractEntitiesBatch(texts []string) [][]Extracted {
	out := make([][]Extracted, len(texts))
	for i := range texts {
		out[i] = []Extracted{}
	}
	return out
}
