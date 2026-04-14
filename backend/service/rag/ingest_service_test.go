package rag

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/milabo0718/offer-pilot/backend/common/rag/loader"
	"github.com/milabo0718/offer-pilot/backend/model"
)

type fakeBatchEmbedder struct {
	dim int
}

func (f *fakeBatchEmbedder) Name() string { return "fake-batch" }

func (f *fakeBatchEmbedder) Dimension() int { return f.dim }

func (f *fakeBatchEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	_ = ctx
	vectors := make([][]float32, 0, len(texts))
	for range texts {
		vectors = append(vectors, []float32{0.1, 0.2, 0.3})
	}
	return vectors, nil
}

func (f *fakeBatchEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	_ = ctx
	_ = text
	return []float32{0.1, 0.2, 0.3}, nil
}

type fakeIngestStore struct {
	ensureErr error
	upsertErr error
	calls     int
}

func (f *fakeIngestStore) IndexName() string { return "idx:test" }

func (f *fakeIngestStore) Ping(ctx context.Context) error {
	_ = ctx
	return nil
}

func (f *fakeIngestStore) SearchModuleReady(ctx context.Context) (bool, string) {
	_ = ctx
	return true, "ok"
}

func (f *fakeIngestStore) IndexExists(ctx context.Context) (bool, error) {
	_ = ctx
	return true, nil
}

func (f *fakeIngestStore) EnsureIndex(ctx context.Context) error {
	_ = ctx
	return f.ensureErr
}

func (f *fakeIngestStore) UpsertBatch(ctx context.Context, chunks []model.Chunk, vectors [][]float32) (model.IngestStats, error) {
	_ = ctx
	_ = vectors
	f.calls++
	if f.upsertErr != nil {
		return model.IngestStats{}, f.upsertErr
	}
	return model.IngestStats{
		TotalChunks:   len(chunks),
		SuccessChunks: len(chunks),
		FailedChunks:  0,
	}, nil
}

func buildFakeChunks(n int) []model.Chunk {
	chunks := make([]model.Chunk, 0, n)
	for i := 0; i < n; i++ {
		chunks = append(chunks, model.Chunk{
			ID:      uuid.NewString(),
			Content: fmt.Sprintf("chunk-%d", i),
			Metadata: model.ChunkMetadata{
				SourceFile:     "fake.md",
				SectionOrIndex: fmt.Sprintf("index_%d", i),
			},
		})
	}
	return chunks
}

func TestIngestDirectory_AllSuccess(t *testing.T) {
	store := &fakeIngestStore{}
	svc := NewIngestService(&fakeBatchEmbedder{dim: 3}, store, 2)

	svc.SetTestHooks(
		func(dir string) ([]loader.SourceDocument, []error) {
			_ = dir
			return []loader.SourceDocument{{SourceFile: "fake.md", DocType: "markdown", Markdown: "## a\nhello"}}, nil
		},
		func(docs []loader.SourceDocument) ([]model.Chunk, []error) {
			_ = docs
			return buildFakeChunks(3), nil
		},
	)

	stats, err := svc.IngestDirectory(context.Background(), "./any")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalChunks != 3 || stats.SuccessChunks != 3 || stats.FailedChunks != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if store.calls != 2 {
		t.Fatalf("expected 2 batches, got %d", store.calls)
	}
}

func TestIngestDirectory_PartialFailure(t *testing.T) {
	store := &fakeIngestStore{upsertErr: fmt.Errorf("redis down")}
	svc := NewIngestService(&fakeBatchEmbedder{dim: 3}, store, 2)

	svc.SetTestHooks(
		func(dir string) ([]loader.SourceDocument, []error) {
			_ = dir
			return []loader.SourceDocument{{SourceFile: "fake.md", DocType: "markdown", Markdown: "## a\nhello"}}, nil
		},
		func(docs []loader.SourceDocument) ([]model.Chunk, []error) {
			_ = docs
			return buildFakeChunks(3), nil
		},
	)

	stats, err := svc.IngestDirectory(context.Background(), "./any")
	if err == nil {
		t.Fatalf("expected error when all batches fail")
	}
	if stats.SuccessChunks != 0 || stats.FailedChunks != 3 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if len(stats.ErrorSamples) == 0 {
		t.Fatalf("expected error samples")
	}
}

func TestIngestDirectory_ErrorSamplesCapped(t *testing.T) {
	store := &fakeIngestStore{}
	svc := NewIngestService(&fakeBatchEmbedder{dim: 3}, store, 1)

	svc.SetTestHooks(
		func(dir string) ([]loader.SourceDocument, []error) {
			_ = dir
			errs := make([]error, 0, 30)
			for i := 0; i < 30; i++ {
				errs = append(errs, fmt.Errorf("load err %d", i))
			}
			return []loader.SourceDocument{}, errs
		},
		func(docs []loader.SourceDocument) ([]model.Chunk, []error) {
			_ = docs
			return []model.Chunk{}, nil
		},
	)

	stats, err := svc.IngestDirectory(context.Background(), "./any")
	if err != nil {
		t.Fatalf("did not expect hard error when no chunks: %v", err)
	}
	if len(stats.ErrorSamples) != 20 {
		t.Fatalf("expected capped error samples length 20, got %d", len(stats.ErrorSamples))
	}
}
