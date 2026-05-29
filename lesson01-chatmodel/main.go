//go:debug x509negativeserial=1

// Lesson 01: 从「直接调 API」到 Eino 的 ChatModel
//
// 你熟悉的 HTTP 调用 ≈ 自己拼 messages + POST /chat/completions
// Eino 把这一步抽象成 ChatModel：输入 []*schema.Message，输出 *schema.Message
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()
	chatModel, provider := newChatModel(ctx)
	fmt.Printf("provider: %s\n\n", provider)

	messages := []*schema.Message{
		schema.SystemMessage("你是一个简洁的技术助教，用中文回答。"),
		schema.UserMessage("用三句话说明：Eino 的 Component 抽象解决了什么问题？"),
	}

	fmt.Println("=== Generate（一次性返回）===")
	out, err := chatModel.Generate(ctx, messages)
	if err != nil {
		log.Fatalf("generate: %v", err)
	}
	fmt.Println(out.Content)

	fmt.Println("\n=== Stream（流式返回）===")
	stream, err := chatModel.Stream(ctx, messages)
	if err != nil {
		log.Fatalf("stream: %v", err)
	}
	defer stream.Close()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("stream recv: %v", err)
		}
		fmt.Print(chunk.Content)
	}
	fmt.Println()
}

func newChatModel(ctx context.Context) (model.ToolCallingChatModel, string) {
	if os.Getenv("EINO_CHAT_PROVIDER") == "ollama" {
		cm, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
			BaseURL: "http://localhost:11434",
			Model:   envOr("OLLAMA_MODEL", "llama3.2"),
		})
		if err != nil {
			log.Fatalf("create ollama model: %v", err)
		}
		return cm, "ollama"
	}

	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		log.Fatal(`请设置环境变量后重试，例如：
  export OPENAI_API_KEY="sk-..."
  export OPENAI_MODEL_NAME="gpt-4o-mini"
  # 可选：兼容 OpenAI 的代理
  # export OPENAI_BASE_URL="https://api.openai.com/v1"

本地 Ollama：
  export EINO_CHAT_PROVIDER=ollama
  export OLLAMA_MODEL=llama3.2`)
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
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // 仅本地调试
			},
		}
	}
	cm, err := einoopenai.NewChatModel(ctx, cfg)
	if err != nil {
		log.Fatalf("create openai model: %v", err)
	}
	return cm, "openai"
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
