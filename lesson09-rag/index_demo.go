package main

import (
	"context"
	"fmt"
)

// runIndexDemo：把 demo/knowledge.md 切块写入 Indexer（不调模型）。
func runIndexDemo(ctx context.Context) error {
	fmt.Println("=== Demo: 索引（Indexer.Store）===")

	chunks, err := loadKnowledgeChunks()
	if err != nil {
		return err
	}
	ids, err := getKnowledgeStore().Store(ctx, chunks)
	if err != nil {
		return err
	}
	fmt.Printf("已索引 %d 个片段（来自 %s）:\n", len(ids), demoKnowledgePath())
	for i, id := range ids {
		fmt.Printf("  [%d] id=%s\n", i+1, id)
	}
	return nil
}
