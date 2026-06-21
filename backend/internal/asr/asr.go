// Package asr 定义语音识别 Provider 抽象与实现。
// 真实实现为阿里云 DashScope Paraformer（录音文件识别，异步，需公网音频 URL，走 OSS）。
package asr

import (
	"context"
	"errors"
	"log"

	"lingua-buddy/internal/config"
)

// ErrFailed 识别失败（可重试）。
var ErrFailed = errors.New("asr failed")

// Transcript 识别结果。
type Transcript struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}

// Provider 语音识别能力抽象。audioURL 为公网可访问的音频地址（OSS 签名 URL）。
type Provider interface {
	Transcribe(ctx context.Context, audioURL, language string) (Transcript, error)
}

// Mock 返回占位识别文本（仅在显式 ASR_PROVIDER=mock 时使用）。
type Mock struct{}

// NewMock 构造。
func NewMock() *Mock { return &Mock{} }

// Transcribe 返回占位结果。
func (m *Mock) Transcribe(_ context.Context, _ string, language string) (Transcript, error) {
	if language == "" || language == "auto" {
		language = "en"
	}
	return Transcript{Text: "[mock 识别文本] this is a sample transcript.", Language: language}, nil
}

// NewProvider 按配置选择 Provider：配置了密钥则用真实 Paraformer，显式 mock 才用 Mock。
func NewProvider(cfg config.ProviderConfig) Provider {
	if cfg.Provider == "mock" {
		log.Println("ASR Provider: 使用 Mock（显式 ASR_PROVIDER=mock）")
		return NewMock()
	}
	if cfg.APIKey == "" {
		log.Println("ASR Provider: 未配置 ASR_API_KEY，回退 Mock")
		return NewMock()
	}
	log.Printf("ASR Provider: 阿里云 DashScope Paraformer model=%s", cfg.Model)
	return NewParaformer(cfg.APIBase, cfg.APIKey, cfg.Model)
}
