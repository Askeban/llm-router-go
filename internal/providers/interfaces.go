package providers

import (
	"context"
	"fmt"
	"github.com/Askeban/llm-router-go/internal/config"
	"strings"
)

type Usage struct {
	PromptTokens, CompletionTokens, TotalTokens int
	ApproxCostUSD                               float64
}
type LLMClient interface {
	Generate(ctx context.Context, prompt string, params map[string]any) (string, Usage, error)
}
type Registry struct{ cfg *config.Config }

func NewRegistry(cfg *config.Config) *Registry { return &Registry{cfg: cfg} }
func (r *Registry) ClientFor(modelID string) (LLMClient, error) {
	id := strings.ToLower(modelID)
	switch {
	case strings.HasPrefix(id, "openai-"):
		return &OpenAI{APIKey: r.cfg.OpenAIKey, BaseURL: r.cfg.OpenAIBaseURL, ModelID: modelID}, nil
	case strings.HasPrefix(id, "anthropic-"):
		return &Anthropic{APIKey: r.cfg.AnthropicKey, ModelID: modelID}, nil
	case strings.HasPrefix(id, "google-"):
		return &Google{APIKey: r.cfg.GoogleAPIKey, ModelID: modelID}, nil
	case strings.HasPrefix(id, "openrouter-"):
		return &OpenRouter{APIKey: r.cfg.OpenRouterKey, ModelID: modelID}, nil
	default:
		return nil, fmt.Errorf("no provider for %s", modelID)
	}
}
