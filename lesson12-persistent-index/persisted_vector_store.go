package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

type persistedIndex struct {
	// Version 用于未来升级格式
	Version int `json:"version"`
	// DocIDs 是按 entries 顺序的文档 ID
	DocIDs []string `json:"doc_ids"`
	// Titles / Contents 用于打印和拼 context（真实项目可只存 ID+chunk 指针）
	Titles   []string    `json:"titles"`
	Contents []string    `json:"contents"`
	Vectors  [][]float64 `json:"vectors"`
}

// persistedVectorStore：向量索引落盘（data/index.json），下次启动直接加载，避免重复向量化。
type persistedVectorStore struct {
	emb embedding.Embedder

	mu   sync.RWMutex
	path string
	idx  *persistedIndex
}

func newPersistedVectorStore(emb embedding.Embedder, path string) *persistedVectorStore {
	return &persistedVectorStore{emb: emb, path: path}
}

func (s *persistedVectorStore) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
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
	titles := make([]string, len(valid))
	contents := make([]string, len(valid))
	for i, d := range valid {
		ids[i] = d.ID
		if ids[i] == "" {
			ids[i] = docID(d.Content)
		}
		titles[i] = docTitle(d)
		contents[i] = d.Content
		texts[i] = titles[i] + "\n" + contents[i]
	}

	fmt.Printf("正在调用 Embedding API 向量化 %d 个片段（仅首次或知识库变更时发生）…\n", len(texts))
	vecs, err := s.emb.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("embed documents: %w", err)
	}
	if len(vecs) != len(valid) {
		return nil, fmt.Errorf("embed count mismatch: got %d want %d", len(vecs), len(valid))
	}

	idx := &persistedIndex{
		Version:  1,
		DocIDs:   ids,
		Titles:   titles,
		Contents: contents,
		Vectors:  vecs,
	}

	if err := s.save(idx); err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.idx = idx
	s.mu.Unlock()
	return ids, nil
}

func (s *persistedVectorStore) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	_ = ctx
	options := retriever.GetCommonOptions(nil, opts...)
	topK := 3
	if options.TopK != nil {
		topK = *options.TopK
	}

	idx, err := s.ensureLoaded()
	if err != nil {
		return nil, err
	}
	if idx == nil || len(idx.Vectors) == 0 {
		return nil, fmt.Errorf("index is empty, please run: go run . -demo build")
	}

	qv, err := s.emb.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(qv) != 1 {
		return nil, fmt.Errorf("embed query unexpected count: %d", len(qv))
	}

	type scored struct {
		i     int
		score float64
	}
	ranked := make([]scored, 0, len(idx.Vectors))
	for i := range idx.Vectors {
		ranked = append(ranked, scored{i: i, score: cosineSimilarity(qv[0], idx.Vectors[i])})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	if len(ranked) > topK {
		ranked = ranked[:topK]
	}

	out := make([]*schema.Document, 0, len(ranked))
	for _, r := range ranked {
		d := &schema.Document{
			ID:      idx.DocIDs[r.i],
			Content: idx.Contents[r.i],
			MetaData: map[string]any{
				"title":  idx.Titles[r.i],
				"source": "vector(persisted)",
			},
		}
		d.WithScore(r.score)
		out = append(out, d)
	}
	return out, nil
}

func (s *persistedVectorStore) docCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.idx == nil {
		return 0
	}
	return len(s.idx.DocIDs)
}

func (s *persistedVectorStore) ensureLoaded() (*persistedIndex, error) {
	s.mu.RLock()
	if s.idx != nil {
		defer s.mu.RUnlock()
		return s.idx, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.idx != nil {
		return s.idx, nil
	}
	idx, err := s.load()
	if err != nil {
		return nil, err
	}
	s.idx = idx
	return s.idx, nil
}

func (s *persistedVectorStore) save(idx *persistedIndex) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	b, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *persistedVectorStore) load() (*persistedIndex, error) {
	b, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var idx persistedIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
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

