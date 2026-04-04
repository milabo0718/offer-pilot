package router

import (
	"github.com/milabo0718/offer-pilot/backend/controller/session"

	"github.com/gin-gonic/gin"
)

func AIRouter(r *gin.RouterGroup, sc *session.SessionController) {

	// 聊天相关接口
	{
		r.GET("/chat/sessions", sc.GetUserSessionsByUserName)
		r.POST("/chat/send-new-session", sc.CreateSessionAndSendMessage)
		r.POST("/chat/send", sc.ChatSend)
		r.POST("/chat/history", sc.ChatHistory)
		r.POST("/chat/send-stream-new-session", sc.CreateStreamSessionAndSendMessage)
		r.POST("/chat/send-stream", sc.ChatStreamSend)
	}
}
