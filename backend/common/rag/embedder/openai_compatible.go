package embedder

import (
	"context"
	"fmt"
	"os"
	"strings"

	einoOpenAIEmbedding "github.com/cloudwego/eino-ext/components/embedding/openai"
)

// OpenAICompatibleEmbedder 对接 OpenAI 兼容 Embedding API。
type OpenAICompatibleEmbedder struct {
	apiKey  string
	baseURL string
	model   string
	dim     int
	client  *einoOpenAIEmbedding.Embedder
}

func NewOpenAICompatibleEmbedder(apiKey, baseURL, model string, dim int) *OpenAICompatibleEmbedder {
	baseURL = strings.TrimRight(baseURL, "/")
	if dim <= 0 {
		dim = 1536
	}
	if strings.TrimSpace(apiKey) == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
	}
	if strings.TrimSpace(model) == "" {
		model = strings.TrimSpace(os.Getenv("OPENAI_MODEL_NAME"))
	}

	return &OpenAICompatibleEmbedder{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		dim:     dim,
	}
}

func (e *OpenAICompatibleEmbedder) Name() string { return "openai-compatible" }

func (e *OpenAICompatibleEmbedder) Dimension() int { return e.dim }

func (e *OpenAICompatibleEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	client, err := e.getOrInitClient(ctx)
	if err != nil {
		return nil, err
	}

	vectors64, err := client.EmbedStrings(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("调用 e ino embedding 失败: %w", err)
	}

	vectors := make([][]float32, 0, len(vectors64))
	for i, vector64 := range vectors64 {
		if len(vector64) == 0 {
			return nil, fmt.Errorf("embedding 返回为空，index=%d", i)
		}
		vectors = append(vectors, float64To32(vector64))
	}

	return vectors, nil
}

func (e *OpenAICompatibleEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	vectors, err := e.EmbedDocuments(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("embedding 返回为空")
	}
	return vectors[0], nil
}

func (e *OpenAICompatibleEmbedder) getOrInitClient(ctx context.Context) (*einoOpenAIEmbedding.Embedder, error) {
	if e.client != nil {
		return e.client, nil
	}

	if strings.TrimSpace(e.apiKey) == "" {
		return nil, fmt.Errorf("embedding API Key 为空")
	}
	if strings.TrimSpace(e.model) == "" {
		return nil, fmt.Errorf("embedding model 为空")
	}
	if strings.TrimSpace(e.baseURL) == "" {
		return nil, fmt.Errorf("embedding BaseURL 为空")
	}

	client, err := einoOpenAIEmbedding.NewEmbedder(ctx, &einoOpenAIEmbedding.EmbeddingConfig{
		APIKey:  e.apiKey,
		BaseURL: e.baseURL,
		Model:   e.model,
	})
	if err != nil {
		return nil, fmt.Errorf("初始化 e ino embedding 客户端失败: %w", err)
	}
	e.client = client
	return e.client, nil
}

func float64To32(vector []float64) []float32 {
	out := make([]float32, 0, len(vector))
	for _, v := range vector {
		out = append(out, float32(v))
	}
	return out
}
