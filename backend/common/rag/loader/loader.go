package loader

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SourceDocument 是统一后的文档结构，供切块模块消费。
type SourceDocument struct {
	SourceFile string
	DocType    string
	Markdown   string
	JSONArray  []json.RawMessage
}

// LoadDocumentsFromDir 扫描目录下的 md/json 文件并加载到内存。
func LoadDocumentsFromDir(dir string) ([]SourceDocument, []error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, []error{fmt.Errorf("读取目录失败: %w", err)}
	}

	docs := make([]SourceDocument, 0)
	errs := make([]error, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		ext := strings.ToLower(filepath.Ext(fileName))
		absPath := filepath.Join(dir, fileName)

		switch ext {
		case ".md":
			doc, loadErr := loadMarkdown(absPath)
			if loadErr != nil {
				errs = append(errs, loadErr)
				continue
			}
			docs = append(docs, doc)
		case ".json":
			doc, loadErr := loadJSONArray(absPath)
			if loadErr != nil {
				errs = append(errs, loadErr)
				continue
			}
			docs = append(docs, doc)
		}
	}

	return docs, errs
}

func loadMarkdown(path string) (SourceDocument, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return SourceDocument{}, fmt.Errorf("读取 Markdown 失败 %s: %w", path, err)
	}

	content := strings.TrimSpace(string(bytes))
	if content == "" {
		return SourceDocument{}, fmt.Errorf("Markdown 内容为空: %s", path)
	}

	return SourceDocument{
		SourceFile: filepath.Base(path),
		DocType:    "markdown",
		Markdown:   content,
	}, nil
}

func loadJSONArray(path string) (SourceDocument, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return SourceDocument{}, fmt.Errorf("读取 JSON 失败 %s: %w", path, err)
	}

	var arr []json.RawMessage
	if err = json.Unmarshal(bytes, &arr); err != nil {
		return SourceDocument{}, fmt.Errorf("JSON 必须是数组结构 %s: %w", path, err)
	}
	if len(arr) == 0 {
		return SourceDocument{}, fmt.Errorf("JSON 数组为空: %s", path)
	}

	return SourceDocument{
		SourceFile: filepath.Base(path),
		DocType:    "json",
		JSONArray:  arr,
	}, nil
}
