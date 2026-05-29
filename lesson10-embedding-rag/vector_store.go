package main

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

type vectorEntry struct {
	doc    *schema.Document
	vector []float64
}

// vectorStore：索引时用 Embedder 向量化，检索时按余弦相似度 TopK。
type vectorStore struct {
	emb  embedding.Embedder
	mu   sync.RWMutex
	list []vectorEntry
}

func newVectorStore(emb embedding.Embedder) *vectorStore {
	return &vectorStore{emb: emb}
}

func (s *vectorStore) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	_ = opts
	valid := make([]*schema.Document, 0, len(docs))
	for _, d := range docs {
		if d != nil {
			valid = append(valid, d)
		}
	}
	if len(valid) == 0 {
		return nil, nil
	}
	texts := make([]string, len(valid))
	ids := make([]string, len(valid))
	for i, d := range valid {
		id := d.ID
		if id == "" {
			id = docID(d.Content)
		}
		ids[i] = id
		texts[i] = docTitle(d) + "\n" + d.Content
	}
	fmt.Printf("正在调用 Embedding API 向量化 %d 个片段…\n", len(texts))
	vecs, err := s.emb.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("embed documents: %w", err)
	}
	if len(vecs) != len(valid) {
		return nil, fmt.Errorf("embed count mismatch: got %d want %d", len(vecs), len(valid))
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.list = make([]vectorEntry, 0, len(valid))
	for i, d := range valid {
		cp := *d
		cp.ID = ids[i]
		s.list = append(s.list, vectorEntry{doc: &cp, vector: vecs[i]})
	}
	return ids, nil
}

func (s *vectorStore) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	options := retriever.GetCommonOptions(nil, opts...)
	topK := 3
	if options.TopK != nil {
		topK = *options.TopK
	}

	s.mu.RLock()
	n := len(s.list)
	s.mu.RUnlock()
	if n == 0 {
		return nil, fmt.Errorf("向量库为空，请先运行: go run . -demo index")
	}

	vecs, err := s.emb.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	qv := vecs[0]

	s.mu.RLock()
	defer s.mu.RUnlock()

	type scored struct {
		doc   *schema.Document
		score float64
	}
	ranked := make([]scored, 0, len(s.list))
	for _, e := range s.list {
		score := cosineSimilarity(qv, e.vector)
		cp := *e.doc
		cp.WithScore(score)
		ranked = append(ranked, scored{doc: &cp, score: score})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	if len(ranked) > topK {
		ranked = ranked[:topK]
	}
	out := make([]*schema.Document, len(ranked))
	for i := range ranked {
		out[i] = ranked[i].doc
	}
	return out, nil
}

func (s *vectorStore) docCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.list)
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}
