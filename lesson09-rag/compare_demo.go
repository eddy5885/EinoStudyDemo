package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
)

// runCompareDemo：同一问题，无检索 vs 有检索，体会 RAG 差异。
func runCompareDemo(ctx context.Context, question string) error {
	fmt.Println("=== Demo: 对比 无 RAG / 有 RAG ===")
	fmt.Printf("问题: %s\n\n", question)

	if err := ensureIndexed(ctx); err != nil {
		return err
	}

	cm := newChatModel(ctx)

	fmt.Println("--- 1) 无 RAG：直接把问题发给模型 ---")
	plain, err := cm.Generate(ctx, []*schema.Message{
		schema.SystemMessage("你是 NeoStack 员工助手，请简洁回答。"),
		schema.UserMessage(question),
	})
	if err != nil {
		return err
	}
	fmt.Println(plain.Content)
	fmt.Println()

	rag, err := getRAGGraph(ctx, cm)
	if err != nil {
		return err
	}

	fmt.Println("--- 2) 有 RAG：先检索 demo/knowledge.md 片段再回答 ---")
	docs, _ := getKnowledgeStore().Retrieve(ctx, question, retriever.WithTopK(3))
	fmt.Println("检索到的片段:")
	for i, d := range docs {
		title, _ := d.MetaData["title"].(string)
		fmt.Printf("  [%d] %s (score=%.2f)\n", i+1, title, d.Score())
	}
	fmt.Println()

	grounded, err := rag.Invoke(ctx, question)
	if err != nil {
		return err
	}
	fmt.Println(grounded.Content)
	fmt.Println()
	fmt.Println("提示: 无 RAG 时模型容易编造「年假天数/城市」；有 RAG 时应引用知识库中的 12 天、杭州等。")
	return nil
}
