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
	"github.com/cloudwego/eino/components/model"
)

func newChatModel(ctx context.Context) model.ToolCallingChatModel {
	if os.Getenv("EINO_CHAT_PROVIDER") == "ollama" {
		cm, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
			BaseURL: "http://localhost:11434",
			Model:   envOr("OLLAMA_MODEL", "llama3.2"),
		})
		if err != nil {
			log.Fatalf("create ollama model: %v", err)
		}
		return cm
	}

	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		log.Fatal(`请设置环境变量（与 lesson01/02 相同）`)
	}

	cfg := &einoopenai.ChatModelConfig{
		APIKey:  key,
		Model:   envOr("OPENAI_MODEL_NAME", "gpt-4o-mini"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
	}
	if os.Getenv("EINO_INSECURE_TLS") == "1" {
		cfg.HTTPClient = &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		}
	}
	cm, err := einoopenai.NewChatModel(ctx, cfg)
	if err != nil {
		log.Fatalf("create openai model: %v", err)
	}
	return cm
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
