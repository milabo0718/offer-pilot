package store

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
	"github.com/milabo0718/offer-pilot/backend/config"
	"github.com/milabo0718/offer-pilot/backend/model"
)

// RedisVectorStore 封装 Redis Stack Vector Search 的核心操作。
type RedisVectorStore struct {
	client         *redis.Client
	indexName      string
	keyPrefix      string
	vectorField    string
	vectorDim      int
	distanceMetric string
}

func (s *RedisVectorStore) IndexName() string {
	return s.indexName
}

// Ping 检查 Redis 是否可达。
func (s *RedisVectorStore) Ping(ctx context.Context) error {
	return s.client.Ping(ctx).Err()
}

// SearchModuleReady 检查 RediSearch 能力是否可用。
func (s *RedisVectorStore) SearchModuleReady(ctx context.Context) (bool, string) {
	_, err := s.client.Do(ctx, "FT._LIST").Result()
	if err == nil {
		return true, "RediSearch 可用"
	}

	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "unknown command") || strings.Contains(errMsg, "ft._list") {
		return false, "当前 Redis 未启用 RediSearch（请使用 Redis Stack）"
	}

	return false, fmt.Sprintf("检查 RediSearch 能力失败: %v", err)
}

// IndexExists 检查目标向量索引是否存在。
func (s *RedisVectorStore) IndexExists(ctx context.Context) (bool, error) {
	_, err := s.client.Do(ctx, "FT.INFO", s.indexName).Result()
	if err == nil {
		return true, nil
	}

	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "unknown index name") {
		return false, nil
	}
	return false, fmt.Errorf("查询索引状态失败: %w", err)
}

func NewRedisVectorStore(client *redis.Client, conf config.RagConfig) *RedisVectorStore {
	keyPrefix := conf.KeyPrefix
	if keyPrefix == "" {
		keyPrefix = "rag:chunk:"
	}
	vectorField := conf.VectorField
	if vectorField == "" {
		vectorField = "vector"
	}
	metric := strings.ToUpper(strings.TrimSpace(conf.DistanceMetric))
	if metric == "" {
		metric = "COSINE"
	}
	return &RedisVectorStore{
		client:         client,
		indexName:      conf.IndexName,
		keyPrefix:      keyPrefix,
		vectorField:    vectorField,
		vectorDim:      conf.VectorDim,
		distanceMetric: metric,
	}
}

// EnsureIndex 确保索引存在，不存在才创建，存在则直接跳过。
func (s *RedisVectorStore) EnsureIndex(ctx context.Context) error {
	_, err := s.client.Do(ctx, "FT.INFO", s.indexName).Result()
	if err == nil {
		return nil
	}

	if !strings.Contains(strings.ToLower(err.Error()), "unknown index name") {
		return fmt.Errorf("查询索引信息失败: %w", err)
	}

	args := []interface{}{
		"FT.CREATE", s.indexName,
		"ON", "HASH",
		"PREFIX", 1, s.keyPrefix,
		"SCHEMA",
		"content", "TEXT",
		"source_file", "TAG",
		"section_or_index", "TAG",
		"tags", "TAG",
		"difficulty", "TAG",
		s.vectorField, "VECTOR", "HNSW", 6,
		"TYPE", "FLOAT32",
		"DIM", s.vectorDim,
		"DISTANCE_METRIC", s.distanceMetric,
	}

	if _, err = s.client.Do(ctx, args...).Result(); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "index already exists") {
			return nil
		}
		return fmt.Errorf("创建索引失败: %w", err)
	}
	return nil
}

// UpsertBatch 批量写入 chunk 与向量。
func (s *RedisVectorStore) UpsertBatch(ctx context.Context, chunks []model.Chunk, vectors [][]float32) (model.IngestStats, error) {
	stats := model.IngestStats{TotalChunks: len(chunks)}
	if len(chunks) != len(vectors) {
		return stats, fmt.Errorf("chunks 与 vectors 数量不一致")
	}

	pipe := s.client.Pipeline()
	validChunks := make([]model.Chunk, 0, len(chunks))
	for i := range chunks {
		if len(vectors[i]) != s.vectorDim {
			stats.FailedChunks++
			stats.ErrorSamples = append(stats.ErrorSamples, fmt.Sprintf("向量维度不匹配 chunk=%s", chunks[i].ID))
			continue
		}

		blob, err := vectorToBytes(vectors[i])
		if err != nil {
			stats.FailedChunks++
			stats.ErrorSamples = append(stats.ErrorSamples, fmt.Sprintf("向量序列化失败 chunk=%s err=%v", chunks[i].ID, err))
			continue
		}

		tagString := sanitizeTags(chunks[i].Metadata.Tags)
		key := s.keyPrefix + chunks[i].ID
		values := map[string]interface{}{
			"id":               chunks[i].ID,
			"content":          chunks[i].Content,
			"source_file":      chunks[i].Metadata.SourceFile,
			"section_or_index": chunks[i].Metadata.SectionOrIndex,
			"tags":             tagString,
			"difficulty":       chunks[i].Metadata.Difficulty,
			s.vectorField:      blob,
		}
		pipe.HSet(ctx, key, values)
		validChunks = append(validChunks, chunks[i])
	}

	results, err := pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return stats, fmt.Errorf("批量入库执行失败: %w", err)
	}

	for i, cmd := range results {
		if i >= len(validChunks) {
			break
		}
		if cmd.Err() == nil {
			stats.SuccessChunks++
		} else {
			stats.FailedChunks++
			stats.ErrorSamples = append(stats.ErrorSamples, fmt.Sprintf("Redis 写入失败 chunk=%s err=%v", validChunks[i].ID, cmd.Err()))
		}
	}

	return stats, nil
}

// Search 执行 KNN 向量检索，并返回 chunk 内容和元数据。
func (s *RedisVectorStore) Search(ctx context.Context, queryVector []float32, topK int, filter *model.SearchFilter) ([]model.SearchResult, error) {
	blob, err := vectorToBytes(queryVector)
	if err != nil {
		return nil, fmt.Errorf("查询向量序列化失败: %w", err)
	}

	baseQuery := "*"
	if filter != nil {
		clauses := make([]string, 0)
		if strings.TrimSpace(filter.SourceFile) != "" {
			clauses = append(clauses, fmt.Sprintf("@source_file:{%s}", escapeTagValue(filter.SourceFile)))
		}
		if len(filter.Tags) > 0 {
			tags := make([]string, 0, len(filter.Tags))
			for _, t := range filter.Tags {
				clean := strings.TrimSpace(t)
				if clean != "" {
					tags = append(tags, escapeTagValue(clean))
				}
			}
			if len(tags) > 0 {
				clauses = append(clauses, fmt.Sprintf("@tags:{%s}", strings.Join(tags, "|")))
			}
		}
		if len(clauses) > 0 {
			baseQuery = strings.Join(clauses, " ")
		}
	}

	query := fmt.Sprintf("(%s)=>[KNN %d @%s $BLOB AS score]", baseQuery, topK, s.vectorField)
	args := []interface{}{
		"FT.SEARCH", s.indexName, query,
		"PARAMS", 2, "BLOB", blob,
		"SORTBY", "score", "ASC",
		"RETURN", 6, "content", "source_file", "section_or_index", "tags", "difficulty", "score",
		"DIALECT", 2,
	}

	raw, err := s.client.Do(ctx, args...).Result()
	if err != nil {
		return nil, fmt.Errorf("执行向量检索失败: %w", err)
	}

	results, err := parseSearchResults(raw, s.keyPrefix)
	if err != nil {
		return nil, err
	}
	return results, nil
}

func vectorToBytes(vector []float32) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, vector); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func parseSearchResults(raw interface{}, keyPrefix string) ([]model.SearchResult, error) {
	items, ok := raw.([]interface{})
	if !ok || len(items) < 1 {
		return []model.SearchResult{}, nil
	}

	results := make([]model.SearchResult, 0)
	for i := 1; i+1 < len(items); i += 2 {
		key := stringify(items[i])
		fieldList, ok := items[i+1].([]interface{})
		if !ok {
			continue
		}

		kv := make(map[string]string)
		for j := 0; j+1 < len(fieldList); j += 2 {
			kv[stringify(fieldList[j])] = stringify(fieldList[j+1])
		}

		score, _ := strconv.ParseFloat(kv["score"], 64)
		chunkID := strings.TrimPrefix(key, keyPrefix)
		if idField := strings.TrimSpace(kv["id"]); idField != "" {
			chunkID = idField
		}

		tags := make([]string, 0)
		if kv["tags"] != "" {
			for _, t := range strings.Split(kv["tags"], ",") {
				clean := strings.TrimSpace(t)
				if clean != "" {
					tags = append(tags, clean)
				}
			}
		}

		results = append(results, model.SearchResult{
			ChunkID: chunkID,
			Content: kv["content"],
			Metadata: model.ChunkMetadata{
				SourceFile:     kv["source_file"],
				SectionOrIndex: kv["section_or_index"],
				Tags:           tags,
				Difficulty:     kv["difficulty"],
			},
			Score: score,
		})
	}

	return results, nil
}

func stringify(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func sanitizeTags(tags []string) string {
	clean := make([]string, 0, len(tags))
	for _, tag := range tags {
		t := strings.TrimSpace(tag)
		if t == "" {
			continue
		}
		t = strings.ReplaceAll(t, ",", "_")
		clean = append(clean, t)
	}
	return strings.Join(clean, ",")
}

func escapeTagValue(v string) string {
	replacer := strings.NewReplacer(
		"-", "\\-",
		".", "\\.",
		" ", "\\ ",
		":", "\\:",
		"/", "\\/",
	)
	return replacer.Replace(v)
}
