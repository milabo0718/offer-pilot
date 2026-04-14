package embedder

import (
	"fmt"
	"strings"

	"github.com/milabo0718/offer-pilot/backend/config"
)

// NewFromConfig 根据配置创建向量化实现。
func NewFromConfig(conf config.RagConfig) (Embedder, error) {
	if conf.UseMockEmbedding {
		return NewMockEmbedder(conf.VectorDim), nil
	}

	provider := strings.ToLower(strings.TrimSpace(conf.EmbeddingProvider))
	switch provider {
	case "", "openai-compatible", "openai":
		return NewOpenAICompatibleEmbedder(conf.EmbeddingAPIKey, conf.EmbeddingBaseURL, conf.EmbeddingModelName, conf.VectorDim), nil
	case "mock":
		return NewMockEmbedder(conf.VectorDim), nil
	default:
		return nil, fmt.Errorf("不支持的 embedding provider: %s", conf.EmbeddingProvider)
	}
}
