package rag

import (
	"context"

	"github.com/milabo0718/offer-pilot/backend/model"
)

// Service 是对外统一的 RAG 服务门面，便于后续接入聊天链路时只依赖这一层。
type Service struct {
	ingest *IngestService
	search *SearchService
}

func NewService(ingest *IngestService, search *SearchService) *Service {
	return &Service{
		ingest: ingest,
		search: search,
	}
}

func (s *Service) EnsureIndex(ctx context.Context) error {
	return s.ingest.EnsureIndex(ctx)
}

func (s *Service) IngestDirectory(ctx context.Context, dir string) (model.IngestStats, error) {
	return s.ingest.IngestDirectory(ctx, dir)
}

// SearchRelevantChunks 满足最小签名要求：仅 query + topK。
func (s *Service) SearchRelevantChunks(ctx context.Context, query string, topK int) ([]model.SearchResult, error) {
	return s.search.SearchRelevantChunks(ctx, query, topK, nil)
}

// SearchRelevantChunksWithFilter 提供可选过滤能力。
func (s *Service) SearchRelevantChunksWithFilter(ctx context.Context, query string, topK int, filter *model.SearchFilter) ([]model.SearchResult, error) {
	return s.search.SearchRelevantChunks(ctx, query, topK, filter)
}
