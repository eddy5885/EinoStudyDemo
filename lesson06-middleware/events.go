package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func processTurn(events *adk.AsyncIterator[*adk.AgentEvent]) ([]*schema.Message, error) {
	var turnMessages []*schema.Message

	for {
		event, ok := events.Next()
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

		switch msg.Role {
		case schema.Tool:
			line := truncate(msg.Content, 200)
			if strings.HasPrefix(msg.Content, "[tool error]") {
				fmt.Printf("[tool %s] %s\n", msg.ToolName, line)
			} else {
				fmt.Printf("[tool %s] %s\n", msg.ToolName, line)
			}
			turnMessages = append(turnMessages, cloneMessage(msg))
		case schema.Assistant:
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					fmt.Printf("[tool call] %s(%s)\n", tc.Function.Name, formatToolArgs(tc.Function.Arguments))
				}
				turnMessages = append(turnMessages, cloneMessage(msg))
				continue
			}
			if msg.Content != "" {
				fmt.Printf("assistant> %s\n", msg.Content)
				turnMessages = append(turnMessages, cloneMessage(msg))
			}
		}
	}

	return turnMessages, nil
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
