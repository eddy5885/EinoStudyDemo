package main

import (
	"context"
	"strings"
	"unicode"

	"github.com/cloudwego/eino/components/retriever"
	flowrouter "github.com/cloudwego/eino/flow/retriever/router"
	"github.com/cloudwego/eino/schema"
)

// newHybridRetriever：并发跑多个 retriever，再用 RRF 融合排序。
//
// 为什么需要这一层：
// - 关键词检索：对「NeoStack」「7天」这类明确关键词很稳，但中文口语问法容易 miss
// - 向量检索：对中文语义更稳，但有时会把“背景说明段”排在制度段前面
// - Hybrid：两边都跑，然后融合，通常更稳
func newHybridRetriever(ctx context.Context) retriever.Retriever {
	_ = ensureKeywordIndexed(ctx)
	_ = ensureVectorIndexed(ctx)

	kw := getKeywordStore()
	vec := getVectorStore(ctx)

	// 给 doc 增加来源标记，便于观察融合效果
	wrapSource := func(src string, base retriever.Retriever) retriever.Retriever {
		return &sourceRetriever{src: src, base: base}
	}

	r, err := flowrouter.NewRetriever(ctx, &flowrouter.Config{
		Retrievers: map[string]retriever.Retriever{
			"keyword": wrapSource("keyword", kw),
			"vector":  wrapSource("vector", vec),
		},
		Router: func(_ context.Context, query string) ([]string, error) {
			// 很简单的路由规则：含中文就两边都跑（更稳），纯英文/数字只跑 keyword 更省钱。
			if hasCJK(query) {
				return []string{"keyword", "vector"}, nil
			}
			return []string{"keyword"}, nil
		},
	})
	if err != nil {
		// 这个 demo 里不返回 err，避免改动签名；直接退化为向量检索
		return vec
	}
	return r
}

type sourceRetriever struct {
	src  string
	base retriever.Retriever
}

func (s *sourceRetriever) Retrieve(ctx context.Context, q string, opts ...retriever.Option) ([]*schema.Document, error) {
	docs, err := s.base.Retrieve(ctx, q, opts...)
	if err != nil {
		return nil, err
	}
	for _, d := range docs {
		if d == nil {
			continue
		}
		if d.MetaData == nil {
			d.MetaData = map[string]any{}
		}
		d.MetaData["source"] = s.src
	}
	return docs, nil
}

func hasCJK(s string) bool {
	for _, r := range strings.TrimSpace(s) {
		// Han/Hiragana/Katakana/Hangul 大致覆盖中日韩语系
		if unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul) {
			return true
		}
	}
	return false
}

