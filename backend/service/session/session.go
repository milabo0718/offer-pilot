package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/milabo0718/offer-pilot/backend/common/aihelper"
	"github.com/milabo0718/offer-pilot/backend/common/code"
	"github.com/milabo0718/offer-pilot/backend/dao/session"
	"github.com/milabo0718/offer-pilot/backend/model"
	ragservice "github.com/milabo0718/offer-pilot/backend/service/rag"

	"github.com/google/uuid"
)

type SessionService struct {
	sessionDao *session.SessionDao
	aiManager  *aihelper.AIHelperManager
	ragService *ragservice.Service
	ragEnabled bool
	ragTopK    int
}

func NewSessionService(sessionDao *session.SessionDao, aiManager *aihelper.AIHelperManager) *SessionService {
	return &SessionService{
		sessionDao: sessionDao,
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
	//获取用户的所有会话ID
	Sessions := s.aiManager.GetUserSessions(userName)

	var SessionInfos []model.SessionInfo

	for _, session := range Sessions {
		SessionInfos = append(SessionInfos, model.SessionInfo{
			SessionID: session,
			Title:     session, // 暂时用sessionID作为标题，后续重构需要的时候可以更改
		})
	}

	return SessionInfos, nil
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
	// 获取AIHelper中的消息历史
	helper, exists := s.aiManager.GetAIHelper(userName, sessionID)
	if !exists {
		return nil, code.CodeServerBusy
	}

	messages := helper.GetMessages()
	history := make([]model.History, 0, len(messages))

	// 转换消息为历史格式（根据消息顺序或内容判断用户/AI消息）
	for i, msg := range messages {
		isUser := i%2 == 0
		history = append(history, model.History{
			IsUser:  isUser,
			Content: msg.Content,
		})
	}

	return history, code.CodeSuccess
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

// buildChatPrompt 先拼接 JD 画像，再按开关决定是否追加 RAG 召回上下文。
func (s *SessionService) buildChatPrompt(ctx context.Context, userQuestion string, jdProfile string) string {
	base := withJDProfile(userQuestion, jdProfile)

	if !s.ragEnabled || s.ragService == nil {
		return base
	}

	results, err := s.ragService.SearchRelevantChunks(ctx, userQuestion, s.ragTopK)
	if err != nil {
		log.Printf("RAG augment 检索失败，已自动降级: %v", err)
		return base
	}
	if len(results) == 0 {
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

	return b.String()
}
