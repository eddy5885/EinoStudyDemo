//go:debug x509negativeserial=1

// Lesson 02: ChatModelAgent + Runner（控制台多轮）
//
// 对比 Lesson 01：
//   - Lesson 01：你直接 chatModel.Generate(messages)
//   - Lesson 02：Agent 封装「指令 + 模型」，Runner 驱动执行，通过 AgentEvent 观察过程
//
// 多轮对话：由调用方维护 history（本章尚未用 Memory/Session 持久化）
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()
	cm := newChatModel(ctx)

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "Lesson02Agent",
		Description: "A minimal console agent for learning Eino ADK.",
		Instruction: "你是简洁的中文助教。回答要短，适合命令行阅读。",
		Model:       cm,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "create agent: %v\n", err)
		os.Exit(1)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	fmt.Println("Lesson 02 — 多轮对话（空行或 Ctrl+D 退出）")
	fmt.Println("提示：Instruction 由 Agent 注入，history 里只需 user/assistant 轮次。")

	history := make([]*schema.Message, 0, 16)
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("you> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		history = append(history, schema.UserMessage(line))
		events := runner.Run(ctx, history)
		reply, err := collectAssistant(events)
		if err != nil {
			fmt.Fprintf(os.Stderr, "run agent: %v\n", err)
			os.Exit(1)
		}
		history = append(history, schema.AssistantMessage(reply, nil))
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "stdin: %v\n", err)
		os.Exit(1)
	}
}
