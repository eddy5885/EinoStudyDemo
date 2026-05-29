package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// PipelineIn / PipelineOut 是 Workflow 的输入输出（可编译为 Runnable，也可封装成 Tool）。
type PipelineIn struct {
	Question string `json:"question"`
}

type PipelineOut struct {
	Answer string `json:"answer"`
}

var (
	pipelineOnce sync.Once
	pipelineRun  compose.Runnable[PipelineIn, PipelineOut]
	pipelineErr  error
)

func getPipeline(ctx context.Context, cm model.BaseChatModel) (compose.Runnable[PipelineIn, PipelineOut], error) {
	pipelineOnce.Do(func() {
		pipelineRun, pipelineErr = buildPipelineWorkflow(ctx, cm)
	})
	return pipelineRun, pipelineErr
}

func buildPipelineWorkflow(ctx context.Context, cm model.BaseChatModel) (compose.Runnable[PipelineIn, PipelineOut], error) {
	wf := compose.NewWorkflow[PipelineIn, PipelineOut]()

	wf.AddLambdaNode("to_messages", compose.InvokableLambda(
		func(_ context.Context, in PipelineIn) ([]*schema.Message, error) {
			return []*schema.Message{
				schema.SystemMessage("用一句话、不超过 30 字回答。"),
				schema.UserMessage(in.Question),
			}, nil
		},
	)).AddInput(compose.START)

	wf.AddChatModelNode("model", cm).AddInput("to_messages")

	wf.AddLambdaNode("pack", compose.InvokableLambda(
		func(_ context.Context, msg *schema.Message) (PipelineOut, error) {
			return PipelineOut{Answer: msg.Content}, nil
		},
	)).AddInput("model")

	wf.End().AddInput("pack")

	return wf.Compile(ctx)
}

func runWorkflowDemo(ctx context.Context) error {
	fmt.Println("\n=== Demo 3: Workflow（多节点流水线）===")

	cm := newChatModel(ctx)
	run, err := getPipeline(ctx, cm)
	if err != nil {
		return err
	}

	out, err := run.Invoke(ctx, PipelineIn{Question: "Chain 和 Graph 有什么区别？"})
	if err != nil {
		return err
	}
	fmt.Println("Workflow 输出:")
	fmt.Println(out.Answer)
	return nil
}
