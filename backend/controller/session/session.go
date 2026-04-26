package session

import (
	"fmt"
	"net/http"

	"github.com/milabo0718/offer-pilot/backend/common/code"
	"github.com/milabo0718/offer-pilot/backend/controller"
	"github.com/milabo0718/offer-pilot/backend/model"
	"github.com/milabo0718/offer-pilot/backend/service/session"

	"github.com/gin-gonic/gin"
)

type (
	GetUserSessionsResponse struct {
		controller.Response
		Sessions []model.SessionInfo `json:"sessions,omitempty"`
	}
	CreateSessionAndSendMessageRequest struct {
		UserQuestion string `json:"question" binding:"required"`  // 用户问题;
		ModelType    string `json:"modelType" binding:"required"` // 模型类型;
		JDProfile    string `json:"jdProfile,omitempty"`          // JD解析画像(JSON字符串)
	}

	CreateSessionAndSendMessageResponse struct {
		AiInformation string `json:"Information,omitempty"` // AI回答
		SessionID     string `json:"sessionId,omitempty"`   // 当前会话ID
		controller.Response
	}

	ChatSendRequest struct {
		UserQuestion string `json:"question" binding:"required"`            // 用户问题;
		ModelType    string `json:"modelType" binding:"required"`           // 模型类型;
		SessionID    string `json:"sessionId,omitempty" binding:"required"` // 当前会话ID
		JDProfile    string `json:"jdProfile,omitempty"`                    // JD解析画像(JSON字符串)
	}

	ChatSendResponse struct {
		AiInformation string `json:"Information,omitempty"` // AI回答
		controller.Response
	}

	ChatHistoryRequest struct {
		SessionID string `json:"sessionId,omitempty" binding:"required"` // 当前会话ID
	}
	ChatHistoryResponse struct {
		History []model.History `json:"history"`
		controller.Response
	}

	JDParseRequest struct {
		JDText    string `json:"jdText" binding:"required"`
		ModelType string `json:"modelType"`
	}

	JDParseResponse struct {
		Data *model.JDParseResult `json:"data,omitempty"`
		controller.Response
	}

	InterviewReportRequest struct {
		SessionID string `json:"sessionId" binding:"required"`
		ModelType string `json:"modelType"`
		JDProfile string `json:"jdProfile,omitempty"`
		Force     bool   `json:"force,omitempty"`
	}

	InterviewReportResponse struct {
		Data *model.InterviewReportData `json:"data,omitempty"`
		controller.Response
	}
)

type SessionController struct {
	sessionService *session.SessionService
}

func NewSessionController(sessionService *session.SessionService) *SessionController {
	return &SessionController{
		sessionService: sessionService,
	}
}

// 获取用户的会话列表
func (sc *SessionController) GetUserSessionsByUserName(ctx *gin.Context) {
	res := new(GetUserSessionsResponse)
	userName := ctx.GetString("userName") // From JWT middleware

	userSessions, err := sc.sessionService.GetUserSessionsByUserName(ctx, userName)
	if err != nil {
		ctx.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.Sessions = userSessions
	ctx.JSON(http.StatusOK, res)
}

// 创建会话并发送消息
func (sc *SessionController) CreateSessionAndSendMessage(ctx *gin.Context) {
	req := new(CreateSessionAndSendMessageRequest)
	res := new(CreateSessionAndSendMessageResponse)
	userName := ctx.GetString("userName") // From JWT middleware
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}
	//内部会创建会话并发送消息，并会将AI回答、当前会话返回
	session_id, aiInformation, code_ := sc.sessionService.CreateSessionAndSendMessage(ctx, userName, req.UserQuestion, req.ModelType, req.JDProfile)

	if code_ != code.CodeSuccess {
		ctx.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	res.AiInformation = aiInformation
	res.SessionID = session_id
	ctx.JSON(http.StatusOK, res)
}

// 创建会话并发送消息（流式输出版本）
func (sc *SessionController) CreateStreamSessionAndSendMessage(ctx *gin.Context) {
	req := new(CreateSessionAndSendMessageRequest)
	userName := ctx.GetString("userName") // From JWT middleware
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusOK, gin.H{"error": "Invalid parameters"})
		return
	}

	// 设置SSE头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("X-Accel-Buffering", "no") // 禁止代理缓存

	// 先创建会话并立即把 sessionId 下发给前端，随后再开始流式输出
	sessionID, code_ := sc.sessionService.CreateStreamSessionOnly(ctx, userName, req.UserQuestion)
	if code_ != code.CodeSuccess {
		ctx.SSEvent("error", gin.H{"message": "Failed to create session"})
		return
	}

	// 先把 sessionId 通过 data 事件发送给前端，前端据此绑定当前会话，侧边栏即可出现新标签
	ctx.Writer.WriteString(fmt.Sprintf("data: {\"sessionId\": \"%s\"}\n\n", sessionID))
	ctx.Writer.Flush()

	// 然后开始把本次回答进行流式发送（包含最后的 [DONE]）
	code_ = sc.sessionService.StreamMessageToExistingSession(ctx, userName, sessionID, req.UserQuestion, req.ModelType, req.JDProfile, http.ResponseWriter(ctx.Writer))
	if code_ != code.CodeSuccess {
		ctx.SSEvent("error", gin.H{"message": "Failed to send message"})
		return
	}
}

// 发送消息（非流式版本）
func (sc *SessionController) ChatSend(ctx *gin.Context) {
	req := new(ChatSendRequest)
	res := new(ChatSendResponse)
	userName := ctx.GetString("userName") // From JWT middleware
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}
	// 发送消息，并会将AI回答返回
	aiInformation, code_ := sc.sessionService.ChatSend(ctx, userName, req.SessionID, req.UserQuestion, req.ModelType, req.JDProfile)

	if code_ != code.CodeSuccess {
		ctx.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	res.AiInformation = aiInformation
	ctx.JSON(http.StatusOK, res)
}

// 发送消息（流式输出版本）
func (sc *SessionController) ChatStreamSend(ctx *gin.Context) {
	req := new(ChatSendRequest)
	userName := ctx.GetString("userName") // From JWT middleware
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusOK, gin.H{"error": "Invalid parameters"})
		return
	}

	// 设置SSE头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("X-Accel-Buffering", "no") // 禁止代理缓存

	code_ := sc.sessionService.ChatStreamSend(ctx, userName, req.SessionID, req.UserQuestion, req.ModelType, req.JDProfile, http.ResponseWriter(ctx.Writer))
	if code_ != code.CodeSuccess {
		ctx.SSEvent("error", gin.H{"message": "Failed to send message"})
		return
	}

}

// 获取会话历史记录
func (sc *SessionController) ChatHistory(ctx *gin.Context) {
	req := new(ChatHistoryRequest)
	res := new(ChatHistoryResponse)
	userName := ctx.GetString("userName") // From JWT middleware
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}
	history, code_ := sc.sessionService.GetChatHistory(ctx, userName, req.SessionID)
	if code_ != code.CodeSuccess {
		ctx.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	res.History = history
	ctx.JSON(http.StatusOK, res)
}

// 解析岗位JD（文本）
func (sc *SessionController) ParseJD(ctx *gin.Context) {
	req := new(JDParseRequest)
	res := new(JDParseResponse)
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}
	if req.ModelType == "" {
		req.ModelType = "1"
	}

	result, code_ := sc.sessionService.ParseJD(ctx, req.JDText, req.ModelType)
	if code_ != code.CodeSuccess {
		ctx.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	res.Data = result
	ctx.JSON(http.StatusOK, res)
}

// GenerateInterviewReport 生成面试评分报告和能力雷达图数据。
func (sc *SessionController) GenerateInterviewReport(ctx *gin.Context) {
	req := new(InterviewReportRequest)
	res := new(InterviewReportResponse)
	userName := ctx.GetString("userName")
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}
	if req.ModelType == "" {
		req.ModelType = "1"
	}

	report, code_ := sc.sessionService.GenerateInterviewReport(ctx, userName, req.SessionID, req.ModelType, req.JDProfile, req.Force)
	if code_ != code.CodeSuccess {
		ctx.JSON(http.StatusOK, res.CodeOf(code_))
		return
	}

	res.Success()
	res.Data = report
	ctx.JSON(http.StatusOK, res)
}
