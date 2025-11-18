package embedding

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino-ext/components/embedding/openai"
	"github.com/cloudwego/eino/components/embedding"
)

type EmBedderConfig struct {
	Provider   string
	APIKey     string
	Model      string
	BaseURL    string
	APIVersion string
	ByAzure    bool
	Timeout    time.Duration
	Dimensions int
}

// NewEmbedder 创建 Embedder，当前支持 OpenAI 与 Ark。
func NewEmbedder(ctx context.Context, cfg *EmBedderConfig) (embedding.Embedder, error) {
	switch cfg.Provider {
	case "openai":
		return openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
			APIKey:     cfg.APIKey,
			Model:      cfg.Model,
			BaseURL:    cfg.BaseURL,
			APIVersion: "", // 如需 Azure：在 config 中设置并传入
			ByAzure:    false,
			Timeout:    cfg.Timeout,
			Dimensions: &cfg.Dimensions,
		})
	case "ark":
		// Ark 不暴露 Dimensions 配置；按默认模型维度即可
		t := cfg.Timeout
		return ark.NewEmbedder(ctx, &ark.EmbeddingConfig{
			APIKey:  cfg.APIKey,
			Model:   cfg.Model,
			BaseURL: cfg.BaseURL,
			Region:  "cn-beijing",
			Timeout: &t,
		})
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", cfg.Provider)
	}
}
