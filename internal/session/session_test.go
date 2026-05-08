package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestManagerCreateAndGet(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	if err := mgr.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir error: %v", err)
	}

	sess, err := mgr.NewSession(10)
	if err != nil {
		t.Fatalf("NewSession error: %v", err)
	}
	if sess.ID == "" {
		t.Error("session ID should not be empty")
	}
	if sess.MaxLoop != 10 {
		t.Errorf("MaxLoop = %d, want 10", sess.MaxLoop)
	}

	got := mgr.Get(sess.ID)
	if got == nil {
		t.Fatal("Get returned nil for existing session")
	}
	if got.ID != sess.ID {
		t.Errorf("Get returned session with ID %q, want %q", got.ID, sess.ID)
	}
}

func TestManagerRemove(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	mgr.EnsureDir()

	sess, _ := mgr.NewSession(10)
	mgr.Remove(sess.ID)

	if got := mgr.Get(sess.ID); got != nil {
		t.Error("Get returned session after Remove")
	}
}

func TestManagerCleanup(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	mgr.EnsureDir()

	mgr.NewSession(10)
	mgr.NewSession(10)
	mgr.Cleanup()

	if len(mgr.sessions) != 0 {
		t.Errorf("expected 0 sessions after cleanup, got %d", len(mgr.sessions))
	}
}

func TestSessionLogMessage(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	mgr.EnsureDir()

	sess, _ := mgr.NewSession(10)
	sess.Turn = 1

	content := json.RawMessage(`{"text":"hello"}`)
	if err := sess.LogMessage("user", content); err != nil {
		t.Fatalf("LogMessage error: %v", err)
	}
	sess.Close()

	// Read the log file
	data, err := os.ReadFile(filepath.Join(dir, string(sess.ID)+".jsonl"))
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	var evt SessionEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if evt.Type != EventMessage {
		t.Errorf("event type = %q, want message", evt.Type)
	}
	if evt.Role != "user" {
		t.Errorf("role = %q, want user", evt.Role)
	}
	if evt.SessionID != sess.ID {
		t.Errorf("session_id mismatch")
	}
}

func TestSessionLogToolCall(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	mgr.EnsureDir()

	sess, _ := mgr.NewSession(10)
	sess.Turn = 1

	input := json.RawMessage(`{"command":"ls"}`)
	if err := sess.LogToolCall("shell", input); err != nil {
		t.Fatalf("LogToolCall error: %v", err)
	}
	sess.Close()

	data, _ := os.ReadFile(filepath.Join(dir, string(sess.ID)+".jsonl"))
	var evt SessionEvent
	json.Unmarshal(data, &evt)
	if evt.Type != EventToolCall {
		t.Errorf("type = %q, want tool_call", evt.Type)
	}
	if evt.ToolName != "shell" {
		t.Errorf("tool_name = %q", evt.ToolName)
	}
}

func TestSessionLogToolResult(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	mgr.EnsureDir()

	sess, _ := mgr.NewSession(10)
	sess.Turn = 1

	result := json.RawMessage(`{"output":"ok"}`)
	if err := sess.LogToolResult("shell", result, false); err != nil {
		t.Fatalf("LogToolResult error: %v", err)
	}
	sess.Close()

	data, _ := os.ReadFile(filepath.Join(dir, string(sess.ID)+".jsonl"))
	var evt SessionEvent
	json.Unmarshal(data, &evt)
	if evt.Type != EventToolResult {
		t.Errorf("type = %q, want tool_result", evt.Type)
	}
	if evt.IsError {
		t.Error("is_error should be false")
	}
}

func TestSessionLogSystem(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	mgr.EnsureDir()

	sess, _ := mgr.NewSession(10)
	if err := sess.LogSystem("test event"); err != nil {
		t.Fatalf("LogSystem error: %v", err)
	}
	sess.Close()

	data, _ := os.ReadFile(filepath.Join(dir, string(sess.ID)+".jsonl"))
	var evt SessionEvent
	json.Unmarshal(data, &evt)
	if evt.Type != EventSystem {
		t.Errorf("type = %q, want system", evt.Type)
	}
}

func TestManagerEnsureDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	mgr := NewManager(dir)
	if err := mgr.EnsureDir(); err != nil {
		t.Fatalf("EnsureDir error: %v", err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("directory was not created")
	}
}
