package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/milabo0718/offer-pilot/backend/common/stt"
	"github.com/milabo0718/offer-pilot/backend/config"
)

func main() {
	conf, err := config.InitConfig("./config")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	client := stt.NewSTTClient(stt.STTConfig{
		APIKey:    conf.STTConfig.APIKey,
		ModelName: conf.STTConfig.ModelName,
		Language:  conf.STTConfig.Language,
	})

	audioFile := "test_output.mp3"
	audioData, err := os.ReadFile(audioFile)
	if err != nil {
		log.Fatalf("读取音频文件失败: %v\n请先运行 go run cmd/tts_test/main.go 生成 test_output.mp3", err)
	}

	fmt.Printf("正在识别音频: %s (%d 字节)\n", audioFile, len(audioData))

	text, err := client.Transcribe(context.Background(), audioData, audioFile)
	if err != nil {
		log.Fatalf("STT 识别失败: %v", err)
	}

	fmt.Printf("识别结果: %s\n", text)
}
