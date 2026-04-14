package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/milabo0718/offer-pilot/backend/common/rag/embedder"
	"github.com/milabo0718/offer-pilot/backend/model"
)

// vectorSearcher 抽象向量检索存储，方便业务层做单元测试。
type vectorSearcher interface {
	Search(ctx context.Context, queryVector []float32, topK int, filter *model.SearchFilter) ([]model.SearchResult, error)
}

// SearchService 封装查询向量化 + Redis KNN 检索流程。
type SearchService struct {
	embedder    embedder.Embedder
	store       vectorSearcher
	defaultTopK int
	maxTopK     int
}

func NewSearchService(embedder embedder.Embedder, store vectorSearcher, defaultTopK, maxTopK int) *SearchService {
	if defaultTopK <= 0 {
		defaultTopK = 5
	}
	if maxTopK <= 0 {
		maxTopK = 20
	}
	return &SearchService{
		embedder:    embedder,
		store:       store,
		defaultTopK: defaultTopK,
		maxTopK:     maxTopK,
	}
}

// SearchRelevantChunks 对外提供最小可用检索能力。
func (s *SearchService) SearchRelevantChunks(ctx context.Context, query string, topK int, filter *model.SearchFilter) ([]model.SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query 不能为空")
	}

	if topK <= 0 {
		topK = s.defaultTopK
	}
	if topK > s.maxTopK {
		topK = s.maxTopK
	}

	vector, err := s.embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query 向量化失败: %w", err)
	}

	results, err := s.store.Search(ctx, vector, topK, filter)
	if err != nil {
		return nil, err
	}
	return results, nil
}
