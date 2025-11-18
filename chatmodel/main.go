package main

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

func main() {

}

type modelConfig struct {
	Provider     string
	APIKey       string
	BaseURL      string
	Model        string
	Timeout      time.Duration
	CustomHeader map[string]string
	APIVersion   string
	MaxTokens    int
	Temperature  float32
}

func NewChatModel(ctx context.Context, cfg *modelConfig) (model.BaseChatModel, error) {
	switch cfg.Provider {
	case "openai":
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:              cfg.APIKey,
			Timeout:             cfg.Timeout,
			Model:               cfg.Model,
			BaseURL:             cfg.BaseURL,
			APIVersion:          cfg.APIVersion,
			MaxCompletionTokens: &cfg.MaxTokens,
			Temperature:         &cfg.Temperature,
		})
	case "azure_openai":
		return openai.NewChatModel(ctx, &openai.ChatModelConfig{
			APIKey:              cfg.APIKey,
			Timeout:             cfg.Timeout,
			Model:               cfg.Model,
			BaseURL:             cfg.BaseURL,
			APIVersion:          cfg.APIVersion,
			MaxCompletionTokens: &cfg.MaxTokens,
			Temperature:         &cfg.Temperature,
		})
	case "ollama":
		return ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
			BaseURL: cfg.BaseURL, // e.g. http://localhost:11434
			Model:   cfg.Model,   // e.g. "llama3"
			Timeout: cfg.Timeout,
		})
	case "ark":
		return ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey:      cfg.APIKey,
			BaseURL:     cfg.BaseURL,
			Model:       cfg.Model,
			Timeout:     &cfg.Timeout,
			MaxTokens:   &cfg.MaxTokens,
			Temperature: &cfg.Temperature,
		})
	default:
		return nil, fmt.Errorf("unsupported llm provider: %s", cfg.Provider)
	}
}
