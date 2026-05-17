package agent

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestAnthropicBuildReqBasic(t *testing.T) {
	p := &AnthropicProvider{
		model:  "deepseek-v4-flash",
		maxTok: 4096,
	}
	req := ProviderRequest{
		System: "you are helpful",
		Messages: []Message{
			{Role: "user", Content: TextContent("hello")},
		},
	}

	ar := p.buildReq(req, false)
	if ar.Model != "deepseek-v4-flash" {
		t.Errorf("model = %q", ar.Model)
	}
	if ar.MaxTokens != 4096 {
		t.Errorf("max_tokens = %d", ar.MaxTokens)
	}
	if ar.System != "you are helpful" {
		t.Errorf("system = %q", ar.System)
	}
	if ar.Stream {
		t.Error("stream should be false")
	}
	if len(ar.Messages) != 1 {
		t.Fatalf("got %d messages, want 1", len(ar.Messages))
	}
	if ar.Messages[0].Role != "user" {
		t.Errorf("role = %q", ar.Messages[0].Role)
	}
}

func TestAnthropicBuildReqWithTools(t *testing.T) {
	p := &AnthropicProvider{model: "test", maxTok: 100}
	req := ProviderRequest{
		System: "test",
		Messages: []Message{
			{Role: "user", Content: TextContent("hi")},
		},
		Tools: []ToolDef{
			{Name: "shell", Description: "run commands", InputSchema: json.RawMessage(`{"type":"object"}`)},
		},
	}

	ar := p.buildReq(req, false)
	if len(ar.Tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(ar.Tools))
	}
	if ar.Tools[0].Name != "shell" {
		t.Errorf("tool name = %q", ar.Tools[0].Name)
	}
}

func TestAnthropicBuildReqToolRole(t *testing.T) {
	p := &AnthropicProvider{model: "test", maxTok: 100}
	toolResult, _ := json.Marshal([]map[string]any{
		{"type": "tool_result", "tool_use_id": "call_1", "content": []map[string]any{
			{"type": "text", "text": "output"},
		}},
	})
	req := ProviderRequest{
		Messages: []Message{
			{Role: "user", Content: TextContent("list files")},
			{Role: "assistant", Content: TextContent("")},
			{Role: "tool", Content: toolResult},
		},
	}

	ar := p.buildReq(req, false)
	// Tool role should be converted to user
	last := ar.Messages[len(ar.Messages)-1]
	if last.Role != "user" {
		t.Errorf("tool message role = %q, want user", last.Role)
	}
}

func TestAnthropicBuildReqAssistantContentPreserved(t *testing.T) {
	p := &AnthropicProvider{model: "test", maxTok: 100}
	assistantContent, _ := json.Marshal([]map[string]any{
		{"type": "text", "text": "I'll help"},
		{"type": "tool_use", "id": "call_1", "name": "shell", "input": map[string]string{"command": "ls"}},
	})
	req := ProviderRequest{
		Messages: []Message{
			{Role: "user", Content: TextContent("hi")},
			{Role: "assistant", Content: assistantContent},
		},
	}

	ar := p.buildReq(req, false)
	if len(ar.Messages) != 2 {
		t.Fatalf("got %d messages, want 2", len(ar.Messages))
	}
	assistant := ar.Messages[1]
	if assistant.Role != "assistant" {
		t.Errorf("role = %q", assistant.Role)
	}
	if len(assistant.Content) == 0 {
		t.Error("content should not be empty")
	}
}

func TestAnthropicBuildReqStream(t *testing.T) {
	p := &AnthropicProvider{model: "test", maxTok: 100}
	req := ProviderRequest{
		Messages: []Message{
			{Role: "user", Content: TextContent("hi")},
		},
	}

	ar := p.buildReq(req, true)
	if !ar.Stream {
		t.Error("stream should be true")
	}
}

func TestAnthropicSetHeaders(t *testing.T) {
	p := &AnthropicProvider{
		apiKey: "sk-test-key",
	}
	req, _ := http.NewRequest("POST", "https://example.com", http.NoBody)
	p.setHeaders(req)

	if req.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", req.Header.Get("Content-Type"))
	}
	if req.Header.Get("x-api-key") != "sk-test-key" {
		t.Errorf("x-api-key = %q", req.Header.Get("x-api-key"))
	}
	if req.Header.Get("anthropic-version") != "2023-06-01" {
		t.Errorf("anthropic-version = %q", req.Header.Get("anthropic-version"))
	}
}

func TestAnthropicBuildReqEmptyMessages(t *testing.T) {
	p := &AnthropicProvider{model: "test", maxTok: 100}
	ar := p.buildReq(ProviderRequest{}, false)
	if len(ar.Messages) != 0 {
		t.Errorf("got %d messages, want 0", len(ar.Messages))
	}
}

func TestAnthropicBuildReqMultipleToolResultsMerged(t *testing.T) {
	// Simulates an assistant making multiple tool calls in one turn.
	// All tool results must be merged into a single user message — consecutive
	// user messages violate the Anthropic API spec and cause a 400 error.
	p := &AnthropicProvider{model: "test", maxTok: 100}
	toolResult1, _ := json.Marshal([]map[string]any{
		{"type": "tool_result", "tool_use_id": "call_1", "content": []map[string]any{
			{"type": "text", "text": "file1.txt\nfile2.txt"},
		}},
	})
	toolResult2, _ := json.Marshal([]map[string]any{
		{"type": "tool_result", "tool_use_id": "call_2", "content": []map[string]any{
			{"type": "text", "text": "hello world"},
		}},
	})
	req := ProviderRequest{
		Messages: []Message{
			{Role: "user", Content: TextContent("list files and read hello.txt")},
			{Role: "assistant", Content: TextContent("")},
			{Role: "tool", Content: toolResult1},
			{Role: "tool", Content: toolResult2},
			{Role: "user", Content: TextContent("thanks")},
		},
	}

	ar := p.buildReq(req, false)
	if len(ar.Messages) != 4 {
		t.Fatalf("got %d messages, want 4 (user, assistant, user[merged tools], user)", len(ar.Messages))
	}
	// The merged tool message (index 2) should be a single user message containing both tool_results
	merged := ar.Messages[2]
	if merged.Role != "user" {
		t.Errorf("merged tool message role = %q, want user", merged.Role)
	}
	var blocks []map[string]any
	if err := json.Unmarshal(merged.Content, &blocks); err != nil {
		t.Fatalf("failed to unmarshal merged content: %v", err)
	}
	if len(blocks) != 2 {
		t.Fatalf("merged content has %d blocks, want 2 (both tool_results)", len(blocks))
	}
	if blocks[0]["type"] != "tool_result" || blocks[0]["tool_use_id"] != "call_1" {
		t.Errorf("first block = %v, want tool_result call_1", blocks[0])
	}
	if blocks[1]["type"] != "tool_result" || blocks[1]["tool_use_id"] != "call_2" {
		t.Errorf("second block = %v, want tool_result call_2", blocks[1])
	}
}

func TestAnthropicBuildReqSingleToolResult(t *testing.T) {
	// A single tool result should still be wrapped in a user message correctly.
	p := &AnthropicProvider{model: "test", maxTok: 100}
	toolResult, _ := json.Marshal([]map[string]any{
		{"type": "tool_result", "tool_use_id": "call_1", "content": []map[string]any{
			{"type": "text", "text": "output"},
		}},
	})
	req := ProviderRequest{
		Messages: []Message{
			{Role: "user", Content: TextContent("list files")},
			{Role: "assistant", Content: TextContent("")},
			{Role: "tool", Content: toolResult},
		},
	}

	ar := p.buildReq(req, false)
	if len(ar.Messages) != 3 {
		t.Fatalf("got %d messages, want 3", len(ar.Messages))
	}
	last := ar.Messages[2]
	if last.Role != "user" {
		t.Errorf("role = %q, want user", last.Role)
	}
	var blocks []map[string]any
	json.Unmarshal(last.Content, &blocks)
	if len(blocks) != 1 {
		t.Errorf("got %d blocks, want 1", len(blocks))
	}
}

func TestAnthropicBuildReqMultipleTools(t *testing.T) {
	p := &AnthropicProvider{model: "test", maxTok: 100}
	req := ProviderRequest{
		Tools: []ToolDef{
			{Name: "shell", Description: "run commands", InputSchema: json.RawMessage(`{"type":"object"}`)},
			{Name: "read", Description: "read files", InputSchema: json.RawMessage(`{"type":"object"}`)},
		},
	}
	ar := p.buildReq(req, false)
	if len(ar.Tools) != 2 {
		t.Fatalf("got %d tools, want 2", len(ar.Tools))
	}
}
