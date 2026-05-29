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

func runAgentDemo(ctx context.Context) error {
	fmt.Println("=== Demo: RAG 流水线封装为 Agent Tool ===")
	fmt.Println("问 NeoStack 年假、办公地点、报销时限等（答案在 demo/knowledge.md）")
	fmt.Println("也可闲聊；技术政策类问题应走 ask_knowledge 工具。")
	fmt.Println("空行退出。\n")

	if err := ensureIndexed(ctx); err != nil {
		return err
	}

	cm := newChatModel(ctx)
	rag, err := getRAGGraph(ctx, cm)
	if err != nil {
		return err
	}

	askKnowledge, err := utils.InferTool("ask_knowledge",
		"查询 NeoStack 内部知识库后回答，适用于年假、办公地点、报销等公司制度问题",
		func(ctx context.Context, in struct {
			Question string `json:"question" jsonschema_description:"用户问题"`
		}) (string, error) {
			msg, err := rag.Invoke(ctx, in.Question)
			if err != nil {
				return "", err
			}
			return msg.Content, nil
		})
	if err != nil {
		return err
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "Lesson09Agent",
		Description: "NeoStack assistant with knowledge base tool.",
		Instruction: `你是 NeoStack 助手。涉及公司制度、年假、办公地点、报销等问题时，必须调用 ask_knowledge，不要凭记忆编造。`,
		Model:         cm,
		MaxIterations: 8,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{Tools: []tool.BaseTool{askKnowledge}},
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
