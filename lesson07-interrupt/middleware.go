package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

func init() {
	// CheckPoint 用 gob 序列化中断信息，自定义类型必须先注册。
	schema.Register[ApprovalInfo]()
	schema.Register[ApprovalResult]()
}

// ApprovalInfo 展示给用户的中断信息（会出现在 InterruptCtx.Info）。
type ApprovalInfo struct {
	ToolName        string `json:"tool_name"`
	ArgumentsInJSON string `json:"arguments"`
}

// ApprovalResult Resume 时用户传入的审批结果。
type ApprovalResult struct {
	Approved         bool
	DisapproveReason *string
}

// approvalMiddleware 对 calc 工具做人工审批：第一次调用中断，Resume 后再真正执行。
type approvalMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func needsApproval(toolName string) bool {
	return toolName == "calc"
}

func (m *approvalMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	if tCtx == nil || !needsApproval(tCtx.Name) {
		return endpoint, nil
	}

	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return "", tool.StatefulInterrupt(ctx, &ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: args,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*ApprovalResult](ctx)
		if isTarget && hasData {
			if data.Approved {
				return endpoint(ctx, storedArgs, opts...)
			}
			if data.DisapproveReason != nil {
				return fmt.Sprintf("tool '%s' 被拒绝: %s", tCtx.Name, *data.DisapproveReason), nil
			}
			return fmt.Sprintf("tool '%s' 被用户拒绝", tCtx.Name), nil
		}

		isTarget, _, _ = tool.GetResumeContext[any](ctx)
		if !isTarget {
			return "", tool.StatefulInterrupt(ctx, &ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: storedArgs,
			}, storedArgs)
		}

		return endpoint(ctx, storedArgs, opts...)
	}, nil
}
