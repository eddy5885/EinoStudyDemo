//go:debug x509negativeserial=1

// Lesson 05: Callback / Trace（可观测性）
//
// 对比 Lesson 04：
//   - 业务能力相同（Agent + Tool）
//   - 新增全局 Callback：在 ChatModel / Tool 执行前后自动打 [trace] 日志
//   - 主流程不改，Callback 是「旁路」钩子
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	registerTraceCallbacks()

	ctx := context.Background()
	workspace, _ := filepath.Abs(demoWorkspace())

	tools, err := buildTools(workspace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build tools: %v\n", err)
		os.Exit(1)
	}

	runner, err := newRunner(ctx, tools)
	if err != nil {
		fmt.Fprintf(os.Stderr, "runner: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Lesson 05 — Agent + Tool + Callback")
	fmt.Println("[trace] 日志在 stderr；对话在 stdout")
	fmt.Printf("demo 目录: %s\n\n", workspace)
	fmt.Println("试试：计算 999 * 888（应看到 2 次 ChatModel + 1 次 Tool 的 trace）")
	fmt.Println("空行退出。\n")

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

		log.Printf("[trace] === user turn: %q ===", line)
		history = append(history, schema.UserMessage(line))
		events := runner.Run(ctx, history)
		turnMsgs, err := processTurn(events)
		if err != nil {
			fmt.Fprintf(os.Stderr, "run agent: %v\n", err)
			os.Exit(1)
		}
		history = append(history, turnMsgs...)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "stdin: %v\n", err)
		os.Exit(1)
	}
}

func newRunner(ctx context.Context, tools []tool.BaseTool) (*adk.Runner, error) {
	cm := newChatModel(ctx)

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "Lesson05Agent",
		Description: "Agent with tools and global trace callbacks.",
		Instruction: `你是中文助教。涉及时间、计算、读 demo 文件时必须调用工具。`,
		Model:         cm,
		MaxIterations: 12,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: false,
	}), nil
}
