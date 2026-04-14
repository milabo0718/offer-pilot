package chunker

import (
	"encoding/json"
	"testing"

	"github.com/milabo0718/offer-pilot/backend/common/rag/loader"
)

func TestBuildChunks_MarkdownByH2(t *testing.T) {
	docs := []loader.SourceDocument{
		{
			SourceFile: "sample.md",
			DocType:    "markdown",
			Markdown:   "# 标题\n\n## 第一节\n内容A\n\n## 第二节\n内容B",
		},
	}

	chunks, errs := BuildChunks(docs)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	if chunks[0].Metadata.SectionOrIndex != "第一节" {
		t.Fatalf("expected first section to be 第一节, got %s", chunks[0].Metadata.SectionOrIndex)
	}
	if chunks[1].Metadata.SectionOrIndex != "第二节" {
		t.Fatalf("expected second section to be 第二节, got %s", chunks[1].Metadata.SectionOrIndex)
	}

	for i, c := range chunks {
		if c.ID == "" {
			t.Fatalf("chunk %d id should not be empty", i)
		}
		if c.Metadata.SourceFile != "sample.md" {
			t.Fatalf("chunk %d source_file mismatch: %s", i, c.Metadata.SourceFile)
		}
	}
}

func TestBuildChunks_MarkdownFallbackFullDocument(t *testing.T) {
	docs := []loader.SourceDocument{
		{
			SourceFile: "plain.md",
			DocType:    "markdown",
			Markdown:   "只有普通正文，没有二级标题",
		},
	}

	chunks, errs := BuildChunks(docs)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Metadata.SectionOrIndex != "full_document" {
		t.Fatalf("expected full_document, got %s", chunks[0].Metadata.SectionOrIndex)
	}
}

func TestBuildChunks_JSONByArrayItemsAndMeta(t *testing.T) {
	item1, _ := json.Marshal(map[string]interface{}{
		"question":   "解释 channel 与 mutex",
		"tags":       []string{"go", "concurrency"},
		"difficulty": "medium",
	})
	item2, _ := json.Marshal(map[string]interface{}{
		"question": "解释 synchronized 与 ReentrantLock",
		"tags":     "java",
	})

	docs := []loader.SourceDocument{
		{
			SourceFile: "qa.json",
			DocType:    "json",
			JSONArray:  []json.RawMessage{item1, item2},
		},
	}

	chunks, errs := BuildChunks(docs)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	if chunks[0].Metadata.SectionOrIndex != "index_0" || chunks[1].Metadata.SectionOrIndex != "index_1" {
		t.Fatalf("unexpected section_or_index values: %s, %s", chunks[0].Metadata.SectionOrIndex, chunks[1].Metadata.SectionOrIndex)
	}

	if len(chunks[0].Metadata.Tags) != 2 {
		t.Fatalf("expected first chunk tags length 2, got %d", len(chunks[0].Metadata.Tags))
	}
	if chunks[0].Metadata.Difficulty != "medium" {
		t.Fatalf("expected difficulty medium, got %s", chunks[0].Metadata.Difficulty)
	}
	if len(chunks[1].Metadata.Tags) != 1 || chunks[1].Metadata.Tags[0] != "java" {
		t.Fatalf("expected second chunk tags to contain java, got %v", chunks[1].Metadata.Tags)
	}
}
