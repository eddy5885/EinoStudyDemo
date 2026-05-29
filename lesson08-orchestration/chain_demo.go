package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// runChainDemo：链式编排 —— 数据单向流动 Template → ChatModel。
func runChainDemo(ctx context.Context) error {
	fmt.Println("=== Demo 1: Chain（链式）===")

	cm := newChatModel(ctx)
	tpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("你是简洁的中文助教。"),
		schema.UserMessage("用三句话介绍：{topic}"),
	)

	chain := compose.NewChain[map[string]any, *schema.Message]()
	chain.AppendChatTemplate(tpl).AppendChatModel(cm)

	runnable, err := chain.Compile(ctx)
	if err != nil {
		return err
	}

	msg, err := runnable.Invoke(ctx, map[string]any{"topic": demoTopic()})
	if err != nil {
		return err
	}
	fmt.Println("Chain 输出:")
	fmt.Println(msg.Content)
	return nil
}
