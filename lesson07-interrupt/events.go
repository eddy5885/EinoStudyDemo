package main

import (
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// turnOutcome 一轮 Run/Resume 消费结果。
type turnOutcome struct {
	Messages   []*schema.Message
	Interrupted *adk.AgentEvent
}

func consumeTurn(iter *adk.AsyncIterator[*adk.AgentEvent]) (turnOutcome, error) {
	var out turnOutcome

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return out, event.Err
		}
		if event.Action != nil && event.Action.Interrupted != nil {
			out.Interrupted = event
			continue
		}
		msgs, err := messagesFromEvent(event)
		if err != nil {
			return out, err
		}
		out.Messages = append(out.Messages, msgs...)
	}
	return out, nil
}

func messagesFromEvent(event *adk.AgentEvent) ([]*schema.Message, error) {
	if event.Output == nil || event.Output.MessageOutput == nil {
		return nil, nil
	}
	msg, err := event.Output.MessageOutput.GetMessage()
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, nil
	}

	switch msg.Role {
	case schema.Tool:
		fmt.Printf("[tool %s] %s\n", msg.ToolName, truncate(msg.Content, 200))
		return []*schema.Message{cloneMessage(msg)}, nil
	case schema.Assistant:
		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				fmt.Printf("[tool call] %s(%s)\n", tc.Function.Name, formatToolArgs(tc.Function.Arguments))
			}
			return []*schema.Message{cloneMessage(msg)}, nil
		}
		if msg.Content != "" {
			fmt.Printf("assistant> %s\n", msg.Content)
			return []*schema.Message{cloneMessage(msg)}, nil
		}
	}
	return nil, nil
}

func rootInterruptID(event *adk.AgentEvent) (string, *ApprovalInfo) {
	if event == nil || event.Action == nil || event.Action.Interrupted == nil {
		return "", nil
	}
	for _, ctx := range event.Action.Interrupted.InterruptContexts {
		if !ctx.IsRootCause {
			continue
		}
		if info, ok := ctx.Info.(*ApprovalInfo); ok {
			return ctx.ID, info
		}
		return ctx.ID, nil
	}
	if len(event.Action.Interrupted.InterruptContexts) > 0 {
		ctx := event.Action.Interrupted.InterruptContexts[0]
		if info, ok := ctx.Info.(*ApprovalInfo); ok {
			return ctx.ID, info
		}
		return ctx.ID, nil
	}
	return "", nil
}

func cloneMessage(msg *schema.Message) *schema.Message {
	cp := *msg
	return &cp
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}

func formatToolArgs(args string) string {
	var v any
	if err := json.Unmarshal([]byte(args), &v); err != nil {
		return args
	}
	b, err := json.Marshal(v)
	if err != nil {
		return args
	}
	return string(b)
}
