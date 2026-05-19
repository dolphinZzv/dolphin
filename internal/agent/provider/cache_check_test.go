package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dolphin/internal/config"

	"gopkg.in/yaml.v3"
)

func findConfigPath(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for i := 0; i < 10; i++ {
		candidate := filepath.Join(dir, ".dolphin", "config.yaml")
		if _, err := os.Stat(candidate); err == nil {
			data, err := os.ReadFile(candidate)
			if err == nil && bytes.Contains(data, []byte("api_key:")) {
				return candidate
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func TestCacheTokensWithDeepSeekPro(t *testing.T) {
	cfgPath := findConfigPath(t)
	if cfgPath == "" {
		t.Skip(".dolphin/config.yaml not found (walked up from CWD)")
	}

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Skipf("read config: %v", err)
	}

	var parsed struct {
		LLM struct {
			APIKey    string `yaml:"api_key"`
			BaseURL   string `yaml:"base_url"`
			Providers []struct {
				APIKey  string `yaml:"api_key"`
				BaseURL string `yaml:"base_url"`
			} `yaml:"providers"`
		} `yaml:"llm"`
	}
	if err := yaml.Unmarshal(raw, &parsed); err != nil {
		t.Skipf("parse config: %v", err)
	}

	apiKey := parsed.LLM.APIKey
	baseURL := parsed.LLM.BaseURL
	if apiKey == "" && len(parsed.LLM.Providers) > 0 && parsed.LLM.Providers[0].APIKey != "" {
		apiKey = parsed.LLM.Providers[0].APIKey
		baseURL = parsed.LLM.Providers[0].BaseURL
	}
	if apiKey == "" {
		apiKey = os.Getenv("DZ_LLM_API_KEY")
	}
	if apiKey == "" {
		t.Skip("no API key available")
	}

	t.Logf("Testing model=deepseek-v4-pro baseURL=%s", baseURL)

	prov := NewOpenAIProvider(&config.ProviderConfig{
		Name:      "cache-test",
		BaseURL:   baseURL,
		APIKey:    apiKey,
		Model:     "deepseek-v4-pro",
		MaxTokens: 1024,
	})

	ctx := context.Background()
	prompt := "What is the capital of France? Answer in one word."

	for i := 0; i < 3; i++ {
		t.Logf("=== Request %d ===", i+1)
		resp, err := prov.Complete(ctx, ProviderRequest{
			System: "You are a helpful assistant.",
			Messages: []Message{
				{Role: "user", Content: json.RawMessage(`"` + prompt + `"`)},
			},
		})
		if err != nil {
			if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
				t.Skipf("auth error: %v", err)
			}
			if strings.Contains(err.Error(), "404") {
				t.Skipf("model not found: %v", err)
			}
			t.Fatalf("Request %d error: %v", i+1, err)
		}

		t.Logf("Content: %s", string(resp.Content))
		if resp.Usage != nil {
			t.Logf("Input: %d, Output: %d, Cached: %d, Miss: %d",
				resp.Usage.InputTokens, resp.Usage.OutputTokens,
				resp.Usage.CachedInputTokens, resp.Usage.MissedInputTokens)
		}
		time.Sleep(1 * time.Second)
	}
}
