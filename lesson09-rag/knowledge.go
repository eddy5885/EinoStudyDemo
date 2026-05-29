package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func demoKnowledgePath() string {
	return filepath.Join("demo", "knowledge.md")
}

// loadKnowledgeChunks 按 Markdown 二级标题切分为 Document。
func loadKnowledgeChunks() ([]*schema.Document, error) {
	raw, err := os.ReadFile(demoKnowledgePath())
	if err != nil {
		return nil, err
	}
	text := string(raw)
	parts := strings.Split(text, "\n## ")
	docs := make([]*schema.Document, 0, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i > 0 {
			part = "## " + part
		}
		title := part
		if idx := strings.IndexByte(part, '\n'); idx >= 0 {
			title = strings.TrimSpace(part[:idx])
			title = strings.TrimPrefix(title, "## ")
		}
		docs = append(docs, &schema.Document{
			ID:      docID(part),
			Content: part,
			MetaData: map[string]any{
				"title": title,
			},
		})
	}
	return docs, nil
}
