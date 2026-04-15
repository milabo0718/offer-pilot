package tts

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/milabo0718/offer-pilot/backend/common/tts"
)

// SynthesizeRequest TTS 合成请求参数
type SynthesizeRequest struct {
	Text string `json:"text" binding:"required"` // 待合成的文本
}

// TTSController 负责处理 TTS 相关 HTTP 请求
type TTSController struct {
	client *tts.TTSClient
}

// NewTTSController 创建 TTSController 实例
func NewTTSController(client *tts.TTSClient) *TTSController {
	return &TTSController{client: client}
}

// Synthesize 将文字一次性合成为语音，返回完整的 MP3 音频
//
// POST /api/v1/ai/tts/synthesize
// Content-Type: application/json
//
// 请求体: {"text": "面试官的问题文本"}
// 响应:   audio/mpeg 二进制音频数据
//
// 适合文本较短的场景，前端拿到完整音频后通过 <audio> 标签或 Blob URL 播放
func (tc *TTSController) Synthesize(ctx *gin.Context) {
	req := new(SynthesizeRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：text 字段不能为空"})
		return
	}

	audio, err := tc.client.Synthesize(ctx.Request.Context(), req.Text)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Data(http.StatusOK, "audio/mpeg", audio)
}

// SynthesizeStream 将文字以流式方式合成语音，返回 chunked MP3 音频流
//
// POST /api/v1/ai/tts/stream
// Content-Type: application/json
//
// 请求体: {"text": "面试官的问题文本"}
// 响应:   audio/mpeg 分块流式音频
//
// 适合文本较长的场景，前端通过 MediaSource API 或 fetch streaming 实现低延迟播放
func (tc *TTSController) SynthesizeStream(ctx *gin.Context) {
	req := new(SynthesizeRequest)
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "参数错误：text 字段不能为空"})
		return
	}

	ctx.Header("Content-Type", "audio/mpeg")
	ctx.Header("Transfer-Encoding", "chunked")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Access-Control-Allow-Origin", "*")
	ctx.Header("X-Accel-Buffering", "no")

	w := ctx.Writer
	if err := tc.client.SynthesizeStream(ctx.Request.Context(), req.Text, w); err != nil {
		// 响应头已发出，无法再写 JSON；仅记录日志，客户端可感知连接中断
		_ = err
		return
	}

	w.Flush()
}
