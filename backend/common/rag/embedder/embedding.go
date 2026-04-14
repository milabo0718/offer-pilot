package embedder

import (
	"context"
)

// Embedder 抽象向量化能力，允许在真实 API 与 Mock 之间切换。
type Embedder interface {
	Name() string
	Dimension() int
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}
