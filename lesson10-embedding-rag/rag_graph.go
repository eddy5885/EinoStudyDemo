package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type ragState struct {
	Question string
}

var (
	vecRAGOnce sync.Once
	vecRAGRun  compose.Runnable[string, *schema.Message]
	vecRAGErr  error
)

func getVectorRAGGraph(ctx context.Context, cm model.BaseChatModel) (compose.Runnable[string, *schema.Message], error) {
	vecRAGOnce.Do(func() {
		vecRAGRun, vecRAGErr = buildRAGGraph(ctx, cm, getVectorStore(ctx))
	})
	return vecRAGRun, vecRAGErr
}

func buildRAGGraph(ctx context.Context, cm model.BaseChatModel, ret retriever.Retriever) (compose.Runnable[string, *schema.Message], error) {
	g := compose.NewGraph[string, *schema.Message](compose.WithGenLocalState(func(context.Context) *ragState {
		return &ragState{}
	}))

	_ = g.AddLambdaNode("stash_query", compose.InvokableLambda(func(ctx context.Context, question string) (string, error) {
		_ = compose.ProcessState(ctx, func(_ context.Context, s *ragState) error {
			s.Question = question
			return nil
		})
		return question, nil
	}))
	_ = g.AddRetrieverNode("retrieve", ret)
	_ = g.AddLambdaNode("to_prompt", compose.InvokableLambda(func(ctx context.Context, docs []*schema.Document) (map[string]any, error) {
		var question string
		_ = compose.ProcessState(ctx, func(_ context.Context, s *ragState) error {
			question = s.Question
			return nil
		})
		return map[string]any{
			"context":  formatDocs(docs),
			"question": question,
		}, nil
	}))

	tpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage(`你是 NeoStack 内部问答助手。只能根据「参考资料」回答。
若资料中没有答案，请明确说「知识库中没有相关信息」，不要编造。`),
		schema.UserMessage("参考资料：\n{context}\n\n问题：{question}"),
	)
	_ = g.AddChatTemplateNode("template", tpl)
	_ = g.AddChatModelNode("model", cm)

	_ = g.AddEdge(compose.START, "stash_query")
	_ = g.AddEdge("stash_query", "retrieve")
	_ = g.AddEdge("retrieve", "to_prompt")
	_ = g.AddEdge("to_prompt", "template")
	_ = g.AddEdge("template", "model")
	_ = g.AddEdge("model", compose.END)

	return g.Compile(ctx)
}

func formatDocs(docs []*schema.Document) string {
	if len(docs) == 0 {
		return "（未检索到相关片段）"
	}
	var b strings.Builder
	for i, d := range docs {
		fmt.Fprintf(&b, "--- 片段 %d (%s) ---\n%s\n", i+1, docTitle(d), d.Content)
	}
	return b.String()
}
