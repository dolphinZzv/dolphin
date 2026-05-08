package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"dolphinzZ/internal/config"
)

func TestShellToolDefinition(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewShellTool(cfg)
	def := tool.Definition()
	if def.Name != "shell" {
		t.Errorf("name = %q, want shell", def.Name)
	}
	if def.Description == "" {
		t.Error("description should not be empty")
	}
	if def.InputSchema == nil {
		t.Error("input_schema should not be nil")
	}
}

func TestShellExecuteEcho(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Shell.Enabled = true
	cfg.MCP.Shell.TimeoutSeconds = 10

	tool := NewShellTool(cfg)
	input, _ := json.Marshal(map[string]string{"command": "echo hello"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if result.Content == "" {
		t.Error("result should not be empty")
	}
}

func TestShellExecuteInvalidInput(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewShellTool(cfg)

	result, err := tool.Execute(context.Background(), json.RawMessage(`not json`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Error("expected is_error for invalid input")
	}
}

func TestShellCommandNotAllowed(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Shell.AllowedCommands = []string{"echo"}

	tool := NewShellTool(cfg)
	input, _ := json.Marshal(map[string]string{"command": "rm -rf /"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Error("expected is_error for disallowed command")
	}
}

func TestShellCommandAllowed(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Shell.AllowedCommands = []string{"echo"}

	tool := NewShellTool(cfg)
	input, _ := json.Marshal(map[string]string{"command": "echo allowed"})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
}

func TestShellTimeout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.MCP.Shell.TimeoutSeconds = 1

	tool := NewShellTool(cfg)
	input, _ := json.Marshal(map[string]string{"command": "sleep 10"})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := tool.Execute(ctx, input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Error("expected is_error for timeout")
	}
}

func TestShellCustomTimeout(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewShellTool(cfg)
	input, _ := json.Marshal(map[string]any{
		"command": "echo hello",
		"timeout": 5,
	})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
}

func TestShellWorkingDirectory(t *testing.T) {
	cfg := config.DefaultConfig()
	tool := NewShellTool(cfg)

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "testfile.txt"), []byte("content"), 0644)

	input, _ := json.Marshal(map[string]string{"command": "ls " + tmpDir})
	result, err := tool.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
}
