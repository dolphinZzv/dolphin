//go:build windows

package shell

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"dolphin/internal/config"
)

func TestShellExecuteWithWorkdir(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Shell.AllowedCommands = nil
	tool := New(cfg)
	ctx := WithWorkdir(context.Background(), t.TempDir())
	result, err := tool.Execute(ctx, json.RawMessage(`{"command":"echo %cd%"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
}

func TestShellExecuteRedirectCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Shell.AllowedCommands = nil
	tool := New(cfg)
	// Use echo with NUL redirect (Windows equivalent of /dev/null)
	result, err := tool.Execute(context.Background(), json.RawMessage(`{"command":"echo hello > `+os.DevNull+` && echo ok"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
}
