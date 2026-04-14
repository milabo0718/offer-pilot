package main

import (
	"context"
	"fmt"
	"log"

	"github.com/milabo0718/offer-pilot/backend/common/rag/embedder"
	"github.com/milabo0718/offer-pilot/backend/common/rag/store"
	"github.com/milabo0718/offer-pilot/backend/common/redis"
	"github.com/milabo0718/offer-pilot/backend/config"
	ragservice "github.com/milabo0718/offer-pilot/backend/service/rag"
)

// 这是最小演示命令：初始化索引 -> 入库 -> 检索并打印结果。
func main() {
	conf, err := config.InitConfig("./config")
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	// 演示默认走 Mock，保证本地在无外部 API 时也能跑通。
	conf.RagConfig.UseMockEmbedding = true

	host, port := conf.RagConfig.RedisHostPort()
	rdb, err := redis.NewRedisClient(&config.RedisConfig{
		RedisHost:     host,
		RedisPort:     port,
		RedisPassword: conf.RagConfig.RedisPassword,
		RedisDb:       conf.RagConfig.RedisDB,
	})
	if err != nil {
		log.Fatalf("初始化 Redis 失败: %v", err)
	}

	emb, err := embedder.NewFromConfig(conf.RagConfig)
	if err != nil {
		log.Fatalf("初始化 Embedding 失败: %v", err)
	}

	vStore := store.NewRedisVectorStore(rdb, conf.RagConfig)
	ingest := ragservice.NewIngestService(emb, vStore, conf.RagConfig.BatchSize)
	search := ragservice.NewSearchService(emb, vStore, conf.RagConfig.DefaultTopK, conf.RagConfig.MaxTopK)
	svc := ragservice.NewService(ingest, search)

	ctx := context.Background()
	if err = svc.EnsureIndex(ctx); err != nil {
		log.Fatalf("初始化索引失败: %v", err)
	}

	stats, err := svc.IngestDirectory(ctx, conf.RagConfig.DefaultIngestDir)
	if err != nil {
		log.Printf("入库任务包含失败: %v", err)
	}
	fmt.Printf("ingest: files=%d chunks=%d success=%d failed=%d\n", stats.TotalFiles, stats.TotalChunks, stats.SuccessChunks, stats.FailedChunks)

	results, err := svc.SearchRelevantChunks(ctx, "Golang 并发 channel 与 mutex", 3)
	if err != nil {
		log.Fatalf("检索失败: %v", err)
	}

	fmt.Println("search results:")
	for i, item := range results {
		fmt.Printf("%d) score=%.6f source=%s section=%s\n", i+1, item.Score, item.Metadata.SourceFile, item.Metadata.SectionOrIndex)
		fmt.Printf("   content: %s\n", item.Content)
	}
}
