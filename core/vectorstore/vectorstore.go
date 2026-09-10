// Package vectorstore: VectorStore 接口与 pgvector 实现。
// 对齐上游 mem0/vector_stores/base.py 接口面 + mem0/vector_stores/pgvector.py 行为
// (余弦距离 <=>, score=max(0,1-d), payload JSONB 过滤, list 返回嵌套一层)。
package vectorstore

// OutputData 对齐 Python pgvector.OutputData。
type OutputData struct {
	ID      string
	Score   *float64
	Payload map[string]any
}

// VectorStore 对齐 Python VectorStoreBase 实际被 Memory 使用的方法面。
// List 返回嵌套一层切片, 复刻 pgvector.py 的 [[OutputData]] 返回形状。
type VectorStore interface {
	Insert(vectors [][]float64, ids []string, payloads []map[string]any) error
	Search(query string, vectors []float64, topK int, filters map[string]any) ([]OutputData, error)
	KeywordSearch(query string, topK int, filters map[string]any) ([]OutputData, error)
	Get(vectorID string) (*OutputData, error)
	Delete(vectorID string) error
	Update(vectorID string, vector []float64, payload map[string]any) error
	List(filters map[string]any, topK int) ([][]OutputData, error)
	ListCols() ([]string, error)
	DeleteCol() error
	CreateCol() error
	Reset() error
	Close()
}
