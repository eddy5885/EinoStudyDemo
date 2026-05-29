package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

// memKnowledgeStore 内存 Indexer + Retriever（关键词重叠打分，无需 Embedding API）。
type memKnowledgeStore struct {
	mu   sync.RWMutex
	docs map[string]*schema.Document
}

func newMemKnowledgeStore() *memKnowledgeStore {
	return &memKnowledgeStore{docs: make(map[string]*schema.Document)}
}

func (s *memKnowledgeStore) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	_ = ctx
	_ = opts
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := make([]string, 0, len(docs))
	for _, d := range docs {
		if d == nil {
			continue
		}
		id := d.ID
		if id == "" {
			id = docID(d.Content)
		}
		cp := *d
		cp.ID = id
		s.docs[id] = &cp
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *memKnowledgeStore) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	_ = ctx
	options := retriever.GetCommonOptions(nil, opts...)
	topK := 3
	if options.TopK != nil {
		topK = *options.TopK
	}

	qTokens := tokenize(query)
	s.mu.RLock()
	defer s.mu.RUnlock()

	type scored struct {
		doc   *schema.Document
		score float64
	}
	ranked := make([]scored, 0, len(s.docs))
	for _, d := range s.docs {
		score := overlapScore(qTokens, tokenize(d.Content))
		if score <= 0 {
			continue
		}
		cp := *d
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

func (s *memKnowledgeStore) docCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.docs)
}

func docID(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:8])
}

func tokenize(s string) map[string]int {
	m := make(map[string]int)
	for _, w := range strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}) {
		if len([]rune(w)) < 2 {
			continue
		}
		m[w]++
	}
	return m
}

func overlapScore(query, doc map[string]int) float64 {
	if len(query) == 0 || len(doc) == 0 {
		return 0
	}
	var hit int
	for w, n := range query {
		if doc[w] > 0 {
			hit += n
		}
	}
	return float64(hit) / float64(len(query))
}
