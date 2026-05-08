package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"dolphinzZ/internal/config"
)

func TestCDPDefinition(t *testing.T) {
	cdp := NewCDPTool(config.DefaultConfig())
	def := cdp.Definition()

	if def.Name != "cdp" {
		t.Errorf("name = %q, want cdp", def.Name)
	}
	if def.Description == "" {
		t.Error("description should not be empty")
	}
	if def.InputSchema == nil {
		t.Error("input schema should not be nil")
	}
}

func TestCDPExecuteInvalidInput(t *testing.T) {
	cdp := NewCDPTool(config.DefaultConfig())
	result, err := cdp.Execute(context.Background(), json.RawMessage(`not json`))
	if err != nil {
		t.Fatalf("Execute should not return error for bad input, got: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for invalid JSON")
	}
}

func TestCDPExecuteUnknownAction(t *testing.T) {
	cdp := NewCDPTool(config.DefaultConfig())
	result, err := cdp.Execute(context.Background(), json.RawMessage(`{"action":"invalid_action"}`))
	if err != nil {
		t.Fatalf("Execute should not return error for unknown action, got: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for unknown action")
	}
	if result.Content == "" {
		t.Error("expected error message in content")
	}
}

func TestCDPExecuteNavigateNoURL(t *testing.T) {
	cdp := NewCDPTool(config.DefaultConfig())
	result, err := cdp.Execute(context.Background(), json.RawMessage(`{"action":"navigate"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for navigate without URL")
	}
}

func TestCDPExecuteClickNoSelector(t *testing.T) {
	cdp := NewCDPTool(config.DefaultConfig())
	result, err := cdp.Execute(context.Background(), json.RawMessage(`{"action":"click"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for click without selector")
	}
}

func TestCDPExecuteScreenshotNoSelector(t *testing.T) {
	cdp := NewCDPTool(config.DefaultConfig())
	result, err := cdp.Execute(context.Background(), json.RawMessage(`{"action":"screenshot"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	// Screenshot without selector is valid (full page screenshot)
	// But will fail because no browser is running
	if !result.IsError {
		t.Log("screenshot without selector is valid by definition (full page)")
	}
}

func TestCDPExecuteEvaluateNoScript(t *testing.T) {
	cdp := NewCDPTool(config.DefaultConfig())
	result, err := cdp.Execute(context.Background(), json.RawMessage(`{"action":"evaluate"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for evaluate without script")
	}
}

func TestCDPExecuteGetTextNoSelector(t *testing.T) {
	cdp := NewCDPTool(config.DefaultConfig())
	result, err := cdp.Execute(context.Background(), json.RawMessage(`{"action":"get_text"}`))
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !result.IsError {
		t.Error("expected IsError for get_text without selector")
	}
}

func TestCDPParameters(t *testing.T) {
	cdp := NewCDPTool(config.DefaultConfig())
	def := cdp.Definition()

	var schema map[string]any
	if err := json.Unmarshal(def.InputSchema, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema missing properties")
	}

	action, ok := props["action"].(map[string]any)
	if !ok {
		t.Fatal("schema missing action property")
	}
	enum, ok := action["enum"].([]any)
	if !ok {
		t.Fatal("action missing enum")
	}

	actions := make(map[string]bool)
	for _, a := range enum {
		actions[a.(string)] = true
	}
	for _, expected := range []string{"navigate", "click", "screenshot", "evaluate", "get_text"} {
		if !actions[expected] {
			t.Errorf("missing action: %s", expected)
		}
	}
}
