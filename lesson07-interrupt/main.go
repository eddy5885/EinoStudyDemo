//go:debug x509negativeserial=1

// Lesson 07: Interrupt / Resume（人机协作）
//
// 对比 Lesson 06：
//   - CheckPointStore：中断时保存执行状态
//   - approvalMiddleware：calc 调用前 StatefulInterrupt，等待用户 y/n
//   - Runner.ResumeWithParams：用户批准后从中断点继续
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
	"github.com/google/uuid"
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

	fmt.Println("Lesson 07 — Interrupt / Resume")
	fmt.Println("calc 工具调用前会暂停，等待你输入 y/n 审批")
	fmt.Println("get_time、read_demo_file 无需审批")
	fmt.Println()
	fmt.Println("试试：计算 12345 * 6789")
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
		checkPointID := uuid.NewString()

		outcome, err := runUntilDone(ctx, runner, history, checkPointID, scanner)
		if err != nil {
			fmt.Fprintf(os.Stderr, "run: %v\n", err)
			os.Exit(1)
		}
		history = append(history, outcome.Messages...)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "stdin: %v\n", err)
		os.Exit(1)
	}
}

func runUntilDone(
	ctx context.Context,
	runner *adk.Runner,
	history []*schema.Message,
	checkPointID string,
	scanner *bufio.Scanner,
) (turnOutcome, error) {
	iter := runner.Run(ctx, history, adk.WithCheckPointID(checkPointID))
	outcome, err := consumeTurn(iter)
	if err != nil {
		return outcome, err
	}

	for outcome.Interrupted != nil {
		id, info := rootInterruptID(outcome.Interrupted)
		if id == "" {
			return outcome, fmt.Errorf("interrupt 事件缺少 context ID")
		}

		approved := promptApproval(scanner, info)
		result := &ApprovalResult{Approved: approved}
		if !approved {
			reason := "用户拒绝"
			result.DisapproveReason = &reason
		}

		resumeIter, err := runner.ResumeWithParams(ctx, checkPointID, &adk.ResumeParams{
			Targets: map[string]any{id: result},
		})
		if err != nil {
			return outcome, err
		}

		resumeOutcome, err := consumeTurn(resumeIter)
		if err != nil {
			return outcome, err
		}
		outcome.Messages = append(outcome.Messages, resumeOutcome.Messages...)
		outcome.Interrupted = resumeOutcome.Interrupted
	}

	return outcome, nil
}

func promptApproval(scanner *bufio.Scanner, info *ApprovalInfo) bool {
	fmt.Println()
	fmt.Println("⚠️  需要审批（Interrupt）")
	if info != nil {
		fmt.Printf("Tool: %s\n", info.ToolName)
		fmt.Printf("参数: %s\n", info.ArgumentsInJSON)
	} else {
		fmt.Println("Tool: calc（待审批）")
	}
	for {
		fmt.Print("批准？(y/n): ")
		if !scanner.Scan() {
			return false
		}
		switch strings.ToLower(strings.TrimSpace(scanner.Text())) {
		case "y", "yes":
			fmt.Println("已批准，继续执行…")
			return true
		case "n", "no":
			fmt.Println("已拒绝，将通知模型…")
			return false
		default:
			fmt.Println("请输入 y 或 n")
		}
	}
}

func newRunner(ctx context.Context, tools []tool.BaseTool) (*adk.Runner, error) {
	cm := newChatModel(ctx)

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "Lesson07Agent",
		Description: "Agent with tool approval interrupt.",
		Instruction: `你是中文助教。计算必须用 calc 工具；查时间用 get_time。
calc 若返回「被拒绝」，向用户说明并勿编造结果。`,
		Model:         cm,
		MaxIterations: 12,
		Handlers: []adk.ChatModelAgentMiddleware{
			&approvalMiddleware{BaseChatModelAgentMiddleware: &adk.BaseChatModelAgentMiddleware{}},
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
		CheckPointStore: newMemoryCheckPointStore(),
	}), nil
}
