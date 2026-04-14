package chunker

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/milabo0718/offer-pilot/backend/common/rag/loader"
	"github.com/milabo0718/offer-pilot/backend/model"
)

var markdownH2Pattern = regexp.MustCompile(`(?m)^##\s+(.+)$`)

// BuildChunks 按文档类型执行切块策略：Markdown 按二级标题，JSON 按数组项。
func BuildChunks(docs []loader.SourceDocument) ([]model.Chunk, []error) {
	chunks := make([]model.Chunk, 0)
	errs := make([]error, 0)

	for _, doc := range docs {
		switch doc.DocType {
		case "markdown":
			mdChunks, mdErr := buildMarkdownChunks(doc)
			if mdErr != nil {
				errs = append(errs, mdErr)
				continue
			}
			chunks = append(chunks, mdChunks...)
		case "json":
			jsonChunks, jsonErr := buildJSONChunks(doc)
			if jsonErr != nil {
				errs = append(errs, jsonErr)
				continue
			}
			chunks = append(chunks, jsonChunks...)
		default:
			errs = append(errs, fmt.Errorf("不支持的文档类型: %s", doc.DocType))
		}
	}

	return chunks, errs
}

func buildMarkdownChunks(doc loader.SourceDocument) ([]model.Chunk, error) {
	text := strings.TrimSpace(doc.Markdown)
	if text == "" {
		return nil, fmt.Errorf("Markdown 为空: %s", doc.SourceFile)
	}

	indices := markdownH2Pattern.FindAllStringSubmatchIndex(text, -1)
	if len(indices) == 0 {
		chunk := newChunk(doc.SourceFile, "full_document", text, nil, "")
		return []model.Chunk{chunk}, nil
	}

	chunks := make([]model.Chunk, 0, len(indices))
	for i, idx := range indices {
		heading := strings.TrimSpace(text[idx[2]:idx[3]])
		contentStart := idx[0]
		contentEnd := len(text)
		if i+1 < len(indices) {
			contentEnd = indices[i+1][0]
		}

		block := strings.TrimSpace(text[contentStart:contentEnd])
		if block == "" {
			continue
		}

		section := heading
		if section == "" {
			section = fmt.Sprintf("section_%d", i+1)
		}
		chunks = append(chunks, newChunk(doc.SourceFile, section, block, nil, ""))
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("Markdown 切块后无有效内容: %s", doc.SourceFile)
	}
	return chunks, nil
}

func buildJSONChunks(doc loader.SourceDocument) ([]model.Chunk, error) {
	if len(doc.JSONArray) == 0 {
		return nil, fmt.Errorf("JSON 数组为空: %s", doc.SourceFile)
	}

	chunks := make([]model.Chunk, 0, len(doc.JSONArray))
	for i, item := range doc.JSONArray {
		content := strings.TrimSpace(string(item))
		if content == "" {
			continue
		}

		itemTags, itemDifficulty := extractOptionalMeta(item)
		section := fmt.Sprintf("index_%d", i)
		chunks = append(chunks, newChunk(doc.SourceFile, section, content, itemTags, itemDifficulty))
	}

	if len(chunks) == 0 {
		return nil, fmt.Errorf("JSON 切块后无有效内容: %s", doc.SourceFile)
	}
	return chunks, nil
}

func extractOptionalMeta(item json.RawMessage) ([]string, string) {
	var obj map[string]interface{}
	if err := json.Unmarshal(item, &obj); err != nil {
		return nil, ""
	}

	tags := make([]string, 0)
	if rawTags, ok := obj["tags"]; ok {
		switch v := rawTags.(type) {
		case []interface{}:
			for _, tag := range v {
				if str, ok := tag.(string); ok {
					str = strings.TrimSpace(str)
					if str != "" {
						tags = append(tags, str)
					}
				}
			}
		case string:
			if v != "" {
				tags = append(tags, v)
			}
		}
	}

	difficulty := ""
	if rawDifficulty, ok := obj["difficulty"]; ok {
		if str, ok := rawDifficulty.(string); ok {
			difficulty = strings.TrimSpace(str)
		}
	}

	return tags, difficulty
}

func newChunk(sourceFile string, section string, content string, tags []string, difficulty string) model.Chunk {
	return model.Chunk{
		ID:      uuid.NewString(),
		Content: content,
		Metadata: model.ChunkMetadata{
			SourceFile:     sourceFile,
			SectionOrIndex: section,
			Tags:           tags,
			Difficulty:     difficulty,
		},
	}
}
