// Package asr 定义语音识别 Provider 抽象与实现。
// 注意：真实阿里云 Paraformer 录音文件识别为异步且需公网音频 URL（走 OSS）；
// 未配置密钥时使用 Mock，便于本地联调与端到端测试。
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

// Provider 语音识别能力抽象。
type Provider interface {
	Transcribe(ctx context.Context, audio []byte, mime, language string) (Transcript, error)
}

// Mock 返回占位识别文本。
type Mock struct{}

// NewMock 构造。
func NewMock() *Mock { return &Mock{} }

// Transcribe 返回占位结果。
func (m *Mock) Transcribe(_ context.Context, _ []byte, _ string, language string) (Transcript, error) {
	if language == "" || language == "auto" {
		language = "en"
	}
	return Transcript{Text: "[mock 识别文本] this is a sample transcript.", Language: language}, nil
}

// NewProvider 按配置选择 Provider。真实 Paraformer 需 OSS，首版未配置或无密钥时用 Mock。
func NewProvider(cfg config.ProviderConfig) Provider {
	if cfg.APIKey == "" || cfg.Provider == "mock" {
		log.Println("ASR Provider: 使用 Mock（未配置 ASR_API_KEY 或显式 mock）")
		return NewMock()
	}
	// TODO: 接入阿里云 DashScope Paraformer（上传 OSS 取签名 URL → 提交任务 → 轮询）。
	log.Println("ASR Provider: 暂未接入真实 Paraformer，回退 Mock")
	return NewMock()
}
