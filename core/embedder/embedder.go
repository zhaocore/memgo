// Package embedder: Embedder 接口与内置 provider (openai/gemini)。
// 对齐上游 memgo/embeddings/base.py (embed + embed_batch)。
package embedder

// Embedder 对齐 Python EmbeddingBase。
// memoryAction 为 "add"/"search"/"update" (provider 可据此区分, 当前内置实现忽略)。
type Embedder interface {
	Embed(text string, memoryAction string) ([]float64, error)
	EmbedBatch(texts []string, memoryAction string) ([][]float64, error)
}
