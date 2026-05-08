package agent

import (
	"encoding/json"
	"testing"
)

func TestTextContent(t *testing.T) {
	raw := TextContent("hello world")
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
	raw := TextContent("")
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
	msg := Message{
		Role:    "user",
		Content: TextContent("hi"),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var decoded Message
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
	tc := ToolCall{
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
	resp := &ProviderResponse{
		Content: TextContent("response text"),
	}
	var blocks []map[string]any
	json.Unmarshal(resp.Content, &blocks)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks", len(blocks))
	}
}

func TestProviderResponseWithToolCalls(t *testing.T) {
	resp := &ProviderResponse{
		Content: TextContent("I'll help"),
		ToolCalls: []ToolCall{
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
