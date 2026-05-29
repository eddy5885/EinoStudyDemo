//go:debug x509negativeserial=1

// Lesson 06: Middleware（横切能力）
//
// 对比 Lesson 04/05：
//   - SafeToolMiddleware：Tool 报错 → 转成 "[tool error] ..." 字符串，对话不中断，模型可纠错
//   - ModelRetryConfig：ChatModel 遇到 429 等临时错误时自动重试
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
	log.SetFlags(log.Ltime)
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

	fmt.Println("Lesson 06 — Middleware")
	fmt.Printf("demo 目录: %s\n", workspace)
	fmt.Println("已启用: SafeToolMiddleware + ModelRetryConfig(429)")
	fmt.Println()
	fmt.Println("建议试两句：")
	fmt.Println("  1) 读取 notes.txt 的内容")
	fmt.Println("  2) 读取 not_exist.txt（应看到 [tool error]，但对话继续）")
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
		Name:        "Lesson06Agent",
		Description: "Agent with safe tool middleware and model retry.",
		Instruction: `你是中文助教。需要读文件、计算、查时间时必须调用工具。
若工具返回 [tool error] 开头的内容，说明调用失败，请根据错误信息换参数或换方案，不要假装成功。`,
		Model:         cm,
		MaxIterations: 12,
		Handlers: []adk.ChatModelAgentMiddleware{
			&safeToolMiddleware{BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{}},
		},
		ModelRetryConfig: &adk.ModelRetryConfig{
			MaxRetries:  3,
			IsRetryAble: isRetryableModelError,
		},
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
