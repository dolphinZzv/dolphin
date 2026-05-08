package agent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"dolphinzZ/internal/config"
	"dolphinzZ/internal/mcp"
	"dolphinzZ/internal/session"
)

func TestRunFullSessionWelcomeAndExit(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Session.Dir = t.TempDir()
	cfg.Session.MaxLoop = 50
	cfg.LLM.MaxContextTokens = 100000

	sessMgr := session.NewManager(cfg.Session.Dir)
	sessMgr.EnsureDir()

	toolReg := mcp.NewRegistry(cfg)
	toolReg.Register(&mockTool{name: "test_tool"})

	prov := &mockProvider{
		responses: []*ProviderResponse{
			{Content: TextContent("hello from LLM"), Usage: &Usage{InputTokens: 5, OutputTokens: 10}},
		},
	}

	agt := &Agent{
		cfg:        cfg,
		sessMgr:    sessMgr,
		toolReg:    toolReg,
		provider:   prov,
		ctxBuilder: NewContextBuilder(),
	}

	io := &mockIO{lines: []string{"say hi", "/exit"}}

	agt.Run(context.Background(), io)

	output := io.writes.String()
	if !strings.Contains(output, "DolphinzZ Agent ready") {
		t.Error("expected welcome message")
	}
	if !strings.Contains(output, "Loaded MCP tools:") {
		t.Error("expected tools list in welcome")
	}
	if !strings.Contains(output, "test_tool") {
		t.Error("expected test_tool in tools list")
	}
	if !strings.Contains(output, "hello from LLM") {
		t.Errorf("expected LLM response in output, got: %s", output)
	}
}

func TestRunHelpCommand(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Session.Dir = t.TempDir()
	cfg.LLM.MaxContextTokens = 100000

	sessMgr := session.NewManager(cfg.Session.Dir)
	sessMgr.EnsureDir()

	toolReg := mcp.NewRegistry(cfg)
	toolReg.Register(&mockTool{name: "test_tool"})

	prov := &mockProvider{}

	agt := &Agent{
		cfg:        cfg,
		sessMgr:    sessMgr,
		toolReg:    toolReg,
		provider:   prov,
		ctxBuilder: NewContextBuilder(),
	}

	io := &mockIO{lines: []string{"/help", "/exit"}}

	agt.Run(context.Background(), io)

	output := io.writes.String()
	if !strings.Contains(output, "Commands:") {
		t.Error("expected help text")
	}
	if !strings.Contains(output, "/exit") {
		t.Error("expected /exit in help")
	}
	if !strings.Contains(output, "Loaded MCP tools:") {
		t.Error("expected tools in help")
	}
}

func TestRunMaxLoopGeneratesSummary(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Session.Dir = t.TempDir()
	cfg.Session.MaxLoop = 1
	cfg.Session.Summary = true
	cfg.LLM.MaxContextTokens = 100000

	sessMgr := session.NewManager(cfg.Session.Dir)
	sessMgr.EnsureDir()

	toolReg := mcp.NewRegistry(cfg)

	prov := &mockProvider{
		responses: []*ProviderResponse{
			{Content: TextContent("response 1"), Usage: &Usage{InputTokens: 5, OutputTokens: 10}},
			{Content: TextContent("response 2"), Usage: &Usage{InputTokens: 5, OutputTokens: 10}},
		},
	}

	agt := &Agent{
		cfg:        cfg,
		sessMgr:    sessMgr,
		toolReg:    toolReg,
		provider:   prov,
		ctxBuilder: NewContextBuilder(),
	}

	io := &mockIO{lines: []string{"first", "second", "/exit"}}

	agt.Run(context.Background(), io)

	output := io.writes.String()
	if !strings.Contains(output, "checkpoint") {
		t.Error("expected checkpoint message at max loop, got:", output)
	}
	if !strings.Contains(output, "response 2") {
		t.Error("expected second response after checkpoint, got:", output)
	}
}

func TestRunEmptyInputSkipped(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Session.Dir = t.TempDir()
	cfg.Session.MaxLoop = 5
	cfg.LLM.MaxContextTokens = 100000

	sessMgr := session.NewManager(cfg.Session.Dir)
	sessMgr.EnsureDir()

	toolReg := mcp.NewRegistry(cfg)

	prov := &mockProvider{
		responses: []*ProviderResponse{
			{Content: TextContent("hi"), Usage: &Usage{InputTokens: 5, OutputTokens: 10}},
		},
	}

	agt := &Agent{
		cfg:        cfg,
		sessMgr:    sessMgr,
		toolReg:    toolReg,
		provider:   prov,
		ctxBuilder: NewContextBuilder(),
	}

	// empty line should be skipped, then "hello" processed
	io := &mockIO{lines: []string{"", "hello", "/exit"}}

	agt.Run(context.Background(), io)

	output := io.writes.String()
	if !strings.Contains(output, "hi") {
		t.Error("expected response after skipping empty input, got:", output)
	}
}

func TestRunToolCallAndStreaming(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Session.Dir = t.TempDir()
	cfg.Session.MaxLoop = 10
	cfg.LLM.MaxContextTokens = 100000

	sessMgr := session.NewManager(cfg.Session.Dir)
	sessMgr.EnsureDir()

	toolReg := mcp.NewRegistry(cfg)
	toolReg.Register(&mockTool{name: "test_tool"})

	prov := &mockProvider{
		responses: []*ProviderResponse{
			{
				Content:    jsonContent(`[{"type":"text","text":"calling tool"},{"type":"tool_use","id":"tu1","name":"test_tool","input":{}}]`),
				ToolCalls:  []ToolCall{{ID: "tu1", Name: "test_tool", Arguments: json.RawMessage(`{}`)}},
				Usage:      &Usage{InputTokens: 10, OutputTokens: 5},
				StopReason: "tool_use",
			},
			{
				Content:    TextContent("tool done"),
				Usage:      &Usage{InputTokens: 20, OutputTokens: 10},
				StopReason: "end_turn",
			},
		},
	}

	agt := &Agent{
		cfg:        cfg,
		sessMgr:    sessMgr,
		toolReg:    toolReg,
		provider:   prov,
		ctxBuilder: NewContextBuilder(),
	}

	io := &mockIO{lines: []string{"do it", "/exit"}}

	agt.Run(context.Background(), io)

	output := io.writes.String()
	if !strings.Contains(output, "calling tool") {
		t.Error("expected reasoning text before tool call, got:", output)
	}
	if !strings.Contains(output, "tool done") {
		t.Error("expected final response after tool, got:", output)
	}
}

// jsonContent is a helper to create json.RawMessage from a JSON string literal.
func jsonContent(s string) json.RawMessage {
	return json.RawMessage(s)
}
