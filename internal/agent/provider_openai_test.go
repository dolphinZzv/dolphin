package agent

import (
	"encoding/json"
	"testing"
)

func TestExtractToolCallID(t *testing.T) {
	content := json.RawMessage(`[{"type":"tool_result","tool_use_id":"call_abc","content":[]}]`)
	id := extractToolCallID(content)
	if id != "call_abc" {
		t.Errorf("got %q, want call_abc", id)
	}
}

func TestExtractToolCallIDNoMatch(t *testing.T) {
	content := json.RawMessage(`[{"type":"text","text":"hello"}]`)
	id := extractToolCallID(content)
	if id != "" {
		t.Errorf("got %q, want empty", id)
	}
}

func TestExtractToolCallIDInvalidJSON(t *testing.T) {
	content := json.RawMessage("not json")
	id := extractToolCallID(content)
	if id != "" {
		t.Errorf("got %q, want empty", id)
	}
}

func TestExtractToolResultStringContent(t *testing.T) {
	content := json.RawMessage(`[{"type":"tool_result","content":"direct output"}]`)
	result := extractToolResult(content)
	if result != "direct output" {
		t.Errorf("got %q, want direct output", result)
	}
}

func TestExtractToolResultArrayContent(t *testing.T) {
	content := json.RawMessage(`[{"type":"tool_result","content":[{"type":"text","text":"array output"}]}]`)
	result := extractToolResult(content)
	if result != "array output" {
		t.Errorf("got %q, want array output", result)
	}
}

func TestExtractToolResultNoToolResult(t *testing.T) {
	content := json.RawMessage(`[{"type":"text","text":"hello"}]`)
	result := extractToolResult(content)
	if result != string(content) {
		t.Errorf("got %q, want raw content fallback", result)
	}
}

func TestExtractToolResultInvalidJSON(t *testing.T) {
	content := json.RawMessage("raw string")
	result := extractToolResult(content)
	if result != "raw string" {
		t.Errorf("got %q, want raw string", result)
	}
}

func TestBuildTools(t *testing.T) {
	p := &OpenAIProvider{}
	defs := []ToolDef{
		{Name: "test", Description: "a test", InputSchema: json.RawMessage(`{"type":"object"}`)},
	}
	tools := p.buildTools(defs)
	if len(tools) != 1 {
		t.Fatalf("got %d tools, want 1", len(tools))
	}
	if tools[0].Function.Name != "test" {
		t.Errorf("name = %q", tools[0].Function.Name)
	}
	if tools[0].Function.Description != "a test" {
		t.Errorf("description = %q", tools[0].Function.Description)
	}
}

func TestBuildToolsEmpty(t *testing.T) {
	p := &OpenAIProvider{}
	tools := p.buildTools(nil)
	if len(tools) != 0 {
		t.Errorf("got %d tools, want 0", len(tools))
	}
}
