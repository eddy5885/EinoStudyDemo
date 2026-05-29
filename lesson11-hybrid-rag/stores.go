package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/schema"
)

var (
	kwOnce  sync.Once
	kwStore *keywordStore

	vecOnce  sync.Once
	vecStore *vectorStore
)

func getKeywordStore() *keywordStore {
	kwOnce.Do(func() { kwStore = newKeywordStore() })
	return kwStore
}

func getVectorStore(ctx context.Context) *vectorStore {
	vecOnce.Do(func() {
		vecStore = newVectorStore(newEmbedder(ctx))
	})
	return vecStore
}

func ensureKeywordIndexed(ctx context.Context) error {
	s := getKeywordStore()
	if s.docCount() > 0 {
		return nil
	}
	chunks, err := loadKnowledgeChunks()
	if err != nil {
		return err
	}
	_, err = s.Store(ctx, chunks)
	return err
}

func ensureVectorIndexed(ctx context.Context) error {
	s := getVectorStore(ctx)
	if s.docCount() > 0 {
		return nil
	}
	chunks, err := loadKnowledgeChunks()
	if err != nil {
		return err
	}
	_, err = s.Store(ctx, chunks)
	return err
}

func (s *keywordStore) docCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.docs)
}

func printRetrievedDocs(label string, docs []*schema.Document) {
	fmt.Printf("%s:\n", label)
	if len(docs) == 0 {
		fmt.Println("  （无命中）")
		return
	}
	for i, d := range docs {
		fmt.Printf("  [%d] %s  score=%.4f\n", i+1, docTitle(d), d.Score())
	}
}
