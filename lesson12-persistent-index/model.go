package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	aclopenai "github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/model"
)

func newChatModel(ctx context.Context) model.ToolCallingChatModel {
	if os.Getenv("EINO_CHAT_PROVIDER") == "ollama" {
		cm, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
			BaseURL: "http://localhost:11434",
			Model:   envOr("OLLAMA_MODEL", "llama3.2"),
		})
		if err != nil {
			log.Fatalf("create ollama: %v", err)
		}
		return cm
	}
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		log.Fatal(`请设置 OPENAI_API_KEY`)
	}
	cfg := &einoopenai.ChatModelConfig{
		APIKey:  key,
		Model:   envOr("OPENAI_MODEL_NAME", "gpt-4o-mini"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
	}
	cfg.HTTPClient = openaiHTTPClient()
	cm, err := einoopenai.NewChatModel(ctx, cfg)
	if err != nil {
		log.Fatalf("create chat model: %v", err)
	}
	return cm
}

func newEmbedder(ctx context.Context) embedding.Embedder {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		log.Fatal(`向量检索需要 OPENAI_API_KEY（与对话模型共用）`)
	}
	cfg := &aclopenai.EmbeddingConfig{
		APIKey:     key,
		BaseURL:    os.Getenv("OPENAI_BASE_URL"),
		Model:      envOr("OPENAI_EMBEDDING_MODEL", "BAAI/bge-large-zh-v1.5"),
		HTTPClient: openaiHTTPClient(),
	}
	emb, err := aclopenai.NewEmbeddingClient(ctx, cfg)
	if err != nil {
		log.Fatalf("create embedder: %v", err)
	}
	return emb
}

func openaiHTTPClient() *http.Client {
	if os.Getenv("EINO_INSECURE_TLS") != "1" {
		return http.DefaultClient
	}
	return &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func defaultQuestion() string {
	return envOr("LESSON12_QUESTION", "NeoStack 公司总部在哪里？")
}

