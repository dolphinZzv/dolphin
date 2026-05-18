package agent

import (
	"encoding/json"
	"testing"

	"dolphin/internal/agent/provider"
)

func TestTextContent(t *testing.T) {
	raw := provider.TextContent("hello world")
	var blocks []map[string]any
	if err := json.Unmarshal(raw, &blocks); err != nil {
		t.Fatalf("TextContent produced invalid JSON: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}
	if blocks[0]["type"] != "text" {
		t.Errorf("block type = %v, want text", blocks[0]["type"])
	}
	text, ok := blocks[0]["text"].(string)
	if !ok || text != "hello world" {
		t.Errorf("block text = %v, want hello world", blocks[0]["text"])
	}
}

func TestTextContentEmpty(t *testing.T) {
	raw := provider.TextContent("")
	var blocks []map[string]any
	json.Unmarshal(raw, &blocks)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1", len(blocks))
	}
	if blocks[0]["text"] != "" {
		t.Errorf("empty text content = %v", blocks[0]["text"])
	}
}

func TestMessageJSON(t *testing.T) {
	msg := provider.Message{
		Role:    "user",
		Content: provider.TextContent("hi"),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var decoded provider.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.Role != "user" {
		t.Errorf("role = %q, want user", decoded.Role)
	}
	if len(decoded.Content) == 0 {
		t.Error("content should not be empty")
	}
}

func TestToolCallJSON(t *testing.T) {
	tc := provider.ToolCall{
		ID:        "call_123",
		Name:      "shell",
		Arguments: json.RawMessage(`{"command":"ls"}`),
	}
	if tc.ID != "call_123" {
		t.Errorf("ID = %q", tc.ID)
	}
	if tc.Name != "shell" {
		t.Errorf("Name = %q", tc.Name)
	}
}

func TestProviderResponseText(t *testing.T) {
	resp := &provider.ProviderResponse{
		Content: provider.TextContent("response text"),
	}
	var blocks []map[string]any
	json.Unmarshal(resp.Content, &blocks)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks", len(blocks))
	}
}

func TestProviderResponseWithToolCalls(t *testing.T) {
	resp := &provider.ProviderResponse{
		Content: provider.TextContent("I'll help"),
		ToolCalls: []provider.ToolCall{
			{ID: "tc1", Name: "shell", Arguments: json.RawMessage(`{"command":"ls"}`)},
		},
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("got %d tool calls, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "shell" {
		t.Errorf("tool name = %q", resp.ToolCalls[0].Name)
	}
}
