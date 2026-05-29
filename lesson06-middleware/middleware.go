package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

// safeToolMiddleware 把 Tool 错误转成字符串结果，交给模型自行纠错，而不是中断整轮对话。
// 放在 Handlers 数组末尾（最内层），与官方建议一致。
type safeToolMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func (m *safeToolMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		result, err := endpoint(ctx, args, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return "", err
			}
			msg := fmt.Sprintf("[tool error] %v", err)
			log.Printf("[mw] tool %s 失败 → 转为模型可读: %s", toolName(tCtx), msg)
			return msg, nil
		}
		return result, nil
	}, nil
}

func toolName(tCtx *adk.ToolContext) string {
	if tCtx == nil {
		return "?"
	}
	return tCtx.Name
}
