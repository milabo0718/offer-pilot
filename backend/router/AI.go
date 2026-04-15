package router

import (
	ragcontroller "github.com/milabo0718/offer-pilot/backend/controller/rag"
	"github.com/milabo0718/offer-pilot/backend/controller/session"
	sttcontroller "github.com/milabo0718/offer-pilot/backend/controller/stt"
	ttscontroller "github.com/milabo0718/offer-pilot/backend/controller/tts"

	"github.com/gin-gonic/gin"
)

func AIRouter(
	r *gin.RouterGroup,
	sc *session.SessionController,
	rc *ragcontroller.RAGController,
	tc *ttscontroller.TTSController,
	stc *sttcontroller.STTController,
) {
	// 聊天相关接口
	{
		r.POST("/jd/parse", sc.ParseJD)
		r.GET("/chat/sessions", sc.GetUserSessionsByUserName)
		r.POST("/chat/send-new-session", sc.CreateSessionAndSendMessage)
		r.POST("/chat/send", sc.ChatSend)
		r.POST("/chat/history", sc.ChatHistory)
		r.POST("/chat/send-stream-new-session", sc.CreateStreamSessionAndSendMessage)
		r.POST("/chat/send-stream", sc.ChatStreamSend)
	}

	// TTS 语音合成接口
	// POST /api/v1/ai/tts/synthesize  返回完整 MP3（短文本）
	// POST /api/v1/ai/tts/stream      流式返回 MP3（长文本低延迟）
	{
		r.POST("/tts/synthesize", tc.Synthesize)
		r.POST("/tts/stream", tc.SynthesizeStream)
	}

	// STT 语音识别接口
	// POST /api/v1/ai/stt/transcribe  上传音频，返回识别文本
	{
		r.POST("/stt/transcribe", stc.TranscribeAudio)
	}

	// RAG 相关接口
	RAGRouter(r, rc)
}
