package session

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/milabo0718/offer-pilot/backend/common/aihelper"
	"github.com/milabo0718/offer-pilot/backend/common/code"
	"github.com/milabo0718/offer-pilot/backend/dao/message"
	"github.com/milabo0718/offer-pilot/backend/dao/session"
	"github.com/milabo0718/offer-pilot/backend/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SessionService struct {
	sessionDao *session.SessionDao
	messageDao *message.MessageDao
	aiManager  *aihelper.AIHelperManager
}

func NewSessionService(sessionDao *session.SessionDao, messageDao *message.MessageDao, aiManager *aihelper.AIHelperManager) *SessionService {
	return &SessionService{
		sessionDao: sessionDao,
		messageDao: messageDao,
		aiManager:  aiManager,
	}
}

func (s *SessionService) GetUserSessionsByUserName(ctx context.Context, userName string) ([]model.SessionInfo, error) {
	sessions, err := s.sessionDao.GetSessionsByUserName(ctx, userName)
	if err != nil {
		return nil, err
	}

	infos := make([]model.SessionInfo, 0, len(sessions))
	for _, sess := range sessions {
		title := strings.TrimSpace(sess.Title)
		if title == "" {
			title = sess.ID
		}
		infos = append(infos, model.SessionInfo{SessionID: sess.ID, Title: title})
	}
	return infos, nil
}

func (s *SessionService) RenameSession(ctx context.Context, userName string, sessionID string, title string) code.Code {
	sid := strings.TrimSpace(sessionID)
	newTitle := strings.TrimSpace(title)
	if sid == "" || newTitle == "" {
		return code.CodeInvalidParams
	}
	if len([]rune(newTitle)) > 50 {
		// 防止超长（DB 是 varchar(100)，这里更保守一点）
		newTitle = string([]rune(newTitle)[:50])
	}

	// ownership check
	sess, err := s.sessionDao.GetSessionByID(ctx, sid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return code.CodeRecordNotFound
		}
		log.Println("RenameSession GetSessionByID error:", err)
		return code.CodeServerBusy
	}
	if sess.UserName != userName {
		return code.CodeForbidden
	}

	if err := s.sessionDao.UpdateSessionTitle(ctx, userName, sid, newTitle); err != nil {
		log.Println("RenameSession UpdateSessionTitle error:", err)
		return code.CodeServerBusy
	}
	return code.CodeSuccess
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

	//3：生成AI回复
	aiResponse, err_ := helper.GenerateResponse(userName, ctx, withJDProfile(userQuestion, jdProfile))
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

	_, err_ := helper.StreamResponse(userName, ctx, cb, withJDProfile(userQuestion, jdProfile))
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

	//2：生成AI回复
	aiResponse, err_ := helper.GenerateResponse(userName, ctx, withJDProfile(userQuestion, jdProfile))
	if err_ != nil {
		log.Println("ChatSend GenerateResponse error:", err_)
		return "", code.AIModelFail
	}

	return aiResponse.Content, code.CodeSuccess
}

func (s *SessionService) GetChatHistory(ctx context.Context, userName string, sessionID string) ([]model.History, code.Code) {
	// 优先从 DB 读取（支持重启后复查；同时避免仅依赖内存 helper）
	if s.messageDao != nil {
		msgs, err := s.messageDao.GetMessagesBySessionIDAndUserName(ctx, sessionID, userName)
		if err == nil && len(msgs) > 0 {
			history := make([]model.History, 0, len(msgs))
			for _, m := range msgs {
				history = append(history, model.History{IsUser: m.IsUser, Content: m.Content})
			}
			return history, code.CodeSuccess
		}
	}

	// fallback：内存 helper
	helper, exists := s.aiManager.GetAIHelper(userName, sessionID)
	if !exists {
		return nil, code.CodeRecordNotFound
	}

	messages := helper.GetMessages()
	history := make([]model.History, 0, len(messages))
	for _, msg := range messages {
		history = append(history, model.History{IsUser: msg.IsUser, Content: msg.Content})
	}
	return history, code.CodeSuccess
}

type interviewReportLLMOutput struct {
	OverallScore   any                        `json:"overallScore"`
	Dimensions     []model.InterviewReportDimension `json:"dimensions"`
	Summary        string                     `json:"summary"`
	Strengths      []string                   `json:"strengths"`
	Risks          []string                   `json:"risks"`
	Suggestions    []string                   `json:"suggestions"`
	Recommendation string                     `json:"recommendation"`
	Detail         string                     `json:"detail"`
}

func clampScore(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func parseAnyInt(v any) (int, bool) {
	switch t := v.(type) {
	case float64:
		return int(t), true
	case float32:
		return int(t), true
	case int:
		return t, true
	case int64:
		return int(t), true
	case json.Number:
		i, err := t.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func buildTranscriptFromMessages(msgs []model.Message) string {
	var b strings.Builder
	for _, m := range msgs {
		role := "面试官"
		if m.IsUser {
			role = "候选人"
		}
		line := strings.TrimSpace(m.Content)
		if line == "" {
			continue
		}
		b.WriteString("[")
		b.WriteString(role)
		b.WriteString("] ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func (s *SessionService) GenerateInterviewReport(ctx context.Context, userName string, sessionID string, modelType string, jdProfile string) (*model.InterviewReport, code.Code) {
	sid := strings.TrimSpace(sessionID)
	if sid == "" {
		return nil, code.CodeInvalidParams
	}
	if modelType == "" {
		modelType = "1"
	}

	// ownership check
	sess, err := s.sessionDao.GetSessionByID(ctx, sid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, code.CodeRecordNotFound
		}
		log.Println("GenerateInterviewReport GetSessionByID error:", err)
		return nil, code.CodeServerBusy
	}
	if sess.UserName != userName {
		return nil, code.CodeForbidden
	}

	var dbMsgs []model.Message
	if s.messageDao != nil {
		dbMsgs, err = s.messageDao.GetMessagesBySessionIDAndUserName(ctx, sid, userName)
		if err != nil {
			log.Println("GenerateInterviewReport GetMessagesBySessionIDAndUserName error:", err)
		}
	}

	// 如果内存中消息更多（DB 可能异步落库未完成），优先用内存消息
	if helper, ok := s.aiManager.GetAIHelper(userName, sid); ok {
		memMsgs := helper.GetMessages()
		if len(memMsgs) > len(dbMsgs) {
			merged := make([]model.Message, 0, len(memMsgs))
			for _, m := range memMsgs {
				merged = append(merged, *m)
			}
			dbMsgs = merged
		}
	}

	transcript := buildTranscriptFromMessages(dbMsgs)
	if strings.TrimSpace(transcript) == "" {
		return nil, code.CodeRecordNotFound
	}

	jdText := strings.TrimSpace(jdProfile)
	if jdText == "" {
		jdText = "（空）"
	}

	dimensionNames := []string{"技术基础", "工程实践", "问题解决", "沟通表达", "学习能力", "岗位匹配度"}

	prompt := `你是一名资深技术面试官与面试评审。
请根据【面试对话记录】与【岗位画像】生成一份“面试评分评价报告”。

强约束：
1) 仅返回 JSON（不要 markdown，不要解释文本）。
2) 维度必须且只能是固定 6 项，并按以下顺序输出：技术基础、工程实践、问题解决、沟通表达、学习能力、岗位匹配度。
3) 每个维度 score 为 0~100 的整数。
4) 字段结构必须满足：
{
  "overallScore": 0,
  "dimensions": [
    {"name": "技术基础", "score": 0, "comment": ""},
    {"name": "工程实践", "score": 0, "comment": ""},
    {"name": "问题解决", "score": 0, "comment": ""},
    {"name": "沟通表达", "score": 0, "comment": ""},
    {"name": "学习能力", "score": 0, "comment": ""},
    {"name": "岗位匹配度", "score": 0, "comment": ""}
  ],
  "summary": "",
  "strengths": [""],
  "risks": [""],
  "suggestions": [""],
  "recommendation": "",
  "detail": ""
}

【岗位画像】
` + jdText + `

【面试对话记录】
` + transcript + `
`

	respText, err := aihelper.ParseTextWithModel(ctx, modelType, prompt, map[string]interface{}{})
	if err != nil {
		log.Println("GenerateInterviewReport ParseTextWithModel error:", err)
		return nil, code.AIModelFail
	}

	jsonText := extractJSONObject(respText)
	if jsonText == "" {
		// 尝试从 ```json ...``` 中提取
		re := regexp.MustCompile("(?s)```json\\s*(\\{.*?\\})\\s*```")
		m := re.FindStringSubmatch(respText)
		if len(m) >= 2 {
			jsonText = m[1]
		}
	}
	if jsonText == "" {
		log.Println("GenerateInterviewReport invalid json response:", respText)
		return nil, code.AIModelFail
	}

	var out interviewReportLLMOutput
	dec := json.NewDecoder(strings.NewReader(jsonText))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		log.Println("GenerateInterviewReport decode error:", err, " raw:", respText)
		return nil, code.AIModelFail
	}

	// normalize dimensions
	dimMap := map[string]model.InterviewReportDimension{}
	for _, d := range out.Dimensions {
		name := strings.TrimSpace(d.Name)
		if name == "" {
			continue
		}
		d.Score = clampScore(d.Score)
		dimMap[name] = d
	}

	finalDims := make([]model.InterviewReportDimension, 0, len(dimensionNames))
	for _, name := range dimensionNames {
		d, ok := dimMap[name]
		if !ok {
			d = model.InterviewReportDimension{Name: name, Score: 0, Comment: "（模型未返回该维度，已补齐默认值）"}
		}
		d.Score = clampScore(d.Score)
		finalDims = append(finalDims, d)
	}

	overall := 0
	if v, ok := parseAnyInt(out.OverallScore); ok {
		overall = clampScore(v)
	} else {
		// fallback average
		sum := 0
		for _, d := range finalDims {
			sum += d.Score
		}
		overall = int(float64(sum)/float64(len(finalDims)) + 0.5)
	}

	report := &model.InterviewReport{
		SessionID:     sid,
		OverallScore:  overall,
		Dimensions:    finalDims,
		Summary:       strings.TrimSpace(out.Summary),
		Strengths:     out.Strengths,
		Risks:         out.Risks,
		Suggestions:   out.Suggestions,
		Recommendation: strings.TrimSpace(out.Recommendation),
		Detail:        strings.TrimSpace(out.Detail),
	}

	// scores mapping for frontend
	nameToKey := map[string]func(*model.InterviewReportScores, int){
		"技术基础":  func(s *model.InterviewReportScores, v int) { s.Tech = v },
		"工程实践":  func(s *model.InterviewReportScores, v int) { s.Eng = v },
		"问题解决":  func(s *model.InterviewReportScores, v int) { s.PS = v },
		"沟通表达":  func(s *model.InterviewReportScores, v int) { s.Comm = v },
		"学习能力":  func(s *model.InterviewReportScores, v int) { s.Learn = v },
		"岗位匹配度": func(s *model.InterviewReportScores, v int) { s.Fit = v },
	}
	var scores model.InterviewReportScores
	for _, d := range finalDims {
		if fn, ok := nameToKey[d.Name]; ok {
			fn(&scores, d.Score)
		}
	}
	report.Scores = scores

	return report, code.CodeSuccess
}

// 流式发送消息
func (s *SessionService) ChatStreamSend(ctx context.Context, userName string, sessionID string, userQuestion string, modelType string, jdProfile string, writer http.ResponseWriter) code.Code {

	return s.StreamMessageToExistingSession(ctx, userName, sessionID, userQuestion, modelType, jdProfile, writer)
}

// 拼接用户问题和岗位画像
func withJDProfile(userQuestion string, jdProfile string) string {
	if strings.TrimSpace(jdProfile) == "" {
		return userQuestion
	}
	return "【岗位画像】\n" + jdProfile + "\n\n【用户问题】\n" + userQuestion
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
