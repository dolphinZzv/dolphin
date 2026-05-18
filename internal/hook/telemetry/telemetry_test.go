package telemetry

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"dolphin/internal/agent/provider"
	"dolphin/internal/config"
	"dolphin/internal/hook"
	"dolphin/internal/mcp"

	"go.opentelemetry.io/otel"
)

func TestInitDisabled(t *testing.T) {
	cfg := config.TelemetryConfig{Enabled: false}
	if err := Init(context.Background(), cfg); err != nil {
		t.Fatalf("Init(disabled) error: %v", err)
	}
}

func TestInitStdout(t *testing.T) {
	cfg := config.TelemetryConfig{
		Enabled:     true,
		ServiceName: "test",
		Exporter:    "stdout",
		SampleRate:  1.0,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Init(ctx, cfg); err != nil {
		t.Fatalf("Init(stdout) error: %v", err)
	}

	tr := Tracer("test")
	_, span := tr.Start(ctx, "test.span")
	span.End()

	if err := Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown error: %v", err)
	}
}

func TestInitInvalidExporter(t *testing.T) {
	cfg := config.TelemetryConfig{
		Enabled:  true,
		Exporter: "invalid",
	}
	if err := Init(context.Background(), cfg); err == nil {
		t.Error("expected error for invalid exporter")
	}
}

func TestTracer(t *testing.T) {
	tr := Tracer("test")
	if tr == nil {
		t.Error("Tracer() returned nil")
	}
}

func TestFullTraceLifecycle(t *testing.T) {
	cfg := config.TelemetryConfig{
		Enabled:     true,
		ServiceName: "test-lifecycle",
		Exporter:    "stdout",
		SampleRate:  1.0,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Init(ctx, cfg); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	defer Shutdown(ctx)

	reg := hook.NewRegistry()
	RegisterHooks(reg)

	sid := "test-session-1"

	// 1. Session start
	if err := reg.Fire(ctx, hook.PointSessionStart, &hook.Context{
		SessionID: sid,
		Turn:      0,
		Values:    make(map[string]any),
	}); err != nil {
		t.Fatalf("session:start error: %v", err)
	}

	// 2. Turn 1: user input
	if err := reg.Fire(ctx, hook.PointUserInput, &hook.Context{
		SessionID: sid,
		Turn:      1,
		UserInput: "Hello, what's the weather?",
		Values:    make(map[string]any),
	}); err != nil {
		t.Fatalf("user:input error: %v", err)
	}

	// 3. LLM call
	req := &provider.ProviderRequest{
		Model:     "test-model",
		MaxTokens: 1024,
		Messages:  []provider.Message{{Role: "user", Content: provider.TextContent("hi")}},
	}
	if err := reg.Fire(ctx, hook.PointBeforeLLM, &hook.Context{
		SessionID: sid,
		Turn:      1,
		Request:   req,
		Values:    make(map[string]any),
	}); err != nil {
		t.Fatalf("llm:before error: %v", err)
	}

	// 4. Response before streaming
	if err := reg.Fire(ctx, hook.PointBeforeResponse, &hook.Context{
		SessionID: sid,
		Turn:      1,
		Values:    make(map[string]any),
	}); err != nil {
		t.Fatalf("response:before error: %v", err)
	}

	// 5. LLM response
	resp := &provider.ProviderResponse{
		Content:    provider.TextContent("It's sunny!"),
		Usage:      &provider.Usage{InputTokens: 10, OutputTokens: 5},
		StopReason: "end_turn",
	}
	if err := reg.Fire(ctx, hook.PointAfterLLM, &hook.Context{
		SessionID: sid,
		Turn:      1,
		Response:  resp,
		Values:    make(map[string]any),
	}); err != nil {
		t.Fatalf("llm:after error: %v", err)
	}

	// 6. Turn 2: next user input (ends turn 1 span, creates turn 2)
	if err := reg.Fire(ctx, hook.PointUserInput, &hook.Context{
		SessionID: sid,
		Turn:      2,
		UserInput: "Thanks!",
		Values:    make(map[string]any),
	}); err != nil {
		t.Fatalf("user:input turn 2 error: %v", err)
	}

	// 7. Second LLM call
	if err := reg.Fire(ctx, hook.PointBeforeLLM, &hook.Context{
		SessionID: sid,
		Turn:      2,
		Request:   req,
		Values:    make(map[string]any),
	}); err != nil {
		t.Fatalf("llm:before turn 2 error: %v", err)
	}

	resp2 := &provider.ProviderResponse{
		Content:    provider.TextContent("You're welcome!"),
		Usage:      &provider.Usage{InputTokens: 15, OutputTokens: 3},
		StopReason: "end_turn",
	}
	if err := reg.Fire(ctx, hook.PointAfterLLM, &hook.Context{
		SessionID: sid,
		Turn:      2,
		Response:  resp2,
		Values:    make(map[string]any),
	}); err != nil {
		t.Fatalf("llm:after turn 2 error: %v", err)
	}

	// 8. Session end
	if err := reg.Fire(ctx, hook.PointSessionEnd, &hook.Context{
		SessionID: sid,
		Turn:      2,
		Values:    make(map[string]any),
	}); err != nil {
		t.Fatalf("session:end error: %v", err)
	}
}

func TestTurnWithToolExecution(t *testing.T) {
	cfg := config.TelemetryConfig{
		Enabled:     true,
		ServiceName: "test-tool",
		Exporter:    "stdout",
		SampleRate:  1.0,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Init(ctx, cfg); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	defer Shutdown(ctx)

	reg := hook.NewRegistry()
	RegisterHooks(reg)

	sid := "test-session-tool"

	// Session start
	reg.Fire(ctx, hook.PointSessionStart, &hook.Context{
		SessionID: sid, Turn: 0, Values: make(map[string]any),
	})

	// Turn 1: user input
	reg.Fire(ctx, hook.PointUserInput, &hook.Context{
		SessionID: sid, Turn: 1, UserInput: "Run shell command",
		Values: make(map[string]any),
	})

	// Sub-turn 1: LLM decides to call tool
	reg.Fire(ctx, hook.PointBeforeLLM, &hook.Context{
		SessionID: sid, Turn: 1,
		Request: &provider.ProviderRequest{Model: "test", MaxTokens: 512},
		Values:  make(map[string]any),
	})

	reg.Fire(ctx, hook.PointBeforeResponse, &hook.Context{
		SessionID: sid, Turn: 1, Values: make(map[string]any),
	})

	reg.Fire(ctx, hook.PointAfterLLM, &hook.Context{
		SessionID: sid, Turn: 1,
		Response: &provider.ProviderResponse{
			StopReason: "tool_use",
			Usage:      &provider.Usage{InputTokens: 20, OutputTokens: 8},
		},
		Values: make(map[string]any),
	})

	// Tool execution
	reg.Fire(ctx, hook.PointBeforeTool, &hook.Context{
		SessionID: sid,
		Turn:      1,
		ToolName:  "shell",
		ToolArgs:  json.RawMessage(`{"command": "date"}`),
		Values:    make(map[string]any),
	})

	result := &mcp.ToolResult{Content: "Sun May 17 18:00:00 CST 2026"}
	reg.Fire(ctx, hook.PointAfterTool, &hook.Context{
		SessionID:  sid,
		Turn:       1,
		ToolName:   "shell",
		ToolResult: result,
		Values:     make(map[string]any),
	})

	// Sub-turn 2: LLM with final response
	reg.Fire(ctx, hook.PointBeforeLLM, &hook.Context{
		SessionID: sid, Turn: 1,
		Request: &provider.ProviderRequest{Model: "test", MaxTokens: 512},
		Values:  make(map[string]any),
	})

	reg.Fire(ctx, hook.PointBeforeResponse, &hook.Context{
		SessionID: sid, Turn: 1, Values: make(map[string]any),
	})

	reg.Fire(ctx, hook.PointAfterLLM, &hook.Context{
		SessionID: sid, Turn: 1,
		Response: &provider.ProviderResponse{
			Content:    provider.TextContent("The date is May 17."),
			StopReason: "end_turn",
			Usage:      &provider.Usage{InputTokens: 30, OutputTokens: 12},
		},
		Values: make(map[string]any),
	})

	// Session end
	reg.Fire(ctx, hook.PointSessionEnd, &hook.Context{
		SessionID: sid, Turn: 1, Values: make(map[string]any),
	})
}

func TestErrorHookMarksAllActiveSpans(t *testing.T) {
	cfg := config.TelemetryConfig{
		Enabled:     true,
		ServiceName: "test-error",
		Exporter:    "stdout",
		SampleRate:  1.0,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Init(ctx, cfg); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	defer Shutdown(ctx)

	reg := hook.NewRegistry()
	RegisterHooks(reg)

	sid := "test-session-err"

	reg.Fire(ctx, hook.PointSessionStart, &hook.Context{
		SessionID: sid, Turn: 0, Values: make(map[string]any),
	})

	reg.Fire(ctx, hook.PointUserInput, &hook.Context{
		SessionID: sid, Turn: 1, UserInput: "test",
		Values: make(map[string]any),
	})

	// Start LLM and tool spans
	reg.Fire(ctx, hook.PointBeforeLLM, &hook.Context{
		SessionID: sid, Turn: 1,
		Request: &provider.ProviderRequest{Model: "test"},
		Values:  make(map[string]any),
	})

	reg.Fire(ctx, hook.PointBeforeTool, &hook.Context{
		SessionID: sid, Turn: 1, ToolName: "cdp",
		Values: make(map[string]any),
	})

	// Fire error — should mark turn, llm, tool spans
	reg.Fire(ctx, hook.PointOnError, &hook.Context{
		SessionID: sid,
		Turn:      1,
		Error:     context.DeadlineExceeded,
		Values:    make(map[string]any),
	})

	// Clean up
	reg.Fire(ctx, hook.PointAfterTool, &hook.Context{
		SessionID: sid, Turn: 1, ToolName: "cdp",
		ToolResult: &mcp.ToolResult{IsError: true, Content: "timeout"},
		Error:      context.DeadlineExceeded,
		Values:     make(map[string]any),
	})

	reg.Fire(ctx, hook.PointAfterLLM, &hook.Context{
		SessionID: sid, Turn: 1,
		Response: &provider.ProviderResponse{},
		Error:    context.DeadlineExceeded,
		Values:   make(map[string]any),
	})

	reg.Fire(ctx, hook.PointSessionEnd, &hook.Context{
		SessionID: sid, Turn: 1, Values: make(map[string]any),
	})
}

func TestGlobalTracerProvider(t *testing.T) {
	// Verify that Init sets the global tracer provider
	cfg := config.TelemetryConfig{
		Enabled:     true,
		ServiceName: "test-global",
		Exporter:    "stdout",
		SampleRate:  1.0,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Init(ctx, cfg); err != nil {
		t.Fatalf("Init error: %v", err)
	}
	defer Shutdown(ctx)

	tp := otel.GetTracerProvider()
	if tp == nil {
		t.Error("global tracer provider is nil after Init")
	}

	tr := tp.Tracer("test")
	if tr == nil {
		t.Error("Tracer() from global provider returned nil")
	}
}
