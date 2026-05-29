package main

import (
	"context"
	"fmt"
)

func runIndexDemo(ctx context.Context) error {
	fmt.Println("=== Demo: 向量索引（Embedder + Indexer.Store）===")
	if err := ensureVectorIndexed(ctx); err != nil {
		return err
	}
	fmt.Printf("向量库已就绪，共 %d 个片段（来自 %s）\n", getVectorStore(ctx).docCount(), demoKnowledgePath())
	return nil
}
