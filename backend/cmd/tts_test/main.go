package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/milabo0718/offer-pilot/backend/common/tts"
	"github.com/milabo0718/offer-pilot/backend/config"
)

func main() {
	conf, err := config.InitConfig("./config")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	client := tts.NewTTSClient(tts.TTSConfig{
		APIKey:     conf.TTSConfig.APIKey,
		ModelName:  conf.TTSConfig.ModelName,
		Voice:      conf.TTSConfig.Voice,
		Format:     conf.TTSConfig.Format,
		SampleRate: conf.TTSConfig.SampleRate,
	})

	text := "你好，我是面试官，请做一下自我介绍。"
	fmt.Printf("正在合成语音: \"%s\"\n", text)

	audio, err := client.Synthesize(context.Background(), text)
	if err != nil {
		log.Fatalf("TTS 合成失败: %v", err)
	}

	outputFile := "test_output.mp3"
	if err := os.WriteFile(outputFile, audio, 0644); err != nil {
		log.Fatalf("保存文件失败: %v", err)
	}

	fmt.Printf("合成成功，已保存 %s (%d 字节)\n", outputFile, len(audio))
	fmt.Println("你可以用播放器打开 test_output.mp3 试听效果")
}
