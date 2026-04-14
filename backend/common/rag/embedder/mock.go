package embedder

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
)

// MockEmbedder 用确定性算法生成向量，保证本地无外部依赖也能跑通流程。
type MockEmbedder struct {
	dim int
}

func NewMockEmbedder(dim int) *MockEmbedder {
	if dim <= 0 {
		dim = 1536
	}
	return &MockEmbedder{dim: dim}
}

func (m *MockEmbedder) Name() string { return "mock" }

func (m *MockEmbedder) Dimension() int { return m.dim }

func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	vectors := make([][]float32, 0, len(texts))
	for _, t := range texts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		vectors = append(vectors, m.embedText(t))
	}
	return vectors, nil
}

func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return m.embedText(text), nil
}

func (m *MockEmbedder) embedText(text string) []float32 {
	vector := make([]float32, m.dim)
	var norm float64
	for i := 0; i < m.dim; i++ {
		h := fnv.New64a()
		_, _ = h.Write([]byte(fmt.Sprintf("%s#%d", text, i)))
		v := float64(h.Sum64()%2000000)/1000000.0 - 1.0
		vector[i] = float32(v)
		norm += v * v
	}

	if norm == 0 {
		return vector
	}

	norm = math.Sqrt(norm)
	for i := range vector {
		vector[i] = float32(float64(vector[i]) / norm)
	}
	return vector
}
