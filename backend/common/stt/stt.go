package stt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// Qwen3-ASR-Flash OpenAI 兼容端点，支持同步 Base64 音频上传
const dashscopeSTTEndpoint = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"

// STTConfig 阿里云百炼 STT（Qwen3-ASR-Flash）客户端配置
type STTConfig struct {
	APIKey    string `mapstructure:"apiKey"`
	ModelName string `mapstructure:"modelName"`
	Language  string `mapstructure:"language"`
}

// STTClient 阿里云百炼语音识别客户端
type STTClient struct {
	config     STTConfig
	httpClient *http.Client
}

// NewSTTClient 创建新的 STT 客户端，未设置的参数使用默认值
func NewSTTClient(cfg STTConfig) *STTClient {
	if cfg.ModelName == "" {
		cfg.ModelName = "qwen3-asr-flash"
	}
	if cfg.Language == "" {
		cfg.Language = "zh"
	}
	return &STTClient{
		config:     cfg,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// chatRequest OpenAI 兼容模式请求体
type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string        `json:"role"`
	Content []chatContent `json:"content"`
}

type chatContent struct {
	Type       string `json:"type"`
	InputAudio string `json:"input_audio,omitempty"`
	Text       string `json:"text,omitempty"`
}

// chatResponse OpenAI 兼容模式响应体
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Transcribe 将音频数据转写为文本
//
// audioData: 音频文件的原始字节（支持 wav / mp3 / m4a / webm / ogg 等格式，≤10 MB）
// filename:  原始文件名（仅用于日志，不影响识别）
// 返回识别出的文本内容
func (c *STTClient) Transcribe(ctx context.Context, audioData []byte, filename string) (string, error) {
	b64Audio := base64.StdEncoding.EncodeToString(audioData)

	// 构造 data URI，Qwen3-ASR-Flash 支持直接传 Base64
	contentType := detectAudioContentType(filename)
	dataURI := "data:" + contentType + ";base64," + b64Audio

	reqBody := chatRequest{
		Model: c.config.ModelName,
		Messages: []chatMessage{
			{
				Role: "user",
				Content: []chatContent{
					{
						Type:       "input_audio",
						InputAudio: dataURI,
					},
				},
			},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("stt: 构建请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dashscopeSTTEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("stt: 创建请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("stt: HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("stt: 读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("stt: API 返回异常状态码 %d: %s", resp.StatusCode, string(respBody))
	}

	var result chatResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("stt: 解析响应失败: %w", err)
	}

	if result.Error != nil {
		return "", fmt.Errorf("stt: API 错误 %s: %s", result.Error.Code, result.Error.Message)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("stt: API 返回空结果, 响应: %s", string(respBody))
	}

	return result.Choices[0].Message.Content, nil
}

func detectAudioContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return "audio/webm"
	}
	contentType := mime.TypeByExtension(ext)
	if strings.TrimSpace(contentType) == "" {
		return "audio/webm"
	}
	return contentType
}
