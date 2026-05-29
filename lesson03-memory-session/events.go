package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

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
