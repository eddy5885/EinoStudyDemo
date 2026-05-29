package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// runAgentDemo：把编译好的 Workflow 封装成 Tool，交给 ChatModelAgent 调用。
func runAgentDemo(ctx context.Context) error {
	fmt.Println("\n=== Demo 4: Workflow 作为 Agent 的 Tool ===")
	fmt.Println("模型可调用 ask_pipeline 工具（内部跑 Workflow，不是单次 Generate）")
	fmt.Println("试试：用流水线工具回答「什么是 Graph 编排？」")
	fmt.Println("空行退出。\n")

	cm := newChatModel(ctx)
	pipeline, err := getPipeline(ctx, cm)
	if err != nil {
		return err
	}

	askPipeline, err := utils.InferTool("ask_pipeline",
		"通过固定流水线（Template+Model）回答问题，适合需要稳定流程的场景",
		func(ctx context.Context, in struct {
			Question string `json:"question" jsonschema_description:"用户问题"`
		}) (string, error) {
			out, err := pipeline.Invoke(ctx, PipelineIn{Question: in.Question})
			if err != nil {
				return "", err
			}
			return out.Answer, nil
		})
	if err != nil {
		return err
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "Lesson08Agent",
		Description: "Agent that can call a pipeline tool.",
		Instruction: "用户问技术问题时，优先使用 ask_pipeline 工具回答。",
		Model:         cm,
		MaxIterations: 8,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{Tools: []tool.BaseTool{askPipeline}},
		},
	})
	if err != nil {
		return err
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{Agent: agent, EnableStreaming: false})
	history := make([]*schema.Message, 0, 8)
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
		iter := runner.Run(ctx, history)
		msgs, err := collectAssistantTurn(iter)
		if err != nil {
			return err
		}
		history = append(history, msgs...)
	}
	return scanner.Err()
}

func collectAssistantTurn(iter *adk.AsyncIterator[*adk.AgentEvent]) ([]*schema.Message, error) {
	var turn []*schema.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return nil, event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}
		msg, err := event.Output.MessageOutput.GetMessage()
		if err != nil {
			return nil, err
		}
		if msg == nil {
			continue
		}
		if msg.Role == schema.Tool {
			fmt.Printf("[tool %s] %s\n", msg.ToolName, msg.Content)
			turn = append(turn, msg)
			continue
		}
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				fmt.Printf("[tool call] %s\n", tc.Function.Name)
			}
			turn = append(turn, msg)
			continue
		}
		if msg.Content != "" {
			fmt.Printf("assistant> %s\n", msg.Content)
			turn = append(turn, msg)
		}
	}
	return turn, nil
}
