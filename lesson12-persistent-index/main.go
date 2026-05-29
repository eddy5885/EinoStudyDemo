//go:debug x509negativeserial=1

// Lesson 12: 持久化向量索引（一次向量化，多次复用）
//
// 你会看到：
// - 第一次 build：会调用 Embedding API 把文档向量化，并写入 data/index.json
// - 之后 retrieve/rag：直接加载 data/index.json，不再重复向量化文档（只 embed query）
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/cloudwego/eino/components/retriever"
)

func main() {
	demo := flag.String("demo", "rag", "build | retrieve | rag")
	question := flag.String("q", "", "问题（默认 LESSON12_QUESTION）")
	flag.Parse()

	q := *question
	if q == "" {
		q = defaultQuestion()
	}

	ctx := context.Background()
	var err error

	switch *demo {
	case "build":
		err = runBuild(ctx)
	case "retrieve":
		err = runRetrieve(ctx, q)
	case "rag":
		err = runRAG(ctx, q)
	default:
		fmt.Fprintf(os.Stderr, "未知 demo: %s（build | retrieve | rag）\n", *demo)
		os.Exit(2)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func storePath() string { return "data/index.json" }

func newStore(ctx context.Context) *persistedVectorStore {
	return newPersistedVectorStore(newEmbedder(ctx), storePath())
}

func runBuild(ctx context.Context) error {
	fmt.Println("=== Demo: build persisted index ===")
	chunks, err := loadKnowledgeChunks()
	if err != nil {
		return err
	}
	s := newStore(ctx)
	ids, err := s.Store(ctx, chunks)
	if err != nil {
		return err
	}
	fmt.Printf("已写入索引: %s（%d chunks）\n", storePath(), len(ids))
	return nil
}

func runRetrieve(ctx context.Context, q string) error {
	fmt.Println("=== Demo: retrieve (persisted vectors) ===")
	fmt.Printf("问题: %s\n\n", q)
	s := newStore(ctx)
	docs, err := s.Retrieve(ctx, q, retriever.WithTopK(3))
	if err != nil {
		return err
	}
	for i, d := range docs {
		fmt.Printf("[%d] %s score=%.4f\n", i+1, docTitle(d), d.Score())
	}
	return nil
}

func runRAG(ctx context.Context, q string) error {
	fmt.Println("=== Demo: RAG with persisted index ===")
	fmt.Printf("问题: %s\n\n", q)

	ret := newStore(ctx)
	cm := newChatModel(ctx)
	rag, err := getRAGGraph(ctx, cm, ret)
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

