package memory

import (
	"fmt"
	"sort"
	"sync"

	"github.com/zhao-core/memgo/core/llm"
	"github.com/zhao-core/memgo/core/vectorstore"
)

// fakeEmbedder: 恒定向量 (契约 stub 同构: 任意文本同向量 → 余弦距离 0 → score 1.0)。
type fakeEmbedder struct{ dims int }

func (f *fakeEmbedder) Embed(text string, action string) ([]float64, error) {
	vecs, err := f.EmbedBatch([]string{text}, action)
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (f *fakeEmbedder) EmbedBatch(texts []string, action string) ([][]float64, error) {
	out := make([][]float64, 0, len(texts))
	for range texts {
		v := make([]float64, f.dims)
		for i := range v {
			v[i] = 0.1
		}
		out = append(out, v)
	}
	return out, nil
}

// fakeVectorStore: 内存实现 (行为面与 pgvector 对齐: 余弦距离 + payload 过滤)。
type fakeVectorStore struct {
	mu   sync.Mutex
	rows map[string]map[string]any
}

func newFakeVectorStore() *fakeVectorStore {
	return &fakeVectorStore{rows: map[string]map[string]any{}}
}

func cosineDist(a, b []float64) float64 {
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	return 1 - dot/(na*nb)
}

func (f *fakeVectorStore) Insert(vectors [][]float64, ids []string, payloads []map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range ids {
		f.rows[ids[i]] = payloads[i]
	}
	return nil
}

func (f *fakeVectorStore) Search(query string, vec []float64, topK int, filters map[string]any) ([]vectorstore.OutputData, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	type scored struct {
		id      string
		dist    float64
		payload map[string]any
	}
	var all []scored
	for id, payload := range f.rows {
		if !matchFilters(payload, filters) {
			continue
		}
		all = append(all, scored{id: id, dist: 0, payload: payload})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].id < all[j].id })
	var out []vectorstore.OutputData
	for _, s := range all {
		if len(out) >= topK {
			break
		}
		score := 1.0 - s.dist
		out = append(out, vectorstore.OutputData{ID: s.id, Score: &score, Payload: s.payload})
	}
	return out, nil
}

func matchFilters(payload map[string]any, filters map[string]any) bool {
	for k, v := range filters {
		pv, ok := payload[k]
		if !ok || fmt.Sprintf("%v", pv) != fmt.Sprintf("%v", v) {
			return false
		}
	}
	return true
}

func (f *fakeVectorStore) KeywordSearch(query string, topK int, filters map[string]any) ([]vectorstore.OutputData, error) {
	return nil, nil
}

func (f *fakeVectorStore) Get(id string) (*vectorstore.OutputData, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.rows[id]
	if !ok {
		return nil, nil
	}
	return &vectorstore.OutputData{ID: id, Payload: p}, nil
}

func (f *fakeVectorStore) Delete(id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.rows, id)
	return nil
}

func (f *fakeVectorStore) Update(id string, vec []float64, payload map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.rows[id]
	if !ok {
		return fmt.Errorf("missing: %s", id)
	}
	if payload != nil {
		for k, v := range payload {
			p[k] = v
		}
	}
	return nil
}

func (f *fakeVectorStore) List(filters map[string]any, topK int) ([][]vectorstore.OutputData, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	ids := make([]string, 0, len(f.rows))
	for id := range f.rows {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var out []vectorstore.OutputData
	for _, id := range ids {
		if matchFilters(f.rows[id], filters) {
			out = append(out, vectorstore.OutputData{ID: id, Payload: f.rows[id]})
		}
		if len(out) >= topK {
			break
		}
	}
	return [][]vectorstore.OutputData{out}, nil
}

func (f *fakeVectorStore) ListCols() ([]string, error) { return nil, nil }
func (f *fakeVectorStore) DeleteCol() error            { return nil }
func (f *fakeVectorStore) CreateCol() error            { return nil }
func (f *fakeVectorStore) Reset() error                { f.rows = map[string]map[string]any{}; return nil }
func (f *fakeVectorStore) Close()                      {}

// fakeLLM: 固定抽取响应 (契约 stub 同构)。
// queue 非空时按序弹出 (第二次调用 = 分类打标), calls 记录各次调用消息供断言。
type fakeLLM struct {
	response string
	queue    []string
	calls    [][]llm.Message
}

func (f *fakeLLM) GenerateResponse(messages []llm.Message, opts llm.GenerateOptions) (string, error) {
	f.calls = append(f.calls, append([]llm.Message(nil), messages...))
	if len(f.queue) > 0 {
		r := f.queue[0]
		f.queue = f.queue[1:]
		return r, nil
	}
	return f.response, nil
}
