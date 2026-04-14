package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

type sourceMeta struct {
	Repo string `json:"repo"`
	Path string `json:"path"`
}

type qaItem struct {
	Question   string     `json:"question"`
	Answer     string     `json:"answer"`
	Tags       []string   `json:"tags"`
	Difficulty string     `json:"difficulty"`
	Source     sourceMeta `json:"source"`
}

type report struct {
	InputDir         string         `json:"input_dir"`
	OutputFile       string         `json:"output_file"`
	SourceMarkdown   int            `json:"source_markdown"`
	StructuredItems  int            `json:"structured_items"`
	SkippedEmpty     int            `json:"skipped_empty"`
	DifficultyCount  map[string]int `json:"difficulty_count"`
	TopTags          []tagCounter   `json:"top_tags"`
	PerRepoItemCount map[string]int `json:"per_repo_item_count"`
}

type tagCounter struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}

var (
	h2Pattern = regexp.MustCompile(`(?m)^##\s+(.+)$`)
)

func main() {
	workspace := flag.String("workspace", "/home/steve/offer-pilot", "项目根目录")
	inDir := flag.String("in", "./examples/rag_data_cleaned_full", "输入目录（相对于 backend）")
	outDir := flag.String("out", "./examples/rag_data_structured_full", "输出目录（相对于 backend）")
	outFileName := flag.String("file", "qa_dataset.json", "输出 JSON 文件名")
	minAnswerRunes := flag.Int("min-answer-runes", 80, "最小答案长度")
	flag.Parse()

	backendRoot := mustAbs(filepath.Join(*workspace, "backend"))
	inputDirAbs := mustAbs(filepath.Join(backendRoot, *inDir))
	outputDirAbs := mustAbs(filepath.Join(backendRoot, *outDir))
	outputFile := filepath.Join(outputDirAbs, *outFileName)

	if err := os.RemoveAll(outputDirAbs); err != nil {
		fatalf("清空输出目录失败: %v", err)
	}
	if err := os.MkdirAll(outputDirAbs, 0o755); err != nil {
		fatalf("创建输出目录失败: %v", err)
	}
	metaDir := filepath.Join(outputDirAbs, "_meta")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		fatalf("创建元数据目录失败: %v", err)
	}

	entries, err := os.ReadDir(inputDirAbs)
	if err != nil {
		fatalf("读取输入目录失败: %v", err)
	}

	mdFiles := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(e.Name())) != ".md" {
			continue
		}
		mdFiles = append(mdFiles, filepath.Join(inputDirAbs, e.Name()))
	}
	sort.Strings(mdFiles)

	items := make([]qaItem, 0, 1024)
	tagMap := map[string]int{}
	diffMap := map[string]int{"easy": 0, "medium": 0, "hard": 0}
	perRepo := map[string]int{}
	skipped := 0

	for _, file := range mdFiles {
		raw, readErr := os.ReadFile(file)
		if readErr != nil {
			skipped++
			continue
		}

		repo, relPath, body := splitSourceHeader(string(raw))
		if strings.TrimSpace(body) == "" {
			skipped++
			continue
		}

		units := splitByH2(body)
		for _, u := range units {
			q := normalizeQuestion(u.title, relPath)
			a := strings.TrimSpace(u.content)
			if utf8.RuneCountInString(a) < *minAnswerRunes {
				continue
			}

			tags := inferTags(repo, relPath, q, a)
			difficulty := inferDifficulty(a)

			for _, t := range tags {
				tagMap[t]++
			}
			diffMap[difficulty]++
			perRepo[repo]++

			items = append(items, qaItem{
				Question:   q,
				Answer:     a,
				Tags:       tags,
				Difficulty: difficulty,
				Source: sourceMeta{
					Repo: repo,
					Path: relPath,
				},
			})
		}
	}

	if len(items) == 0 {
		fatalf("未生成任何结构化条目，请检查输入目录或阈值")
	}

	encoded, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		fatalf("序列化 JSON 失败: %v", err)
	}
	if err = os.WriteFile(outputFile, encoded, 0o644); err != nil {
		fatalf("写入输出 JSON 失败: %v", err)
	}

	tops := topTags(tagMap, 30)
	rep := report{
		InputDir:         inputDirAbs,
		OutputFile:       outputFile,
		SourceMarkdown:   len(mdFiles),
		StructuredItems:  len(items),
		SkippedEmpty:     skipped,
		DifficultyCount:  diffMap,
		TopTags:          tops,
		PerRepoItemCount: perRepo,
	}

	repBytes, _ := json.MarshalIndent(rep, "", "  ")
	reportFile := filepath.Join(metaDir, "structify_report.json")
	if err = os.WriteFile(reportFile, repBytes, 0o644); err != nil {
		fatalf("写入报告失败: %v", err)
	}

	fmt.Printf("二次结构化完成\n")
	fmt.Printf("输入 markdown: %d\n", len(mdFiles))
	fmt.Printf("输出问答: %d\n", len(items))
	fmt.Printf("输出文件: %s\n", outputFile)
	fmt.Printf("统计报告: %s\n", reportFile)
}

type sectionUnit struct {
	title   string
	content string
}

func splitSourceHeader(raw string) (string, string, string) {
	repo := "unknown"
	relPath := "unknown"

	parts := strings.SplitN(raw, "\n---\n", 2)
	if len(parts) == 2 {
		header := parts[0]
		body := strings.TrimSpace(parts[1])
		scanner := bufio.NewScanner(strings.NewReader(header))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.HasPrefix(line, "- 仓库:") {
				repo = strings.TrimSpace(strings.TrimPrefix(line, "- 仓库:"))
			}
			if strings.HasPrefix(line, "- 文件:") {
				relPath = strings.TrimSpace(strings.TrimPrefix(line, "- 文件:"))
			}
		}
		return repo, relPath, body
	}

	return repo, relPath, strings.TrimSpace(raw)
}

func splitByH2(body string) []sectionUnit {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}

	idx := h2Pattern.FindAllStringSubmatchIndex(body, -1)
	if len(idx) == 0 {
		title := firstHeading(body)
		if title == "" {
			title = "通用问题"
		}
		return []sectionUnit{{title: title, content: body}}
	}

	out := make([]sectionUnit, 0, len(idx))
	for i, m := range idx {
		title := strings.TrimSpace(body[m[2]:m[3]])
		start := m[0]
		end := len(body)
		if i+1 < len(idx) {
			end = idx[i+1][0]
		}
		block := strings.TrimSpace(body[start:end])
		if block == "" {
			continue
		}
		out = append(out, sectionUnit{title: title, content: block})
	}
	if len(out) == 0 {
		return []sectionUnit{{title: "通用问题", content: body}}
	}
	return out
}

func firstHeading(body string) string {
	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimLeft(line, "#"))
		}
	}
	return ""
}

func normalizeQuestion(title, relPath string) string {
	title = strings.TrimSpace(strings.Trim(title, "#"))
	if title != "" {
		return title
	}
	base := strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	base = strings.ReplaceAll(base, "_", " ")
	if strings.TrimSpace(base) == "" {
		return "通用问题"
	}
	return base
}

func inferTags(repo, relPath, question, answer string) []string {
	text := strings.ToLower(strings.Join([]string{repo, relPath, question, answer}, " "))
	candidates := []struct {
		k string
		v []string
	}{
		{"cpp", []string{"c++", "stl", "template", "智能指针", "内存池", "lambda", "move"}},
		{"golang", []string{"go", "golang", "goroutine", "channel"}},
		{"java", []string{"java", "jvm", "juc", "thread"}},
		{"redis", []string{"redis", "缓存", "cache", "过期", "哨兵", "cluster"}},
		{"mysql", []string{"mysql", "innodb", "索引", "事务", "mvcc"}},
		{"mq", []string{"kafka", "rabbitmq", "消息队列", "mq"}},
		{"distributed", []string{"分布式", "一致性", "raft", "zk", "zookeeper", "微服务"}},
		{"network", []string{"tcp", "http", "rpc", "网络"}},
		{"os", []string{"linux", "io", "进程", "线程", "内存"}},
		{"algorithm", []string{"算法", "排序", "二分", "动态规划", "dfs", "bfs", "链表", "树"}},
	}

	tags := make([]string, 0)
	for _, c := range candidates {
		for _, kw := range c.v {
			if strings.Contains(text, strings.ToLower(kw)) {
				tags = append(tags, c.k)
				break
			}
		}
	}

	if len(tags) == 0 {
		tags = append(tags, "general")
	}
	return dedup(tags)
}

func inferDifficulty(answer string) string {
	runes := utf8.RuneCountInString(answer)
	complexHints := []string{"原理", "机制", "源码", "一致性", "事务", "性能", "优化", "实现", "为什么", "推导", "模型"}
	hit := 0
	for _, h := range complexHints {
		if strings.Contains(answer, h) {
			hit++
		}
	}

	score := 0
	if runes >= 500 {
		score += 2
	}
	if runes >= 1200 {
		score += 2
	}
	score += hit

	if score >= 5 {
		return "hard"
	}
	if score >= 2 {
		return "medium"
	}
	return "easy"
}

func topTags(m map[string]int, n int) []tagCounter {
	arr := make([]tagCounter, 0, len(m))
	for k, v := range m {
		arr = append(arr, tagCounter{Tag: k, Count: v})
	}
	sort.Slice(arr, func(i, j int) bool {
		if arr[i].Count == arr[j].Count {
			return arr[i].Tag < arr[j].Tag
		}
		return arr[i].Count > arr[j].Count
	})
	if len(arr) > n {
		arr = arr[:n]
	}
	return arr
}

func dedup(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
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
