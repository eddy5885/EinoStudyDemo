//go:debug x509negativeserial=1

// Lesson 11: Hybrid RAG（关键词 + 向量融合检索）
//
// 对比 Lesson 10：
//   - Lesson10：只用向量检索（语义稳，但有时“背景段”会排前）
//   - Lesson11：同时跑 keyword + vector，再用 RRF 融合排序（通常更稳）
//
// Demo 建议：
//   go run . -demo retrieve -q "总部在哪里"
//   go run . -demo rag -q "NeoStack 公司总部在哪里？"
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/cloudwego/eino/components/retriever"
)

func main() {
	demo := flag.String("demo", "retrieve", "retrieve | rag")
	question := flag.String("q", "", "问题（默认 LESSON11_QUESTION）")
	flag.Parse()

	q := *question
	if q == "" {
		q = defaultQuestion()
	}

	ctx := context.Background()
	var err error

	switch *demo {
	case "retrieve":
		err = runRetrieveDemo(ctx, q)
	case "rag":
		err = runRAGDemo(ctx, q)
	default:
		fmt.Fprintf(os.Stderr, "未知 demo: %s（retrieve | rag）\n", *demo)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runRetrieveDemo(ctx context.Context, q string) error {
	fmt.Println("=== Demo: Hybrid Retrieve（keyword + vector 融合）===")
	fmt.Printf("问题: %s\n\n", q)

	// 先确保索引就绪
	if err := ensureKeywordIndexed(ctx); err != nil {
		return err
	}
	if err := ensureVectorIndexed(ctx); err != nil {
		return err
	}

	kwDocs, _ := getKeywordStore().Retrieve(ctx, q, retriever.WithTopK(3))
	vecDocs, _ := getVectorStore(ctx).Retrieve(ctx, q, retriever.WithTopK(3))
	hyDocs, err := newHybridRetriever(ctx).Retrieve(ctx, q, retriever.WithTopK(3))
	if err != nil {
		return err
	}

	printRetrievedDocs("keyword", kwDocs)
	fmt.Println()
	printRetrievedDocs("vector", vecDocs)
	fmt.Println()
	printRetrievedDocs("hybrid(fused)", hyDocs)
	fmt.Println()
	fmt.Println("提示: hybrid 会并发跑多个 retriever，再做融合排序；最终喂给模型时仍会再 trim 到 Top3。")
	return nil
}

func runRAGDemo(ctx context.Context, q string) error {
	fmt.Println("=== Demo: Hybrid RAG（Retrieve → Template → ChatModel）===")
	fmt.Printf("问题: %s\n\n", q)

	if err := ensureKeywordIndexed(ctx); err != nil {
		return err
	}
	if err := ensureVectorIndexed(ctx); err != nil {
		return err
	}

	// 先看喂给模型前的检索结果（未 trim）
	docs, err := newHybridRetriever(ctx).Retrieve(ctx, q, retriever.WithTopK(3))
	if err != nil {
		return err
	}
	printRetrievedDocs("hybrid retrieve（before graph trim）", docs)
	fmt.Println()

	cm := newChatModel(ctx)
	rag, err := getHybridRAGGraph(ctx, cm)
	if err != nil {
		return err
	}
	msg, err := rag.Invoke(ctx, q)
	if err != nil {
		return err
	}
	fmt.Println(msg.Content)
	return nil
}

