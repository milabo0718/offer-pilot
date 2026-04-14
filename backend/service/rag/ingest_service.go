package rag

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/milabo0718/offer-pilot/backend/common/rag/chunker"
	"github.com/milabo0718/offer-pilot/backend/common/rag/embedder"
	"github.com/milabo0718/offer-pilot/backend/common/rag/loader"
	"github.com/milabo0718/offer-pilot/backend/model"
)

type documentLoader func(dir string) ([]loader.SourceDocument, []error)
type chunkBuilder func(docs []loader.SourceDocument) ([]model.Chunk, []error)
type ingestProgress func(done, total, success, failed int)

type ingestStore interface {
	EnsureIndex(ctx context.Context) error
	UpsertBatch(ctx context.Context, chunks []model.Chunk, vectors [][]float32) (model.IngestStats, error)
	IndexName() string
	Ping(ctx context.Context) error
	SearchModuleReady(ctx context.Context) (bool, string)
	IndexExists(ctx context.Context) (bool, error)
}

// IngestService 负责把本地文档转换成可检索向量数据。
type IngestService struct {
	embedder  embedder.Embedder
	store     ingestStore
	batchSize int
	loadDocs  documentLoader
	build     chunkBuilder
	progress  ingestProgress
}

func NewIngestService(embedder embedder.Embedder, store ingestStore, batchSize int) *IngestService {
	if batchSize <= 0 {
		batchSize = 16
	}
	return &IngestService{
		embedder:  embedder,
		store:     store,
		batchSize: batchSize,
		loadDocs:  loader.LoadDocumentsFromDir,
		build:     chunker.BuildChunks,
	}
}

// SetTestHooks 仅用于单元测试注入假数据源与切块逻辑。
func (s *IngestService) SetTestHooks(loaderFn documentLoader, chunkFn chunkBuilder) {
	if loaderFn != nil {
		s.loadDocs = loaderFn
	}
	if chunkFn != nil {
		s.build = chunkFn
	}
}

// SetProgressCallback 设置入库进度回调（可选）。
func (s *IngestService) SetProgressCallback(cb ingestProgress) {
	s.progress = cb
}

func (s *IngestService) EnsureIndex(ctx context.Context) error {
	return s.store.EnsureIndex(ctx)
}

// Health 检查 RAG 依赖的 Redis 与索引状态。
func (s *IngestService) Health(ctx context.Context) model.RAGHealthStatus {
	status := model.RAGHealthStatus{
		IndexName: s.store.IndexName(),
	}

	if err := s.store.Ping(ctx); err != nil {
		status.Message = fmt.Sprintf("Redis 不可达: %v", err)
		return status
	}
	status.RedisReachable = true

	ready, msg := s.store.SearchModuleReady(ctx)
	status.RedisSearchReady = ready
	if !ready {
		status.Message = msg
		return status
	}

	exists, err := s.store.IndexExists(ctx)
	if err != nil {
		status.Message = err.Error()
		return status
	}
	status.IndexExists = exists

	if exists {
		status.Message = "RAG 健康，索引已就绪"
	} else {
		status.Message = "RAG 健康，但索引尚未创建"
	}

	return status
}

// IngestDirectory 扫描目录、切块、向量化并批量入库，返回基础统计信息。
func (s *IngestService) IngestDirectory(ctx context.Context, dir string) (model.IngestStats, error) {
	stats := model.IngestStats{}

	docs, loadErrs := s.loadDocs(dir)
	stats.TotalFiles = len(docs)
	for _, e := range loadErrs {
		stats.ErrorSamples = append(stats.ErrorSamples, e.Error())
	}

	chunks, chunkErrs := s.build(docs)
	for _, e := range chunkErrs {
		stats.ErrorSamples = append(stats.ErrorSamples, e.Error())
	}
	stats.TotalChunks = len(chunks)

	if len(chunks) == 0 {
		stats.FailedChunks = 0
		if len(stats.ErrorSamples) == 0 {
			stats.ErrorSamples = append(stats.ErrorSamples, "没有可入库的切块数据")
		}
		if len(stats.ErrorSamples) > 20 {
			stats.ErrorSamples = stats.ErrorSamples[:20]
		}
		return stats, nil
	}

	for i := 0; i < len(chunks); i += s.batchSize {
		end := i + s.batchSize
		if end > len(chunks) {
			end = len(chunks)
		}

		batch := chunks[i:end]
		texts := make([]string, 0, len(batch))
		for _, c := range batch {
			// 仅用于向量化的兜底截断，避免上游 Embedding API 因输入过长报错。
			// 不影响原始 chunk 内容入库。
			texts = append(texts, trimToMaxUTF8Bytes(strings.TrimSpace(c.Content), 8000))
		}

		vectors, err := s.embedder.EmbedDocuments(ctx, texts)
		if err != nil {
			stats.FailedChunks += len(batch)
			stats.ErrorSamples = append(stats.ErrorSamples, fmt.Sprintf("batch[%d:%d] 向量化失败: %v", i, end, err))
			if s.progress != nil {
				s.progress(end, len(chunks), stats.SuccessChunks, stats.FailedChunks)
			}
			continue
		}

		batchStats, err := s.store.UpsertBatch(ctx, batch, vectors)
		if err != nil {
			stats.FailedChunks += len(batch)
			stats.ErrorSamples = append(stats.ErrorSamples, fmt.Sprintf("batch[%d:%d] 入库失败: %v", i, end, err))
			if s.progress != nil {
				s.progress(end, len(chunks), stats.SuccessChunks, stats.FailedChunks)
			}
			continue
		}

		stats.SuccessChunks += batchStats.SuccessChunks
		stats.FailedChunks += batchStats.FailedChunks
		stats.ErrorSamples = append(stats.ErrorSamples, batchStats.ErrorSamples...)

		if s.progress != nil {
			s.progress(end, len(chunks), stats.SuccessChunks, stats.FailedChunks)
		}
	}

	// 控制错误样本数量，避免返回体过大。
	if len(stats.ErrorSamples) > 20 {
		stats.ErrorSamples = stats.ErrorSamples[:20]
	}

	if stats.SuccessChunks == 0 && strings.TrimSpace(strings.Join(stats.ErrorSamples, "")) != "" {
		return stats, fmt.Errorf("入库任务失败")
	}
	return stats, nil
}

// trimToMaxUTF8Bytes 将文本裁剪到最大字节数，并保证结果是有效 UTF-8 串。
func trimToMaxUTF8Bytes(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return "."
	}
	if len(s) <= maxBytes {
		if s == "" {
			return "."
		}
		return s
	}

	b := []byte(s[:maxBytes])
	for len(b) > 0 && !utf8.Valid(b) {
		b = b[:len(b)-1]
	}
	if len(b) == 0 {
		return "."
	}
	return string(b)
}
