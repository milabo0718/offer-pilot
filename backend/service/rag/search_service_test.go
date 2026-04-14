package rag

import (
	"context"
	"testing"

	"github.com/milabo0718/offer-pilot/backend/model"
)

type fakeEmbedder struct {
	vector []float32
	err    error
}

func (f *fakeEmbedder) Name() string { return "fake" }

func (f *fakeEmbedder) Dimension() int { return len(f.vector) }

func (f *fakeEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	_ = ctx
	_ = texts
	return nil, nil
}

func (f *fakeEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	if f.err != nil {
		return nil, f.err
	}
	return f.vector, nil
}

type fakeStore struct {
	lastTopK   int
	lastFilter *model.SearchFilter
	results    []model.SearchResult
	err        error
}

func (f *fakeStore) Search(ctx context.Context, queryVector []float32, topK int, filter *model.SearchFilter) ([]model.SearchResult, error) {
	_ = ctx
	_ = queryVector
	f.lastTopK = topK
	f.lastFilter = filter
	if f.err != nil {
		return nil, f.err
	}
	return f.results, nil
}

func TestSearchRelevantChunks_EmptyQuery(t *testing.T) {
	svc := NewSearchService(&fakeEmbedder{vector: []float32{0.1, 0.2}}, &fakeStore{}, 5, 20)

	_, err := svc.SearchRelevantChunks(context.Background(), "   ", 3, nil)
	if err == nil {
		t.Fatalf("expected error for empty query")
	}
}

func TestSearchRelevantChunks_UseDefaultTopKWhenInvalid(t *testing.T) {
	store := &fakeStore{results: []model.SearchResult{}}
	svc := NewSearchService(&fakeEmbedder{vector: []float32{0.1, 0.2}}, store, 5, 20)

	_, err := svc.SearchRelevantChunks(context.Background(), "golang 并发", 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.lastTopK != 5 {
		t.Fatalf("expected default topK=5, got %d", store.lastTopK)
	}
}

func TestSearchRelevantChunks_ClampTopKToMax(t *testing.T) {
	store := &fakeStore{results: []model.SearchResult{}}
	svc := NewSearchService(&fakeEmbedder{vector: []float32{0.1, 0.2}}, store, 5, 20)

	_, err := svc.SearchRelevantChunks(context.Background(), "golang 并发", 999, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.lastTopK != 20 {
		t.Fatalf("expected clamped topK=20, got %d", store.lastTopK)
	}
}

func TestSearchRelevantChunks_PassFilter(t *testing.T) {
	store := &fakeStore{results: []model.SearchResult{}}
	svc := NewSearchService(&fakeEmbedder{vector: []float32{0.1, 0.2}}, store, 5, 20)

	filter := &model.SearchFilter{SourceFile: "sample_golang_backend.md", Tags: []string{"go"}}
	_, err := svc.SearchRelevantChunks(context.Background(), "golang 并发", 3, filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.lastFilter == nil || store.lastFilter.SourceFile != "sample_golang_backend.md" {
		t.Fatalf("expected filter to be passed through")
	}
}
