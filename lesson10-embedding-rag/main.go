//go:debug x509negativeserial=1

// Lesson 10: RAG + Embedding 向量检索
//
// 对比 Lesson 09：
//   - Lesson09：关键词整词匹配，中文问「总部」可能命不中「办公地点」
//   - Lesson10：索引/检索时调用 Embedding API，按语义相似度选 TopK
//
// 环境变量（除 OPENAI_API_KEY / BASE_URL 外）：
//   OPENAI_EMBEDDING_MODEL  默认 BAAI/bge-large-zh-v1.5（SiliconFlow 等兼容平台常用）
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/cloudwego/eino/components/retriever"
)

func main() {
	demo := flag.String("demo", "compare", "index | compare | rag")
	question := flag.String("q", "", "问题（compare/rag）")
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
	case "compare":
		err = runCompareDemo(ctx, q)
	case "rag":
		err = runRAGDemo(ctx, q)
	default:
		fmt.Fprintf(os.Stderr, "未知 demo: %s（index | compare | rag）\n", *demo)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runRAGDemo(ctx context.Context, question string) error {
	fmt.Println("=== Demo: 向量 RAG（Retrieve → Template → ChatModel）===")
	fmt.Printf("问题: %s\n\n", question)

	if err := ensureVectorIndexed(ctx); err != nil {
		return err
	}

	vecDocs, err := getVectorStore(ctx).Retrieve(ctx, question, retriever.WithTopK(3))
	if err != nil {
		return err
	}
	printRetrievedDocs("检索片段", vecDocs)
	fmt.Println()

	cm := newChatModel(ctx)
	rag, err := getVectorRAGGraph(ctx, cm)
	if err != nil {
		return err
	}
	msg, err := rag.Invoke(ctx, question)
	if err != nil {
		return err
	}
	fmt.Println(msg.Content)
	return nil
}
