package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"dolphinzZ/internal/config"
)

func TestRegistryRegisterAndGet(t *testing.T) {
	r := NewRegistry(config.DefaultConfig())
	r.Register(&testTool{name: "echo"})

	tool, ok := r.Get("echo")
	if !ok {
		t.Fatal("expected to find tool 'echo'")
	}
	if tool.Definition().Name != "echo" {
		t.Errorf("tool name = %q", tool.Definition().Name)
	}
}

func TestRegistryGetNotFound(t *testing.T) {
	r := NewRegistry(config.DefaultConfig())
	_, ok := r.Get("nonexistent")
	if ok {
		t.Error("expected false for nonexistent tool")
	}
}

func TestRegistryList(t *testing.T) {
	r := NewRegistry(config.DefaultConfig())
	r.Register(&testTool{name: "a"})
	r.Register(&testTool{name: "b"})

	defs := r.List()
	if len(defs) != 2 {
		t.Fatalf("got %d definitions, want 2", len(defs))
	}
}

func TestRegistryExecute(t *testing.T) {
	r := NewRegistry(config.DefaultConfig())
	r.Register(&testTool{name: "echo"})

	result, err := r.Execute(context.Background(), "echo", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result.Content != "ok" {
		t.Errorf("result = %q, want ok", result.Content)
	}
}

func TestRegistryExecuteNotFound(t *testing.T) {
	r := NewRegistry(config.DefaultConfig())
	_, err := r.Execute(context.Background(), "missing", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for missing tool")
	}
}

// testTool implements Tool for testing.
type testTool struct {
	name string
}

func (t *testTool) Definition() ToolDefinition {
	return ToolDefinition{
		Name:        t.name,
		Description: "test tool",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}
}

func (t *testTool) Execute(_ context.Context, _ json.RawMessage) (*ToolResult, error) {
	return &ToolResult{Content: "ok"}, nil
}
