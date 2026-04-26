package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/milabo0718/offer-pilot/backend/common/aihelper"
	"github.com/milabo0718/offer-pilot/backend/common/code"
	messagedao "github.com/milabo0718/offer-pilot/backend/dao/message"
	reportdao "github.com/milabo0718/offer-pilot/backend/dao/report"
	"github.com/milabo0718/offer-pilot/backend/dao/session"
	"github.com/milabo0718/offer-pilot/backend/model"
	ragservice "github.com/milabo0718/offer-pilot/backend/service/rag"

	"github.com/google/uuid"
)

type SessionService struct {
	sessionDao *session.SessionDao
	messageDao *messagedao.MessageDao
	reportDao  *reportdao.ReportDao
	aiManager  *aihelper.AIHelperManager
	ragService *ragservice.Service
	ragEnabled bool
	ragTopK    int
}

func NewSessionService(
	sessionDao *session.SessionDao,
	messageDao *messagedao.MessageDao,
	reportDao *reportdao.ReportDao,
	aiManager *aihelper.AIHelperManager,
) *SessionService {
	return &SessionService{
		sessionDao: sessionDao,
		messageDao: messageDao,
		reportDao:  reportDao,
		aiManager:  aiManager,
		ragTopK:    3,
	}
}

// ConfigureRAGAugment 配置聊天前置检索增强，失败会自动降级为普通聊天。
func (s *SessionService) ConfigureRAGAugment(ragSvc *ragservice.Service, enabled bool, topK int) {
	s.ragService = ragSvc
	s.ragEnabled = enabled
	if topK > 0 {
		s.ragTopK = topK
	}
}

func (s *SessionService) GetUserSessionsByUserName(ctx context.Context, userName string) ([]model.SessionInfo, error) {
	dbSessions, err := s.sessionDao.GetSessionsByUserName(ctx, userName)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(dbSessions))
	sessionInfos := make([]model.SessionInfo, 0, len(dbSessions))
	for _, item := range dbSessions {
		seen[item.ID] = struct{}{}
		sessionInfos = append(sessionInfos, model.SessionInfo{
			SessionID: item.ID,
			Title:     item.Title,
		})
	}

	for _, sessionID := range s.aiManager.GetUserSessions(userName) {
		if _, ok := seen[sessionID]; ok {
			continue
		}
		sessionInfos = append(sessionInfos, model.SessionInfo{
			SessionID: sessionID,
			Title:     sessionID,
		})
	}

	return sessionInfos, nil
}

func (s *SessionService) CreateSessionAndSendMessage(ctx context.Context, userName string, userQuestion string, modelType string, jdProfile string) (string, string, code.Code) {
	//1：创建一个新的会话
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: userName,
		Title:    userQuestion, // 可以根据需求设置标题，这边暂时用用户第一次的问题作为标题
	}

	createdSession, err := s.sessionDao.CreateSession(ctx, newSession)
	if err != nil {
		log.Println("CreateSessionAndSendMessage CreateSession error:", err)
		return "", "", code.CodeServerBusy
	}

	//2：获取AIHelper并通过其管理消息
	config := map[string]interface{}{
		"apiKey": "your-api-key",
	}
	helper, err := s.aiManager.GetOrCreateAIHelper(ctx, userName, createdSession.ID, modelType, config)
	if err != nil {
		log.Println("CreateSessionAndSendMessage GetOrCreateAIHelper error:", err)
		return "", "", code.AIModelFail
	}
	helper.SetSystemPrompt(buildInterviewerSystemPrompt(jdProfile))

	//3：生成AI回复
	prompt := s.buildChatPrompt(ctx, userQuestion, jdProfile)
	aiResponse, err_ := helper.GenerateResponse(userName, ctx, prompt)
	if err_ != nil {
		log.Println("CreateSessionAndSendMessage GenerateResponse error:", err_)
		return "", "", code.AIModelFail
	}

	return createdSession.ID, aiResponse.Content, code.CodeSuccess
}

func (s *SessionService) CreateStreamSessionOnly(ctx context.Context, userName string, userQuestion string) (string, code.Code) {
	newSession := &model.Session{
		ID:       uuid.New().String(),
		UserName: userName,
		Title:    userQuestion,
	}
	createdSession, err := s.sessionDao.CreateSession(ctx, newSession)
	if err != nil {
		log.Println("CreateStreamSessionOnly CreateSession error:", err)
		return "", code.CodeServerBusy
	}
	return createdSession.ID, code.CodeSuccess
}

func (s *SessionService) StreamMessageToExistingSession(ctx context.Context, userName string, sessionID string, userQuestion string, modelType string, jdProfile string, writer http.ResponseWriter) code.Code {
	// 确保 writer 支持 Flush
	flusher, ok := writer.(http.Flusher)
	if !ok {
		log.Println("StreamMessageToExistingSession: streaming unsupported")
		return code.CodeServerBusy
	}

	config := map[string]interface{}{
		"apiKey": "your-api-key",
	}
	helper, err := s.aiManager.GetOrCreateAIHelper(ctx, userName, sessionID, modelType, config)
	if err != nil {
		log.Println("StreamMessageToExistingSession GetOrCreateAIHelper error:", err)
		return code.AIModelFail
	}
	helper.SetSystemPrompt(buildInterviewerSystemPrompt(jdProfile))

	cb := func(msg string) {
		// 直接发送数据，不转义
		// SSE 格式：data: <content>\n\n
		log.Printf("[SSE] Sending chunk: %s (len=%d)\n", msg, len(msg))
		_, err := writer.Write([]byte("data: " + msg + "\n\n"))
		if err != nil {
			log.Println("[SSE] Write error:", err)
			return
		}
		flusher.Flush() //  每次必须 flush
		log.Println("[SSE] Flushed")
	}

	prompt := s.buildChatPrompt(ctx, userQuestion, jdProfile)
	_, err_ := helper.StreamResponse(userName, ctx, cb, prompt)
	if err_ != nil {
		log.Println("StreamMessageToExistingSession StreamResponse error:", err_)
		return code.AIModelFail
	}

	_, err = writer.Write([]byte("data: [DONE]\n\n"))
	if err != nil {
		log.Println("StreamMessageToExistingSession write DONE error:", err)
		return code.AIModelFail
	}
	flusher.Flush()

	return code.CodeSuccess
}

func (s *SessionService) CreateStreamSessionAndSendMessage(ctx context.Context, userName string, userQuestion string, modelType string, jdProfile string, writer http.ResponseWriter) (string, code.Code) {

	sessionID, code_ := s.CreateStreamSessionOnly(ctx, userName, userQuestion)
	if code_ != code.CodeSuccess {
		return "", code_
	}

	code_ = s.StreamMessageToExistingSession(ctx, userName, sessionID, userQuestion, modelType, jdProfile, writer)
	if code_ != code.CodeSuccess {

		return sessionID, code_
	}

	return sessionID, code.CodeSuccess
}

func (s *SessionService) ChatSend(ctx context.Context, userName string, sessionID string, userQuestion string, modelType string, jdProfile string) (string, code.Code) {
	//1：获取AIHelper
	config := map[string]interface{}{
		"apiKey": "your-api-key", // TODO: 从配置中获取
	}
	helper, err := s.aiManager.GetOrCreateAIHelper(ctx, userName, sessionID, modelType, config)
	if err != nil {
		log.Println("ChatSend GetOrCreateAIHelper error:", err)
		return "", code.AIModelFail
	}
	helper.SetSystemPrompt(buildInterviewerSystemPrompt(jdProfile))

	//2：生成AI回复
	prompt := s.buildChatPrompt(ctx, userQuestion, jdProfile)
	aiResponse, err_ := helper.GenerateResponse(userName, ctx, prompt)
	if err_ != nil {
		log.Println("ChatSend GenerateResponse error:", err_)
		return "", code.AIModelFail
	}

	return aiResponse.Content, code.CodeSuccess
}

func (s *SessionService) GetChatHistory(ctx context.Context, userName string, sessionID string) ([]model.History, code.Code) {
	helper, exists := s.aiManager.GetAIHelper(userName, sessionID)
	if exists {
		messages := helper.GetMessages()
		if len(messages) > 0 {
			history := make([]model.History, 0, len(messages))
			for _, msg := range messages {
				history = append(history, model.History{
					IsUser:  msg.IsUser,
					Content: msg.Content,
				})
			}
			return history, code.CodeSuccess
		}
	}

	if _, err := s.sessionDao.GetSessionByIDAndUserName(ctx, sessionID, userName); err != nil {
		log.Println("GetChatHistory GetSessionByIDAndUserName error:", err)
		return nil, code.CodeRecordNotFound
	}

	messages, err := s.messageDao.GetMessagesBySessionID(ctx, sessionID)
	if err != nil {
		log.Println("GetChatHistory GetMessagesBySessionID error:", err)
		return nil, code.CodeServerBusy
	}

	history := make([]model.History, 0, len(messages))
	for _, msg := range messages {
		history = append(history, model.History{
			IsUser:  msg.IsUser,
			Content: msg.Content,
		})
	}

	return history, code.CodeSuccess
}

// 流式发送消息
func (s *SessionService) ChatStreamSend(ctx context.Context, userName string, sessionID string, userQuestion string, modelType string, jdProfile string, writer http.ResponseWriter) code.Code {

	return s.StreamMessageToExistingSession(ctx, userName, sessionID, userQuestion, modelType, jdProfile, writer)
}

func (s *SessionService) GenerateInterviewReport(ctx context.Context, userName string, sessionID string, modelType string, jdProfile string, force bool) (*model.InterviewReportData, code.Code) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, code.CodeInvalidParams
	}
	if modelType == "" {
		modelType = "1"
	}

	if _, err := s.sessionDao.GetSessionByIDAndUserName(ctx, sessionID, userName); err != nil {
		log.Println("GenerateInterviewReport session not found:", err)
		return nil, code.CodeRecordNotFound
	}

	if !force && s.reportDao != nil {
		if cached, err := s.reportDao.GetBySessionID(ctx, sessionID); err == nil && strings.TrimSpace(cached.ReportJSON) != "" {
			var report model.InterviewReportData
			if err = json.Unmarshal([]byte(cached.ReportJSON), &report); err == nil {
				return &report, code.CodeSuccess
			}
			log.Println("GenerateInterviewReport cached unmarshal error:", err)
		}
	}

	history, code_ := s.GetChatHistory(ctx, userName, sessionID)
	if code_ != code.CodeSuccess {
		return nil, code_
	}
	if len(history) == 0 {
		return nil, code.CodeInvalidParams
	}

	report, err := s.generateModelInterviewReport(ctx, sessionID, modelType, jdProfile, history)
	if err != nil {
		log.Println("GenerateInterviewReport model fallback:", err)
		report = s.buildFallbackInterviewReport(sessionID, history, jdProfile)
	}
	report.SessionID = sessionID
	report.EvidenceCount = countUserAnswers(history)
	if report.GeneratedAt.IsZero() {
		report.GeneratedAt = time.Now()
	}
	normalizeReport(report)

	if s.reportDao != nil {
		payload, err := json.Marshal(report)
		if err != nil {
			log.Println("GenerateInterviewReport marshal error:", err)
		} else {
			err = s.reportDao.Upsert(ctx, &model.InterviewReport{
				SessionID:  sessionID,
				UserName:   userName,
				ModelType:  modelType,
				JDProfile:  jdProfile,
				ReportJSON: string(payload),
			})
			if err != nil {
				log.Println("GenerateInterviewReport save report error:", err)
			}
		}
	}

	return report, code.CodeSuccess
}

func (s *SessionService) generateModelInterviewReport(ctx context.Context, sessionID string, modelType string, jdProfile string, history []model.History) (*model.InterviewReportData, error) {
	prompt := buildReportPrompt(jdProfile, history)
	respText, err := aihelper.ParseTextWithModel(ctx, modelType, prompt, map[string]interface{}{})
	if err != nil {
		return nil, err
	}

	jsonText := extractJSONObject(respText)
	if jsonText == "" {
		return nil, fmt.Errorf("report model returned non-json response")
	}

	var report model.InterviewReportData
	if err = json.Unmarshal([]byte(jsonText), &report); err != nil {
		return nil, err
	}
	report.SessionID = sessionID
	return &report, nil
}

func buildReportPrompt(jdProfile string, history []model.History) string {
	var b strings.Builder
	b.WriteString(`你是 OfferPilot 的资深技术面试评估官。请基于下面的岗位画像和真实面试对话，生成一份结构化中文评估报告。

只返回 JSON，不要输出 Markdown 或额外解释。评分范围 0-100，必须给出整数。
JSON schema:
{
  "summary": "一句话总评",
  "recommendation": "是否建议进入下一轮，以及理由",
  "scores": {
    "tech": 0,
    "eng": 0,
    "ps": 0,
    "comm": 0,
    "learn": 0,
    "fit": 0
  },
  "strengths": ["亮点1", "亮点2"],
  "risks": ["风险1", "风险2"],
  "actionItems": ["后续提升建议1", "后续提升建议2"],
  "detail": "分维度说明评分依据，引用候选人的回答证据"
}

评估维度说明：
- tech: 技术基础
- eng: 工程实践
- ps: 问题解决
- comm: 沟通表达
- learn: 学习能力
- fit: 岗位匹配度

【岗位画像】
`)
	if strings.TrimSpace(jdProfile) == "" {
		b.WriteString("未提供，按通用后端/软件工程师岗位评估。\n")
	} else {
		b.WriteString(jdProfile)
		b.WriteString("\n")
	}
	b.WriteString("\n【面试对话】\n")
	for i, item := range history {
		role := "候选人"
		if !item.IsUser {
			role = "面试官"
		}
		content := strings.TrimSpace(item.Content)
		if len([]rune(content)) > 1200 {
			content = string([]rune(content)[:1200]) + "..."
		}
		b.WriteString(fmt.Sprintf("%02d. %s：%s\n", i+1, role, content))
	}
	return b.String()
}

func (s *SessionService) buildFallbackInterviewReport(sessionID string, history []model.History, jdProfile string) *model.InterviewReportData {
	stats := summarizeAnswerStats(history)
	base := 52
	if stats.answerCount >= 2 {
		base += 8
	}
	if stats.avgRunes >= 40 {
		base += 8
	}
	if stats.avgRunes >= 120 {
		base += 6
	}
	if stats.technicalHits >= 3 {
		base += 8
	}
	if strings.TrimSpace(jdProfile) != "" {
		base += 4
	}
	base = clampScore(base)

	scores := model.AbilityScores{
		Tech:  clampScore(base + minInt(stats.technicalHits*2, 10) - shortAnswerPenalty(stats.avgRunes)),
		Eng:   clampScore(base + minInt(stats.engineeringHits*3, 12) - 3),
		PS:    clampScore(base + minInt(stats.problemHits*3, 12) - 2),
		Comm:  clampScore(55 + minInt(stats.avgRunes/4, 25) - repeatedShortPenalty(stats.shortAnswers, stats.answerCount)),
		Learn: clampScore(base - 1 + minInt(stats.reflectionHits*4, 12)),
		Fit:   clampScore(base + profileMatchBonus(jdProfile, history)),
	}

	strengths := []string{}
	if stats.technicalHits > 0 {
		strengths = append(strengths, "回答中出现了与岗位相关的技术关键词，具备一定技术基础。")
	}
	if stats.avgRunes >= 80 {
		strengths = append(strengths, "候选人回答长度较充分，能展开说明自己的思路。")
	}
	if stats.engineeringHits > 0 {
		strengths = append(strengths, "有工程实践意识，回答中涉及项目、服务、数据库或中间件等实践内容。")
	}
	if len(strengths) == 0 {
		strengths = append(strengths, "已完成基础问答流程，可以作为后续深入面试的初始样本。")
	}

	risks := []string{}
	if stats.answerCount < 2 {
		risks = append(risks, "有效作答轮次偏少，当前报告可信度有限。")
	}
	if stats.avgRunes < 40 {
		risks = append(risks, "回答偏短，缺少推理过程、边界条件和实践证据。")
	}
	if stats.technicalHits == 0 {
		risks = append(risks, "技术关键词和具体方案较少，暂时难以判断技术深度。")
	}
	if len(risks) == 0 {
		risks = append(risks, "建议继续追问真实项目细节、故障处理和取舍依据。")
	}

	actionItems := []string{
		"后续回答建议采用“结论-原因-例子-边界”的结构，增强可评估性。",
		"补充真实项目中的数据规模、性能指标、失败案例和复盘结论。",
		"针对岗位关键词准备 3-5 个可展开的技术案例。",
	}

	detail := fmt.Sprintf(
		"本报告为模型不可用时的本地兜底评估，基于候选人作答轮次、平均回答长度、技术关键词和岗位画像匹配度生成。当前有效作答 %d 轮，平均回答约 %d 字，技术相关命中 %d 次。建议在完整面试结束后重新生成模型评分报告，以获得更精细的证据引用。",
		stats.answerCount,
		stats.avgRunes,
		stats.technicalHits,
	)

	return &model.InterviewReportData{
		SessionID:      sessionID,
		Summary:        fallbackSummary(scores, stats),
		Recommendation: fallbackRecommendation(scores, stats),
		Scores:         scores,
		Strengths:      strengths,
		Risks:          risks,
		ActionItems:    actionItems,
		Detail:         detail,
		EvidenceCount:  stats.answerCount,
		GeneratedAt:    time.Now(),
		Fallback:       true,
	}
}

type answerStats struct {
	answerCount     int
	totalRunes      int
	avgRunes        int
	shortAnswers    int
	technicalHits   int
	engineeringHits int
	problemHits     int
	reflectionHits  int
}

func summarizeAnswerStats(history []model.History) answerStats {
	technical := regexp.MustCompile(`(?i)(go|golang|java|c\+\+|redis|mysql|sql|mq|kafka|http|tcp|rpc|微服务|并发|锁|索引|事务|缓存|线程|进程|channel|map|sync|context|性能|分布式)`)
	engineering := regexp.MustCompile(`(?i)(项目|服务|系统|上线|部署|监控|日志|压测|排查|故障|数据库|中间件|接口|架构|优化|k8s|docker)`)
	problem := regexp.MustCompile(`(?i)(因为|所以|首先|然后|如果|但是|权衡|取舍|复杂度|瓶颈|方案|步骤|边界|风险)`)
	reflection := regexp.MustCompile(`(?i)(复盘|学习|改进|总结|不足|优化|经验|下次|理解|查阅)`)

	stats := answerStats{}
	for _, item := range history {
		if !item.IsUser {
			continue
		}
		content := strings.TrimSpace(item.Content)
		if content == "" || content == "__START__" || strings.Contains(content, "请开始对我的模拟面试") {
			continue
		}
		stats.answerCount++
		runeCount := len([]rune(content))
		stats.totalRunes += runeCount
		if runeCount < 30 {
			stats.shortAnswers++
		}
		stats.technicalHits += len(technical.FindAllString(content, -1))
		stats.engineeringHits += len(engineering.FindAllString(content, -1))
		stats.problemHits += len(problem.FindAllString(content, -1))
		stats.reflectionHits += len(reflection.FindAllString(content, -1))
	}
	if stats.answerCount > 0 {
		stats.avgRunes = stats.totalRunes / stats.answerCount
	}
	return stats
}

func normalizeReport(report *model.InterviewReportData) {
	report.Scores.Tech = clampScore(report.Scores.Tech)
	report.Scores.Eng = clampScore(report.Scores.Eng)
	report.Scores.PS = clampScore(report.Scores.PS)
	report.Scores.Comm = clampScore(report.Scores.Comm)
	report.Scores.Learn = clampScore(report.Scores.Learn)
	report.Scores.Fit = clampScore(report.Scores.Fit)

	if strings.TrimSpace(report.Summary) == "" {
		report.Summary = "已生成面试评估报告。"
	}
	if strings.TrimSpace(report.Recommendation) == "" {
		report.Recommendation = "建议结合更多追问结果综合判断。"
	}
	if len(report.Strengths) == 0 {
		report.Strengths = []string{"候选人已完成本轮模拟面试，可继续结合追问补充证据。"}
	}
	if len(report.Risks) == 0 {
		report.Risks = []string{"当前报告未发现明确高风险，但仍建议补充真实项目证据。"}
	}
	if len(report.ActionItems) == 0 {
		report.ActionItems = []string{"继续围绕岗位关键技能准备结构化案例。"}
	}
	if strings.TrimSpace(report.Detail) == "" {
		report.Detail = "暂无更详细说明。"
	}
}

func countUserAnswers(history []model.History) int {
	return summarizeAnswerStats(history).answerCount
}

func fallbackSummary(scores model.AbilityScores, stats answerStats) string {
	avg := averageScores(scores)
	if stats.answerCount == 0 {
		return "当前缺少有效作答，暂无法形成稳定结论。"
	}
	if avg >= 78 {
		return "候选人整体表现较好，技术表达和岗位匹配度具备进一步面试价值。"
	}
	if avg >= 64 {
		return "候选人具备一定基础，但技术深度和项目证据仍需继续追问验证。"
	}
	return "候选人当前表现偏初级，建议重点补强基础概念、工程实践和表达结构。"
}

func fallbackRecommendation(scores model.AbilityScores, stats answerStats) string {
	avg := averageScores(scores)
	if stats.answerCount < 2 {
		return "建议先补充 2-3 轮追问后再做正式筛选判断。"
	}
	if avg >= 75 {
		return "建议进入下一轮，并重点追问真实项目中的复杂问题处理。"
	}
	if avg >= 60 {
		return "可作为待观察候选人，建议增加编码题或系统设计题验证深度。"
	}
	return "暂不建议直接进入下一轮，除非岗位要求偏初级或有培养空间。"
}

func averageScores(scores model.AbilityScores) int {
	values := []int{scores.Tech, scores.Eng, scores.PS, scores.Comm, scores.Learn, scores.Fit}
	sort.Ints(values)
	total := 0
	for _, v := range values {
		total += v
	}
	return total / len(values)
}

func profileMatchBonus(jdProfile string, history []model.History) int {
	jd := strings.ToLower(jdProfile)
	if strings.TrimSpace(jd) == "" {
		return 0
	}
	bonus := 0
	keywords := []string{"go", "golang", "java", "redis", "mysql", "kafka", "mq", "微服务", "并发", "分布式", "docker", "k8s", "sql", "http"}
	var answerText strings.Builder
	for _, item := range history {
		if item.IsUser {
			answerText.WriteString(strings.ToLower(item.Content))
			answerText.WriteByte('\n')
		}
	}
	answers := answerText.String()
	for _, kw := range keywords {
		if strings.Contains(jd, kw) && strings.Contains(answers, kw) {
			bonus += 2
		}
	}
	return minInt(bonus, 12)
}

func shortAnswerPenalty(avgRunes int) int {
	if avgRunes == 0 {
		return 12
	}
	if avgRunes < 30 {
		return 8
	}
	if avgRunes < 60 {
		return 4
	}
	return 0
}

func repeatedShortPenalty(shortAnswers int, answerCount int) int {
	if answerCount == 0 {
		return 18
	}
	if shortAnswers*2 >= answerCount {
		return 10
	}
	return 0
}

func clampScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

// buildInterviewerSystemPrompt 构造 AI 面试官的系统提示词。
// 岗位画像会被嵌入 system 消息，模型在整个会话里都按"面试官"角色主动出题、追问。
func buildInterviewerSystemPrompt(jdProfile string) string {
	var b strings.Builder
	b.WriteString(`你是 OfferPilot 的 AI 面试官，正在对用户进行一对一技术模拟面试。

【你的行为准则】
1. 你来主导整个面试：每一轮只向用户提出【一道】面试题，等待用户回答后，先用 1-2 句简短点评（指出亮点与不足），再自然抛出下一道题或合适的追问。
2. 不要一次性给出多道题目，不要长篇讲解理论（除非用户明确让你展开讲解）。
3. 题目要贴合下方【岗位画像】，由浅入深：先基础概念 → 再原理/源码 → 再系统设计 / 工程实践 / 项目经验。根据用户作答质量动态调整难度。
4. 使用中文，语气专业、友好，像真实面试官那样提问；提问时使用自然语言，不要用 markdown 标题或大量列表包裹问题。
5. 当用户说"开始面试 / 开始吧 / 请开始 / 你好"等表示开始的内容，或者输入为空白 / "__START__"，视为面试起点：先用一句话自我介绍（表明你是本岗位的 AI 面试官 + 岗位名），再抛出第一道题。
6. 用户说"换题 / 跳过 / 下一题"时立即换一道相关题；用户明显跑题时礼貌拉回到本岗位。
7. 如果用户给出面试题的答案，绝不要自行展开长篇讲解把答案"自问自答"，而应针对他的回答进行点评 + 追问或出下一题。
8. 当用户回答不出来的时候，直接给出答案，并给出答案的解析，再给出下一道题。
9. 第一道题应该出关于sync.Map和map+互斥锁的底层实现的题目。
`)

	jd := strings.TrimSpace(jdProfile)
	if jd != "" {
		b.WriteString("\n【岗位画像】（据此出题）\n")
		b.WriteString(jd)
		b.WriteString("\n")
	} else {
		b.WriteString("\n【岗位画像】未提供具体岗位，按通用后端 / 软件工程师面试方向出题。\n")
	}

	return b.String()
}

// 解析岗位JD（文本）
func (s *SessionService) ParseJD(ctx context.Context, jdText string, modelType string) (*model.JDParseResult, code.Code) {
	if strings.TrimSpace(jdText) == "" {
		return nil, code.CodeInvalidParams
	}
	if modelType == "" {
		modelType = "1"
	}

	prompt := `你是资深HR，请从以下JD中提取信息并仅返回JSON，不要输出其它文本。
字段要求：
{
  "jobTitle": "岗位名称",
  "skills": ["技能1","技能2"],
  "experience": "经验要求",
  "keywords": ["关键词1","关键词2"],
  "summary": "一句话总结"
}

JD内容：
` + jdText

	respText, err := aihelper.ParseTextWithModel(ctx, modelType, prompt, map[string]interface{}{})
	if err != nil {
		log.Println("ParseJD ParseTextWithModel error:", err)
		return nil, code.AIModelFail
	}

	jsonText := extractJSONObject(respText)
	if jsonText == "" {
		log.Println("ParseJD invalid json response:", respText)
		return nil, code.AIModelFail
	}

	var result model.JDParseResult
	if err = json.Unmarshal([]byte(jsonText), &result); err != nil {
		log.Println("ParseJD unmarshal error:", err, " raw:", respText)
		return nil, code.AIModelFail
	}
	log.Printf("ParseJD success: jobTitle=%s skills=%v experience=%s keywords=%v summary=%s",
		result.JobTitle, result.Skills, result.Experience, result.Keywords, result.Summary)

	return &result, code.CodeSuccess
}

func extractJSONObject(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || end <= start {
		return ""
	}
	return text[start : end+1]
}

// buildChatPrompt 在用户消息末尾按开关追加 RAG 召回上下文。
// Step3 的核心落点：真实聊天链路（新会话 / 已有会话 / 流式 / 非流式）统一走这里。
// 注意：JD 岗位画像已写入 system prompt，这里不再重复拼接，避免污染会话历史。
func (s *SessionService) buildChatPrompt(ctx context.Context, userQuestion string, jdProfile string) string {
	_ = jdProfile
	base := userQuestion

	if !s.ragEnabled || s.ragService == nil {
		log.Printf("[RAG Augment] skip: enabled=%v service=%v", s.ragEnabled, s.ragService != nil)
		return base
	}

	results, err := s.ragService.SearchRelevantChunks(ctx, userQuestion, s.ragTopK)
	if err != nil {
		log.Printf("[RAG Augment] 检索失败，自动降级为无 RAG 回答: %v", err)
		return base
	}
	if len(results) == 0 {
		log.Printf("[RAG Augment] 未召回任何片段，退化为无 RAG 回答: question=%q", userQuestion)
		return base
	}

	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\n【RAG召回上下文】\n")
	b.WriteString("以下内容来自题库检索，请优先用于生成更贴近岗位需求的回答：\n")

	for i, item := range results {
		content := strings.TrimSpace(item.Content)
		if len(content) > 300 {
			content = content[:300] + "..."
		}
		b.WriteString(fmt.Sprintf("%d. 来源=%s 节点=%s\n%s\n", i+1, item.Metadata.SourceFile, item.Metadata.SectionOrIndex, content))
	}

	log.Printf("[RAG Augment] injected %d chunks into prompt (topK=%d, question=%q)", len(results), s.ragTopK, userQuestion)
	return b.String()
}
