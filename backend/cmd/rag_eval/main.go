package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/milabo0718/offer-pilot/backend/common/rag/embedder"
	"github.com/milabo0718/offer-pilot/backend/common/rag/store"
	"github.com/milabo0718/offer-pilot/backend/common/redis"
	"github.com/milabo0718/offer-pilot/backend/config"
	ragservice "github.com/milabo0718/offer-pilot/backend/service/rag"
)

type sample struct {
	ID           string
	Query        string
	ExpectedTags []string
}

type hitEvidence struct {
	Rank     int      `json:"rank"`
	Score    float64  `json:"score"`
	Tags     []string `json:"tags"`
	Repo     string   `json:"repo"`
	Question string   `json:"question"`
}

type row struct {
	ID           string      `json:"id"`
	Query        string      `json:"query"`
	ExpectedTags []string    `json:"expected_tags"`
	HitAt1       bool        `json:"hit_at_1"`
	HitAt3       bool        `json:"hit_at_3"`
	MRR          float64     `json:"mrr"`
	Evidence     hitEvidence `json:"evidence"`
	Error        string      `json:"error,omitempty"`
}

type report struct {
	GeneratedAt string  `json:"generated_at"`
	TopK        int     `json:"top_k"`
	Total       int     `json:"total"`
	HitAt1      float64 `json:"hit_at_1"`
	HitAt3      float64 `json:"hit_at_3"`
	MRR         float64 `json:"mrr"`
	Rows        []row   `json:"rows"`
}

type qaContent struct {
	Question string `json:"question"`
	Source   struct {
		Repo string `json:"repo"`
		Path string `json:"path"`
	} `json:"source"`
}

func main() {
	workspace := flag.String("workspace", "/home/steve/offer-pilot", "项目根目录")
	topK := flag.Int("topk", 5, "检索TopK")
	outMd := flag.String("out-md", "./examples/rag_evaluation_results.md", "输出Markdown（相对于backend）")
	outJSON := flag.String("out-json", "./examples/_meta/rag_evaluation_results.json", "输出JSON（相对于backend）")
	flag.Parse()

	backendRoot := mustAbs(filepath.Join(*workspace, "backend"))
	outMdPath := mustAbs(filepath.Join(backendRoot, *outMd))
	outJSONPath := mustAbs(filepath.Join(backendRoot, *outJSON))

	if err := os.MkdirAll(filepath.Dir(outMdPath), 0o755); err != nil {
		fatalf("创建输出目录失败: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(outJSONPath), 0o755); err != nil {
		fatalf("创建输出目录失败: %v", err)
	}

	conf, err := config.InitConfig("./config")
	if err != nil {
		fatalf("读取配置失败: %v", err)
	}

	host, port := conf.RagConfig.RedisHostPort()
	rdb, err := redis.NewRedisClient(&config.RedisConfig{
		RedisHost:     host,
		RedisPort:     port,
		RedisPassword: conf.RagConfig.RedisPassword,
		RedisDb:       conf.RagConfig.RedisDB,
	})
	if err != nil {
		fatalf("初始化Redis失败: %v", err)
	}

	emb, err := embedder.NewFromConfig(conf.RagConfig)
	if err != nil {
		fatalf("初始化Embedding失败: %v", err)
	}

	vStore := store.NewRedisVectorStore(rdb, conf.RagConfig)
	searchSvc := ragservice.NewSearchService(emb, vStore, conf.RagConfig.DefaultTopK, conf.RagConfig.MaxTopK)

	samples := buildSamples()
	rows := make([]row, 0, len(samples))
	h1, h3 := 0, 0
	mrr := 0.0
	ctx := context.Background()

	for _, s := range samples {
		r := row{ID: s.ID, Query: s.Query, ExpectedTags: s.ExpectedTags}

		results, searchErr := searchSvc.SearchRelevantChunks(ctx, s.Query, *topK, nil)
		if searchErr != nil {
			r.Error = searchErr.Error()
			rows = append(rows, r)
			continue
		}

		firstRank := 0
		evidence := hitEvidence{}
		for idx, item := range results {
			rank := idx + 1
			if !hasTagOverlap(item.Metadata.Tags, s.ExpectedTags) {
				continue
			}
			if firstRank == 0 {
				firstRank = rank
				evidence = hitEvidence{
					Rank:  rank,
					Score: item.Score,
					Tags:  item.Metadata.Tags,
				}
				repo, q := parseContentMeta(item.Content)
				evidence.Repo = repo
				evidence.Question = q
			}
		}

		if firstRank == 1 {
			r.HitAt1 = true
			h1++
		}
		if firstRank > 0 && firstRank <= 3 {
			r.HitAt3 = true
			h3++
		}
		if firstRank > 0 {
			r.MRR = 1.0 / float64(firstRank)
			mrr += r.MRR
		}
		r.Evidence = evidence
		rows = append(rows, r)
	}

	total := len(rows)
	rep := report{
		GeneratedAt: time.Now().Format(time.RFC3339),
		TopK:        *topK,
		Total:       total,
		HitAt1:      ratio(h1, total),
		HitAt3:      ratio(h3, total),
		MRR:         round3(mrr / float64(total)),
		Rows:        rows,
	}

	js, _ := json.MarshalIndent(rep, "", "  ")
	if err = os.WriteFile(outJSONPath, js, 0o644); err != nil {
		fatalf("写入JSON报告失败: %v", err)
	}

	md := renderMarkdown(rep)
	if err = os.WriteFile(outMdPath, []byte(md), 0o644); err != nil {
		fatalf("写入Markdown报告失败: %v", err)
	}

	fmt.Printf("评测完成\n")
	fmt.Printf("样本数: %d, Hit@1=%.3f, Hit@3=%.3f, MRR=%.3f\n", rep.Total, rep.HitAt1, rep.HitAt3, rep.MRR)
	fmt.Printf("Markdown: %s\n", outMdPath)
	fmt.Printf("JSON: %s\n", outJSONPath)
}

func buildSamples() []sample {
	return []sample{
		{"Q01", "C++ 智能指针 shared_ptr 和 unique_ptr 区别", []string{"cpp"}},
		{"Q02", "C++ 左值右值和 move 语义怎么理解", []string{"cpp"}},
		{"Q03", "手写线程池一般要考虑哪些模块", []string{"cpp", "os"}},
		{"Q04", "Linux epoll 和 select 的区别及适用场景", []string{"os", "network"}},
		{"Q05", "进程和线程的核心区别是什么", []string{"os"}},
		{"Q06", "TCP 三次握手四次挥手为什么要这样设计", []string{"network"}},
		{"Q07", "HTTP 和 HTTPS 的主要差异", []string{"network"}},
		{"Q08", "Redis 为什么快，从 IO 模型到数据结构说一下", []string{"redis", "os"}},
		{"Q09", "Redis 缓存穿透、击穿、雪崩怎么治理", []string{"redis"}},
		{"Q10", "Redis 持久化 RDB 和 AOF 的区别", []string{"redis"}},
		{"Q11", "MySQL InnoDB 索引失效的常见原因", []string{"mysql"}},
		{"Q12", "MySQL 事务隔离级别和 MVCC 的关系", []string{"mysql"}},
		{"Q13", "Kafka 怎么保证消息不丢和有序", []string{"mq", "distributed"}},
		{"Q14", "RabbitMQ 消息堆积如何处理", []string{"mq"}},
		{"Q15", "分布式系统里 CAP 和一致性怎么取舍", []string{"distributed"}},
		{"Q16", "服务注册发现有哪些方案，ZK 和 Nacos 区别", []string{"distributed"}},
		{"Q17", "Go goroutine 调度模型 GPM 是什么", []string{"golang", "os"}},
		{"Q18", "Go channel 和 mutex 在并发控制上怎么选", []string{"golang"}},
		{"Q19", "Java 内存模型 JMM 的可见性和有序性", []string{"java"}},
		{"Q20", "CAS 的 ABA 问题怎么解决", []string{"java"}},
		{"Q21", "设计模式里单例模式有哪些线程安全写法", []string{"cpp", "java"}},
		{"Q22", "快排和归并排序复杂度与稳定性对比", []string{"algorithm"}},
		{"Q23", "二分查找模板在什么场景容易写错", []string{"algorithm"}},
		{"Q24", "死锁发生的必要条件和排查思路", []string{"os"}},
		{"Q25", "面试中如何回答为什么 Redis 用跳表", []string{"redis"}},
	}
}

func parseContentMeta(content string) (string, string) {
	var q qaContent
	if err := json.Unmarshal([]byte(content), &q); err == nil {
		repo := strings.TrimSpace(q.Source.Repo)
		if repo == "" {
			repo = "unknown"
		}
		question := strings.TrimSpace(q.Question)
		if question == "" {
			question = "(无 question 字段)"
		}
		return repo, question
	}
	return "unknown", firstN(content, 40)
}

func hasTagOverlap(a, b []string) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	m := map[string]struct{}{}
	for _, x := range a {
		m[strings.ToLower(strings.TrimSpace(x))] = struct{}{}
	}
	for _, y := range b {
		if _, ok := m[strings.ToLower(strings.TrimSpace(y))]; ok {
			return true
		}
	}
	return false
}

func renderMarkdown(rep report) string {
	var b strings.Builder
	b.WriteString("# RAG 检索评测结果（自动生成）\n\n")
	b.WriteString(fmt.Sprintf("- 生成时间: %s\n", rep.GeneratedAt))
	b.WriteString(fmt.Sprintf("- TopK: %d\n", rep.TopK))
	b.WriteString(fmt.Sprintf("- 样本总数: %d\n", rep.Total))
	b.WriteString(fmt.Sprintf("- Hit@1: %.3f\n", rep.HitAt1))
	b.WriteString(fmt.Sprintf("- Hit@3: %.3f\n", rep.HitAt3))
	b.WriteString(fmt.Sprintf("- MRR: %.3f\n\n", rep.MRR))

	b.WriteString("| 编号 | 查询语句 | 期望标签 | Hit@1 | Hit@3 | MRR | 首个相关命中证据 |\n")
	b.WriteString("|---|---|---|---|---|---|---|\n")

	rows := make([]row, len(rep.Rows))
	copy(rows, rep.Rows)
	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	for _, r := range rows {
		evi := "无"
		if r.Error != "" {
			evi = "error: " + safePipe(r.Error)
		} else if r.Evidence.Rank > 0 {
			evi = fmt.Sprintf("rank=%d repo=%s score=%.4f q=%s", r.Evidence.Rank, safePipe(r.Evidence.Repo), r.Evidence.Score, safePipe(firstN(r.Evidence.Question, 24)))
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %t | %t | %.3f | %s |\n",
			r.ID,
			safePipe(firstN(r.Query, 36)),
			safePipe(strings.Join(r.ExpectedTags, ",")),
			r.HitAt1,
			r.HitAt3,
			r.MRR,
			evi,
		))
	}
	return b.String()
}

func firstN(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

func safePipe(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "|", "\\|")
}

func ratio(x, total int) float64 {
	if total == 0 {
		return 0
	}
	return round3(float64(x) / float64(total))
}

func round3(v float64) float64 {
	return float64(int(v*1000+0.5)) / 1000
}

func mustAbs(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		fatalf("解析路径失败 %s: %v", p, err)
	}
	return abs
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
