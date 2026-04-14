package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/milabo0718/offer-pilot/backend/common/rag/embedder"
	"github.com/milabo0718/offer-pilot/backend/common/rag/store"
	"github.com/milabo0718/offer-pilot/backend/common/redis"
	"github.com/milabo0718/offer-pilot/backend/config"
	ragservice "github.com/milabo0718/offer-pilot/backend/service/rag"
)

func main() {
	conf, err := config.InitConfig("./config")
	if err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	dir := flag.String("dir", conf.RagConfig.DefaultIngestDir, "待入库目录")
	batchSize := flag.Int("batch", conf.RagConfig.BatchSize, "批处理大小")
	useMock := flag.Bool("mock", conf.RagConfig.UseMockEmbedding, "是否启用Mock Embedding")
	flag.Parse()

	conf.RagConfig.UseMockEmbedding = *useMock
	if *batchSize > 0 {
		conf.RagConfig.BatchSize = *batchSize
	}

	// 阿里 text-embedding-v3 单次批量上限为 10，做命令行兜底避免 400。
	if !conf.RagConfig.UseMockEmbedding && conf.RagConfig.BatchSize > 10 {
		log.Printf("检测到真实 Embedding 批量=%d，超过接口上限，自动降为 10", conf.RagConfig.BatchSize)
		conf.RagConfig.BatchSize = 10
	}
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
	ingestService := ragservice.NewIngestService(emb, vStore, conf.RagConfig.BatchSize)
	ingestService.SetProgressCallback(func(done, total, success, failed int) {
		if total <= 0 {
			return
		}
		const width = 30
		ratio := float64(done) / float64(total)
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		filled := int(ratio * width)
		bar := strings.Repeat("=", filled)
		if filled < width {
			bar += ">" + strings.Repeat(" ", width-filled-1)
		}
		if filled >= width {
			bar = strings.Repeat("=", width)
		}

		fmt.Printf("\rbuilding vectors [%s] %6.2f%% (%d/%d) success=%d failed=%d", bar, ratio*100, done, total, success, failed)
		if done >= total {
			fmt.Print("\n")
		}
	})

	ctx := context.Background()
	if err = ingestService.EnsureIndex(ctx); err != nil {
		log.Fatalf("初始化索引失败: %v", err)
	}

	stats, err := ingestService.IngestDirectory(ctx, *dir)
	if err != nil {
		log.Printf("入库任务有失败: %v", err)
	}

	fmt.Printf("ingest done: files=%d total=%d success=%d failed=%d\n", stats.TotalFiles, stats.TotalChunks, stats.SuccessChunks, stats.FailedChunks)
	for _, sample := range stats.ErrorSamples {
		fmt.Printf("error: %s\n", sample)
	}
}
