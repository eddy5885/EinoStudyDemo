package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// runGraphDemo：图编排 —— 先分支选风格，再 Template → ChatModel。
func runGraphDemo(ctx context.Context) error {
	fmt.Println("\n=== Demo 2: Graph（分支 + 节点）===")

	cm := newChatModel(ctx)

	g := compose.NewGraph[map[string]any, *schema.Message]()

	branch := compose.NewGraphBranch(
		func(_ context.Context, in map[string]any) (string, error) {
			if style, _ := in["style"].(string); style == "poem" {
				return "set_poem", nil
			}
			return "set_plain", nil
		},
		map[string]bool{"set_poem": true, "set_plain": true},
	)

	_ = g.AddLambdaNode("set_poem", compose.InvokableLambda(func(_ context.Context, in map[string]any) (map[string]any, error) {
		in["hint"] = "请用四句诗"
		return in, nil
	}))
	_ = g.AddLambdaNode("set_plain", compose.InvokableLambda(func(_ context.Context, in map[string]any) (map[string]any, error) {
		in["hint"] = "请用平实说明文"
		return in, nil
	}))

	tpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage("{hint}"),
		schema.UserMessage("主题：{topic}"),
	)
	_ = g.AddChatTemplateNode("template", tpl)
	_ = g.AddChatModelNode("model", cm)

	_ = g.AddBranch(compose.START, branch)
	_ = g.AddEdge("set_poem", "template")
	_ = g.AddEdge("set_plain", "template")
	_ = g.AddEdge("template", "model")
	_ = g.AddEdge("model", compose.END)

	runnable, err := g.Compile(ctx)
	if err != nil {
		return err
	}

	msg, err := runnable.Invoke(ctx, map[string]any{
		"style": "poem",
		"topic": demoTopic(),
	})
	if err != nil {
		return err
	}
	fmt.Println("Graph 输出 (style=poem):")
	fmt.Println(msg.Content)
	return nil
}
