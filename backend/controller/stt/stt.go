package stt

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/milabo0718/offer-pilot/backend/common/stt"
)

// TranscribeResponse STT 识别结果响应
type TranscribeResponse struct {
	Text string `json:"text"` // 识别出的文本内容
}

// STTController 负责处理 STT 相关 HTTP 请求
type STTController struct {
	client *stt.STTClient
}

// NewSTTController 创建 STTController 实例
func NewSTTController(client *stt.STTClient) *STTController {
	return &STTController{client: client}
}

// TranscribeAudio 将上传的音频文件转写为文本
//
// POST /api/v1/ai/stt/transcribe
// Content-Type: multipart/form-data
//
// 表单字段:
//
//	audio  - 音频文件（必填，支持 wav / mp3 / m4a / webm / ogg）
//
// 响应:
//
//	{"text": "用户作答的文本内容"}
//
// 面试流程中，前端录制用户麦克风音频后通过此接口上传，
// 后端调用 DashScope Paraformer 进行识别并返回文字，
// 再将该文字作为用户回答传入对话流程。
func (sc *STTController) TranscribeAudio(ctx *gin.Context) {
	fileHeader, err := ctx.FormFile("audio")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "缺少 audio 字段，请以 multipart/form-data 上传音频文件"})
		return
	}

	// 限制上传大小：25 MB（DashScope 单次转写上限）
	const maxFileSize = 25 << 20
	if fileHeader.Size > maxFileSize {
		ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "音频文件超过 25 MB 限制"})
		return
	}

	f, err := fileHeader.Open()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "打开音频文件失败"})
		return
	}
	defer f.Close()

	audioData := make([]byte, fileHeader.Size)
	if _, err = f.Read(audioData); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "读取音频数据失败"})
		return
	}

	text, err := sc.client.Transcribe(ctx.Request.Context(), audioData, fileHeader.Filename)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, TranscribeResponse{Text: text})
}
