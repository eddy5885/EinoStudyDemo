package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/retriever"
)

// runCompareDemo：同一问题，关键词 Top3 vs 向量 Top3（只比检索，不调对话模型）。
func runCompareDemo(ctx context.Context, question string) error {
	fmt.Println("=== Demo: 检索对比（关键词 vs 向量）===")
	fmt.Printf("问题: %s\n\n", question)

	if err := ensureKeywordIndexed(ctx); err != nil {
		return err
	}
	if err := ensureVectorIndexed(ctx); err != nil {
		return err
	}

	kwDocs, err := getKeywordStore().Retrieve(ctx, question, retriever.WithTopK(3))
	if err != nil {
		return err
	}
	vecDocs, err := getVectorStore(ctx).Retrieve(ctx, question, retriever.WithTopK(3))
	if err != nil {
		return err
	}

	printRetrievedDocs("关键词检索 (Lesson09 同款)", kwDocs)
	fmt.Println()
	printRetrievedDocs("向量检索 (Embedding 余弦相似度)", vecDocs)
	fmt.Println()
	fmt.Println("预期: 问「总部在哪里」时，向量检索应更常命中「办公地点」；关键词可能命中「休假制度」或为空。")
	return nil
}
