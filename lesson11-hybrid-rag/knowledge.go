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

func loadKnowledgeChunks() ([]*schema.Document, error) {
	raw, err := os.ReadFile(demoKnowledgePath())
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(raw), "\n## ")
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

func docTitle(d *schema.Document) string {
	if d == nil || d.MetaData == nil {
		return ""
	}
	t, _ := d.MetaData["title"].(string)
	return t
}
