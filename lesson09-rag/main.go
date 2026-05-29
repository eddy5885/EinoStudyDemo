//go:debug x509negativeserial=1

// Lesson 09: RAG（检索增强生成）
//
// 对比 Lesson 08：
//   - 在调模型之前，先用 Retriever 从知识库取片段（本地关键词检索，无需 Embedding API）
//   - Graph: Retrieve → 拼 context → Template → ChatModel
//   - 体会：多出来的不是「更好的提示词」，而是「资料里有什么」
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	demo := flag.String("demo", "compare", "index | rag | compare | agent")
	question := flag.String("q", "", "问题（compare/rag 使用，默认 LESSON09_QUESTION）")
	flag.Parse()

	q := *question
	if q == "" {
		q = defaultQuestion()
	}

	ctx := context.Background()
	var err error

	switch *demo {
	case "index":
		err = runIndexDemo(ctx)
	case "rag":
		err = runRAGDemo(ctx, q)
	case "compare":
		err = runCompareDemo(ctx, q)
	case "agent":
		err = runAgentDemo(ctx)
	default:
		fmt.Fprintf(os.Stderr, "未知 demo: %s（index | rag | compare | agent）\n", *demo)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runRAGDemo(ctx context.Context, question string) error {
	fmt.Println("=== Demo: RAG Graph ===")
	fmt.Printf("问题: %s\n\n", question)

	if err := ensureIndexed(ctx); err != nil {
		return err
	}

	cm := newChatModel(ctx)
	rag, err := getRAGGraph(ctx, cm)
	if err != nil {
		return err
	}

	docs, _ := getKnowledgeStore().Retrieve(ctx, question)
	fmt.Println("检索片段:")
	for i, d := range docs {
		fmt.Printf("  [%d] score=%.2f\n", i+1, d.Score())
	}
	fmt.Println()

	msg, err := rag.Invoke(ctx, question)
	if err != nil {
		return err
	}
	fmt.Println(msg.Content)
	return nil
}
