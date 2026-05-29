package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	cbutil "github.com/cloudwego/eino/utils/callbacks"
)

type toolStartKey struct{}

// registerTraceCallbacks 注册全局 Callback，观察 ChatModel / Tool 生命周期。
// 业务代码无需改动，组件执行时会自动触发。
func registerTraceCallbacks() {
	startKey := toolStartKey{}

	handler := cbutil.NewHandlerHelper().
		ChatModel(&cbutil.ModelCallbackHandler{
			OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
				log.Printf("[trace] ChatModel ▶ %s  messages=%d tools=%d",
					label(info), len(input.Messages), len(input.Tools))
				return ctx
			},
			OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *model.CallbackOutput) context.Context {
				usage := ""
				if output != nil && output.TokenUsage != nil {
					u := output.TokenUsage
					usage = fmt.Sprintf(" tokens=%d (prompt=%d completion=%d)",
						u.TotalTokens, u.PromptTokens, u.CompletionTokens)
				}
				toolCalls := 0
				if output != nil && output.Message != nil {
					toolCalls = len(output.Message.ToolCalls)
				}
				log.Printf("[trace] ChatModel ◀ %s  tool_calls=%d%s",
					label(info), toolCalls, usage)
				return ctx
			},
			OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
				log.Printf("[trace] ChatModel ✗ %s  err=%v", label(info), err)
				return ctx
			},
		}).
		Tool(&cbutil.ToolCallbackHandler{
			OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *tool.CallbackInput) context.Context {
				log.Printf("[trace] Tool ▶ %s  args=%s", label(info), truncate(input.ArgumentsInJSON, 100))
				return context.WithValue(ctx, startKey, time.Now())
			},
			OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *tool.CallbackOutput) context.Context {
				dur := time.Duration(0)
				if t, ok := ctx.Value(startKey).(time.Time); ok {
					dur = time.Since(t)
				}
				resp := ""
				if output != nil {
					resp = truncate(output.Response, 100)
				}
				log.Printf("[trace] Tool ◀ %s  duration=%v  response=%s", label(info), dur.Round(time.Millisecond), resp)
				return ctx
			},
			OnError: func(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
				log.Printf("[trace] Tool ✗ %s  err=%v", label(info), err)
				return ctx
			},
		}).
		Handler()

	callbacks.AppendGlobalHandlers(handler)
}

func label(info *callbacks.RunInfo) string {
	if info == nil {
		return "unknown"
	}
	comp := string(info.Component)
	if info.Name != "" {
		return comp + "/" + info.Name
	}
	return comp
}
