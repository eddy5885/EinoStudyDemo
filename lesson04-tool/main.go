//go:debug x509negativeserial=1

// Lesson 04: Tool（让 Agent 能「动手」）
//
// 对比 Lesson 03：
//   - 不再只是生成文本，模型可发起 tool_call，由 ToolsNode 执行后把结果塞回上下文
//   - 使用 utils.InferTool 从 Go 函数生成 InvokableTool
//   - history 需保留 assistant(tool_calls) + tool 消息，否则下一轮模型会「失忆」
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
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

	fmt.Println("Lesson 04 — Agent + Tool")
	fmt.Printf("demo 目录: %s\n", workspace)
	fmt.Println("内置工具: get_time | calc | read_demo_file")
	fmt.Println("试试：")
	fmt.Println("  - 用 Asia/Shanghai 时区现在几点？")
	fmt.Println("  - 计算 12345 * 6789")
	fmt.Println("  - 读取 notes.txt 里关于 Eino 的那句话")
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
		Name:        "Lesson04Agent",
		Description: "Console agent with calculator, clock and demo file tools.",
		Instruction: `你是中文助教。涉及当前时间、数学计算、读取 demo 目录文件时，必须调用工具，不要猜测或心算。
工具 read_demo_file 只能读取 demo 目录内文件。`,
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
