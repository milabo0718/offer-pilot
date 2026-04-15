package tts

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DashScope CosyVoice 非实时语音合成 HTTP 端点
const dashscopeTTSEndpoint = "https://dashscope.aliyuncs.com/api/v1/services/audio/tts/SpeechSynthesizer"

// TTSConfig 阿里云百炼 TTS 客户端配置
type TTSConfig struct {
	APIKey     string `mapstructure:"apiKey"`
	ModelName  string `mapstructure:"modelName"`
	Voice      string `mapstructure:"voice"`
	Format     string `mapstructure:"format"`
	SampleRate int    `mapstructure:"sampleRate"`
}

// ttsRequest 请求体结构（voice/format/sample_rate 位于 input 内）
type ttsRequest struct {
	Model string   `json:"model"`
	Input ttsInput `json:"input"`
}

type ttsInput struct {
	Text       string `json:"text"`
	Voice      string `json:"voice"`
	Format     string `json:"format"`
	SampleRate int    `json:"sample_rate"`
}

// ttsResponse 非流式返回体
type ttsResponse struct {
	RequestID string    `json:"request_id"`
	Output    ttsOutput `json:"output"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
}

type ttsOutput struct {
	Audio struct {
		URL string `json:"url"`
	} `json:"audio"`
}

// ttsSSEEvent 流式 SSE 事件
type ttsSSEEvent struct {
	RequestID string `json:"request_id"`
	Output    struct {
		FinishReason string `json:"finish_reason"`
		Audio        struct {
			URL string `json:"url"`
		} `json:"audio"`
	} `json:"output"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// TTSClient 阿里云百炼 TTS 客户端，封装 CosyVoice 语音合成能力
type TTSClient struct {
	config     TTSConfig
	httpClient *http.Client
}

// NewTTSClient 创建新的 TTS 客户端，未设置的参数使用默认值
func NewTTSClient(cfg TTSConfig) *TTSClient {
	if cfg.ModelName == "" {
		cfg.ModelName = "cosyvoice-v2"
	}
	if cfg.Voice == "" {
		cfg.Voice = "longxiaochun"
	}
	if cfg.Format == "" {
		cfg.Format = "mp3"
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 22050
	}
	return &TTSClient{
		config:     cfg,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// Synthesize 非流式合成：返回音频文件的 URL，再下载为字节
func (c *TTSClient) Synthesize(ctx context.Context, text string) ([]byte, error) {
	bodyBytes, err := c.buildRequestBody(text)
	if err != nil {
		return nil, fmt.Errorf("tts: 构建请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dashscopeTTSEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("tts: 创建请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tts: HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tts: 读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tts: API 返回异常状态码 %d: %s", resp.StatusCode, string(respBody))
	}

	var result ttsResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("tts: 解析响应失败: %w", err)
	}
	if result.Code != "" {
		return nil, fmt.Errorf("tts: API 错误 %s: %s", result.Code, result.Message)
	}

	audioURL := result.Output.Audio.URL
	if audioURL == "" {
		return nil, fmt.Errorf("tts: API 未返回音频 URL, 响应: %s", string(respBody))
	}

	return c.downloadAudio(ctx, audioURL)
}

// SynthesizeStream 流式合成：SSE 方式逐句返回音频 URL，逐段下载并写入 w
func (c *TTSClient) SynthesizeStream(ctx context.Context, text string, w io.Writer) error {
	bodyBytes, err := c.buildRequestBody(text)
	if err != nil {
		return fmt.Errorf("tts: 构建请求体失败: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, dashscopeTTSEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("tts: 创建请求失败: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DashScope-SSE", "enable")

	streamClient := &http.Client{}
	resp, err := streamClient.Do(req)
	if err != nil {
		return fmt.Errorf("tts: HTTP 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("tts: API 返回异常状态码 %d: %s", resp.StatusCode, string(errBody))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimPrefix(line, "data:")

		var event ttsSSEEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		if event.Code != "" {
			return fmt.Errorf("tts: 流式 API 错误 %s: %s", event.Code, event.Message)
		}

		audioURL := event.Output.Audio.URL
		if audioURL == "" {
			continue
		}

		audioData, err := c.downloadAudio(ctx, audioURL)
		if err != nil {
			return fmt.Errorf("tts: 下载流式音频分片失败: %w", err)
		}
		if _, err := w.Write(audioData); err != nil {
			return fmt.Errorf("tts: 写入音频流失败: %w", err)
		}
	}

	return scanner.Err()
}

func (c *TTSClient) buildRequestBody(text string) ([]byte, error) {
	reqBody := ttsRequest{
		Model: c.config.ModelName,
		Input: ttsInput{
			Text:       text,
			Voice:      c.config.Voice,
			Format:     c.config.Format,
			SampleRate: c.config.SampleRate,
		},
	}
	return json.Marshal(reqBody)
}

// downloadAudio 从 OSS URL 下载音频字节
func (c *TTSClient) downloadAudio(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("下载音频失败, 状态码: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
