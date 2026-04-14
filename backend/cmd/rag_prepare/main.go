package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

type repoRule struct {
	Name            string
	Root            string
	License         string
	IncludePrefixes []string
	ExcludePrefixes []string
	ExcludeDirNames map[string]struct{}
}

type sourceFile struct {
	Repo    string `json:"repo"`
	Path    string `json:"path"`
	License string `json:"license"`
}

type cleanedItem struct {
	sourceFile
	OutputFile string `json:"output_file"`
	RuneCount  int    `json:"rune_count"`
}

type report struct {
	Profile            string                  `json:"profile"`
	OutputDir          string                  `json:"output_dir"`
	Manifest           string                  `json:"manifest"`
	TotalInputFiles    int                     `json:"total_input_files"`
	TotalOutputFiles   int                     `json:"total_output_files"`
	SkippedTooShort    int                     `json:"skipped_too_short"`
	SkippedReadError   int                     `json:"skipped_read_error"`
	SkippedCleanError  int                     `json:"skipped_clean_error"`
	PerRepoInputCount  map[string]int          `json:"per_repo_input_count"`
	PerRepoOutputCount map[string]int          `json:"per_repo_output_count"`
	Items              []cleanedItem           `json:"items"`
	SkippedFiles       map[string][]sourceFile `json:"skipped_files"`
}

var (
	mdImagePattern     = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	htmlTagPattern     = regexp.MustCompile(`<[^>]+>`)
	onlyURLLinePattern = regexp.MustCompile(`^https?://\S+$`)
	multiBlankPattern  = regexp.MustCompile(`\n{3,}`)
)

func main() {
	workspaceDefault := "/home/steve/offer-pilot"
	workspace := flag.String("workspace", workspaceDefault, "项目根目录")
	outDir := flag.String("out", "./examples/rag_data_cleaned", "输出目录（相对于 backend）")
	minRunes := flag.Int("min-runes", 120, "最小正文长度（rune）")
	profile := flag.String("profile", "strict", "清洗档位：strict 或 full")
	flag.Parse()

	profileVal := strings.ToLower(strings.TrimSpace(*profile))
	if profileVal != "strict" && profileVal != "full" {
		fatalf("不支持的 profile: %s（仅支持 strict/full）", *profile)
	}

	backendRoot := mustAbs(filepath.Join(*workspace, "backend"))
	outputDirAbs := mustAbs(filepath.Join(backendRoot, *outDir))

	rules := buildRules(*workspace, profileVal)

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

	rep := report{
		Profile:            profileVal,
		OutputDir:          outputDirAbs,
		Manifest:           filepath.Join(metaDir, "manifest.json"),
		PerRepoInputCount:  map[string]int{},
		PerRepoOutputCount: map[string]int{},
		Items:              make([]cleanedItem, 0),
		SkippedFiles:       map[string][]sourceFile{"too_short": {}, "read_error": {}, "clean_error": {}},
	}

	usedNames := map[string]int{}
	for _, rule := range rules {
		files, err := collectMarkdownFiles(rule)
		if err != nil {
			fatalf("扫描仓库失败 %s: %v", rule.Name, err)
		}
		sort.Slice(files, func(i, j int) bool {
			return files[i].Path < files[j].Path
		})

		rep.PerRepoInputCount[rule.Name] = len(files)
		rep.TotalInputFiles += len(files)

		for _, sf := range files {
			raw, err := os.ReadFile(filepath.Join(rule.Root, sf.Path))
			if err != nil {
				rep.SkippedReadError++
				rep.SkippedFiles["read_error"] = append(rep.SkippedFiles["read_error"], sf)
				continue
			}

			cleaned, cleanErr := cleanMarkdown(string(raw))
			if cleanErr != nil {
				rep.SkippedCleanError++
				rep.SkippedFiles["clean_error"] = append(rep.SkippedFiles["clean_error"], sf)
				continue
			}

			runes := utf8.RuneCountInString(cleaned)
			if runes < *minRunes {
				rep.SkippedTooShort++
				rep.SkippedFiles["too_short"] = append(rep.SkippedFiles["too_short"], sf)
				continue
			}

			outName := buildOutputName(sf, usedNames)
			outPath := filepath.Join(outputDirAbs, outName)
			finalContent := withSourceHeader(sf, cleaned)
			if err = os.WriteFile(outPath, []byte(finalContent), 0o644); err != nil {
				fatalf("写入输出文件失败 %s: %v", outPath, err)
			}

			rep.TotalOutputFiles++
			rep.PerRepoOutputCount[rule.Name]++
			rep.Items = append(rep.Items, cleanedItem{sourceFile: sf, OutputFile: outName, RuneCount: runes})
		}
	}

	manifest, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fatalf("序列化清洗报告失败: %v", err)
	}
	if err = os.WriteFile(rep.Manifest, manifest, 0o644); err != nil {
		fatalf("写入清洗报告失败: %v", err)
	}

	fmt.Printf("清洗完成\n")
	fmt.Printf("档位: %s\n", profileVal)
	fmt.Printf("输出目录: %s\n", rep.OutputDir)
	fmt.Printf("输出文件数: %d / 输入文件数: %d\n", rep.TotalOutputFiles, rep.TotalInputFiles)
	fmt.Printf("跳过: too_short=%d read_error=%d clean_error=%d\n", rep.SkippedTooShort, rep.SkippedReadError, rep.SkippedCleanError)
	fmt.Printf("报告: %s\n", rep.Manifest)
}

func buildRules(workspace string, profile string) []repoRule {
	if profile == "full" {
		return []repoRule{
			{
				Name:            "Interview",
				Root:            mustAbs(filepath.Join(workspace, "Interview")),
				License:         "CC BY-NC-SA 4.0",
				ExcludeDirNames: setOf("img", "asset", "old", ".git"),
			},
			{
				Name:            "interview-baguwen",
				Root:            mustAbs(filepath.Join(workspace, "interview-baguwen")),
				License:         "README 限制非商业使用",
				ExcludeDirNames: setOf("img", ".git"),
			},
			{
				Name:            "java-eight-part",
				Root:            mustAbs(filepath.Join(workspace, "java-eight-part")),
				License:         "README 声明转载需授权",
				ExcludeDirNames: setOf("img", ".git"),
			},
			{
				Name:            "cpp_interview",
				Root:            mustAbs(filepath.Join(workspace, "cpp_interview")),
				License:         "README 标注 MIT（仓库未见 LICENSE 文件）",
				ExcludeDirNames: setOf(".git", ".vscode"),
			},
		}
	}

	return []repoRule{
		{
			Name:    "Interview",
			Root:    mustAbs(filepath.Join(workspace, "Interview")),
			License: "CC BY-NC-SA 4.0",
			IncludePrefixes: []string{
				"docs/Algorithm",
				"docs/面试求职",
			},
			ExcludePrefixes: []string{
				"docs/Kaggle",
				"docs/GitHub",
				"docs/简历指南",
			},
			ExcludeDirNames: setOf("img", "asset", "old", ".git", "src"),
		},
		{
			Name:    "interview-baguwen",
			Root:    mustAbs(filepath.Join(workspace, "interview-baguwen")),
			License: "README 限制非商业使用",
			IncludePrefixes: []string{
				"cache",
				"database",
				"gc",
				"golang",
				"microservice",
				"mq",
				"pattern",
				"redis",
				"sharding",
			},
			ExcludeDirNames: setOf("img", ".git"),
		},
		{
			Name:    "java-eight-part",
			Root:    mustAbs(filepath.Join(workspace, "java-eight-part")),
			License: "README 声明转载需授权",
			IncludePrefixes: []string{
				"docs/java",
				"docs/redis",
				"docs/mq",
				"docs/distributed",
			},
			ExcludePrefixes: []string{
				"docs/it-hot",
				"docs/tools",
			},
			ExcludeDirNames: setOf("img", ".git"),
		},
		{
			Name:    "cpp_interview",
			Root:    mustAbs(filepath.Join(workspace, "cpp_interview")),
			License: "README 标注 MIT（仓库未见 LICENSE 文件）",
			IncludePrefixes: []string{
				"C++.md",
				"linux服务器.md",
				"操作系统.md",
				"计算机网络.md",
				"数据库.md",
				"数据结构及算法.md",
				"手撕代码.md",
				"设计模式.md",
				"书籍笔记",
			},
			ExcludePrefixes: []string{
				"离谱问题.md",
			},
			ExcludeDirNames: setOf(".git", ".vscode"),
		},
	}
}

func collectMarkdownFiles(rule repoRule) ([]sourceFile, error) {
	out := make([]sourceFile, 0)
	err := filepath.WalkDir(rule.Root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(rule.Root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}

		if d.IsDir() {
			if _, ok := rule.ExcludeDirNames[d.Name()]; ok {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.ToLower(filepath.Ext(rel)) != ".md" {
			return nil
		}

		if len(rule.IncludePrefixes) > 0 && !matchAnyPrefix(rel, rule.IncludePrefixes) {
			return nil
		}
		if len(rule.ExcludePrefixes) > 0 && matchAnyPrefix(rel, rule.ExcludePrefixes) {
			return nil
		}

		out = append(out, sourceFile{Repo: rule.Name, Path: rel, License: rule.License})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func cleanMarkdown(raw string) (string, error) {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	lines := make([]string, 0)
	for scanner.Scan() {
		line := scanner.Text()
		trim := strings.TrimSpace(line)

		if trim == "" {
			lines = append(lines, "")
			continue
		}

		if shouldDropLine(trim) {
			continue
		}

		line = mdImagePattern.ReplaceAllString(line, "")
		line = htmlTagPattern.ReplaceAllString(line, "")
		line = strings.TrimRight(line, " \t")
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	cleaned := strings.TrimSpace(strings.Join(lines, "\n"))
	cleaned = multiBlankPattern.ReplaceAllString(cleaned, "\n\n")
	if cleaned == "" {
		return "", fmt.Errorf("cleaned empty")
	}
	return cleaned, nil
}

func shouldDropLine(trim string) bool {
	lower := strings.ToLower(trim)

	if strings.HasPrefix(trim, "!") && strings.Contains(trim, "](") {
		return true
	}
	if strings.HasPrefix(lower, "<img") || strings.HasPrefix(lower, "<div") || strings.HasPrefix(lower, "</div") {
		return true
	}
	if strings.HasPrefix(lower, "<p align") || strings.HasPrefix(lower, "</p") {
		return true
	}
	if strings.HasPrefix(lower, "<table") || strings.HasPrefix(lower, "</table") || strings.HasPrefix(lower, "<tr") || strings.HasPrefix(lower, "</tr") || strings.HasPrefix(lower, "<td") || strings.HasPrefix(lower, "</td") {
		return true
	}
	if strings.HasPrefix(lower, "<!--") || strings.HasPrefix(lower, "-->") {
		return true
	}
	if strings.Contains(lower, "img.shields.io") {
		return true
	}
	if onlyURLLinePattern.MatchString(trim) {
		return true
	}
	if strings.Contains(trim, "二维码") {
		return true
	}

	return false
}

func withSourceHeader(sf sourceFile, body string) string {
	header := strings.Builder{}
	header.WriteString("# 来源信息\n\n")
	header.WriteString(fmt.Sprintf("- 仓库: %s\n", sf.Repo))
	header.WriteString(fmt.Sprintf("- 文件: %s\n", sf.Path))
	header.WriteString(fmt.Sprintf("- 许可: %s\n", sf.License))
	header.WriteString("\n---\n\n")
	header.WriteString(body)
	header.WriteString("\n")
	return header.String()
}

func buildOutputName(sf sourceFile, used map[string]int) string {
	base := sf.Repo + "__" + strings.ReplaceAll(sf.Path, "/", "__")
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = safeFileName(base)
	if base == "" {
		base = "doc"
	}

	used[base]++
	if used[base] == 1 {
		return base + ".md"
	}
	return fmt.Sprintf("%s__%d.md", base, used[base])
}

func safeFileName(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := b.String()
	out = strings.Trim(out, "_")
	out = strings.ReplaceAll(out, "__", "_")
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	return out
}

func matchAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func setOf(vals ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(vals))
	for _, v := range vals {
		out[v] = struct{}{}
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
