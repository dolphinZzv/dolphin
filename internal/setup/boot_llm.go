package setup

import (
	"context"

	"dolphin/internal/llm"
)

type LLMBootstrapper struct{}

func (b *LLMBootstrapper) Name() string { return "llm" }
func (b *LLMBootstrapper) Index() int   { return 50 }
func (b *LLMBootstrapper) Bootstrap(ctx context.Context, c *Context) error {
	if c.LLMProvider != nil {
		return nil
	}
	providerName := c.Config.GetString("llm.provider")
	c.LLMProvider = llm.NewProvider(llm.Config{
		Provider:    providerName,
		Model:       c.Config.GetString("llm.model"),
		APIKey:      c.Config.GetString("llm." + providerName + ".api_key"),
		BaseURL:     c.Config.GetString("llm." + providerName + ".base_url"),
		Temperature: c.Config.GetFloat("llm.temperature"),
		MaxTokens:   c.Config.GetInt("llm.max_tokens"),
		MaxRetries:  c.Config.GetInt("llm.max_retries"),
		Timeout:     c.Config.GetDuration("llm.timeout"),
	}, c.Logger)
	return nil
}
