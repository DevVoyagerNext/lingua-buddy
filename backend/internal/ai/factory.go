package ai

import (
	"log"

	"lingua-buddy/internal/config"
)

// NewProvider 按配置选择 Provider：无 API Key 或显式 mock 时使用 Mock，便于离线运行。
func NewProvider(cfg config.ProviderConfig) Provider {
	if cfg.Provider == "mock" || cfg.APIKey == "" {
		log.Println("AI Provider: 使用 Mock（未配置 AI_API_KEY 或 AI_PROVIDER=mock）")
		return NewMock()
	}
	log.Printf("AI Provider: DashScope model=%s", cfg.Model)
	return NewDashScope(cfg.APIBase, cfg.APIKey, cfg.Model)
}
