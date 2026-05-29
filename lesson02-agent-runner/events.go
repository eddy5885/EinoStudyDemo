package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// collectAssistant 消费 Runner 产出的事件流，打印并汇总本轮 assistant 文本。
func collectAssistant(events *adk.AsyncIterator[*adk.AgentEvent]) (string, error) {
	var sb strings.Builder
	fmt.Print("assistant> ")

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		mv := event.Output.MessageOutput
		// 本章无 Tool，只关心 assistant 输出
		if mv.Role != "" && mv.Role != schema.Assistant {
			continue
		}

		if mv.IsStreaming {
			for {
				frame, err := mv.MessageStream.Recv()
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					return "", err
				}
				sb.WriteString(frame.Content)
				fmt.Print(frame.Content)
			}
			fmt.Println()
			continue
		}

		if mv.Message != nil && mv.Message.Content != "" {
			sb.WriteString(mv.Message.Content)
			fmt.Println(mv.Message.Content)
		}
	}

	return sb.String(), nil
}
