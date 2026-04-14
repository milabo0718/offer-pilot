package model

// ChunkMetadata 保存切块来源和可选标签信息，后续检索时可以用于过滤。
type ChunkMetadata struct {
	SourceFile     string   `json:"source_file"`
	SectionOrIndex string   `json:"section_or_index"`
	Tags           []string `json:"tags,omitempty"`
	Difficulty     string   `json:"difficulty,omitempty"`
}

// Chunk 表示一个可向量化并存储到向量库的最小语义单元。
type Chunk struct {
	ID       string        `json:"id"`
	Content  string        `json:"content"`
	Metadata ChunkMetadata `json:"metadata"`
}

// SearchFilter 是最小可用过滤条件。
type SearchFilter struct {
	SourceFile string   `json:"source_file,omitempty"`
	Tags       []string `json:"tags,omitempty"`
}

// SearchResult 是向量检索返回项。
type SearchResult struct {
	ChunkID  string        `json:"chunk_id"`
	Content  string        `json:"content"`
	Metadata ChunkMetadata `json:"metadata"`
	Score    float64       `json:"score"`
}

// IngestStats 用于记录一次入库任务的基础统计数据。
type IngestStats struct {
	TotalFiles    int      `json:"total_files"`
	TotalChunks   int      `json:"total_chunks"`
	SuccessChunks int      `json:"success_chunks"`
	FailedChunks  int      `json:"failed_chunks"`
	ErrorSamples  []string `json:"error_samples,omitempty"`
}

// RAGSearchRequest 是检索接口请求体。
type RAGSearchRequest struct {
	Query  string       `json:"query" binding:"required"`
	TopK   int          `json:"topK,omitempty"`
	Filter SearchFilter `json:"filter,omitempty"`
}

// RAGSearchResponse 是检索接口返回体。
type RAGSearchResponse struct {
	Results []SearchResult `json:"results"`
}

// RAGIngestRequest 是手动触发入库的请求体。
type RAGIngestRequest struct {
	Directory string `json:"directory,omitempty"`
}

// RAGIngestResponse 是手动触发入库的返回体。
type RAGIngestResponse struct {
	Stats IngestStats `json:"stats"`
}

// RAGHealthStatus 是 RAG 运行环境健康检查结果。
type RAGHealthStatus struct {
	RedisReachable   bool   `json:"redis_reachable"`
	RedisSearchReady bool   `json:"redis_search_ready"`
	IndexExists      bool   `json:"index_exists"`
	IndexName        string `json:"index_name"`
	Message          string `json:"message"`
}
