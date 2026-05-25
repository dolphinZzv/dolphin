package transport

import (
	"io"
	"testing"
	"time"
)

// testIO implements UserIO for testing with configurable read/write.
type testIO struct {
	readCh  chan string
	writeCh chan string
	name    string
}

func newTestIO() *testIO {
	return &testIO{
		readCh:  make(chan string, 100),
		writeCh: make(chan string, 100),
		name:    "test",
	}
}

func (t *testIO) ReadLine() (string, error) {
	msg, ok := <-t.readCh
	if !ok {
		return "", io.EOF
	}
	return msg, nil
}

func (t *testIO) WriteLine(s string) error {
	t.writeCh <- s
	return nil
}

func (t *testIO) WriteString(s string) error {
	return t.WriteLine(s)
}

func (t *testIO) Flush() error               { return nil }
func (t *testIO) Capabilities() Capabilities { return Capabilities{Streaming: true} }
func (t *testIO) Context() string            { return "" }
func (t *testIO) Name() string               { return t.name }

func TestMultiIO_NonAtLine(t *testing.T) {
	inner := newTestIO()
	m := NewMultiIO(inner)

	inner.readCh <- "hello"
	line, err := m.ReadLine()
	if err != nil {
		t.Fatal(err)
	}
	if line != "hello" {
		t.Fatalf("expected 'hello', got %q", line)
	}
}

func TestMultiIO_AtLineNoAgent(t *testing.T) {
	inner := newTestIO()
	m := NewMultiIO(inner)

	// @mention without a matching agent should be returned as-is
	inner.readCh <- "@helper do something"
	line, err := m.ReadLine()
	if err != nil {
		t.Fatal(err)
	}
	if line != "@helper do something" {
		t.Fatalf("expected '@helper do something', got %q", line)
	}
}

func TestMultiIO_AtLineToAgent(t *testing.T) {
	inner := newTestIO()
	m := NewMultiIO(inner)

	subIO := m.RegisterAgent("helper")
	msgCh := make(chan string, 1)
	go func() {
		msg, err := subIO.ReadLine()
		if err != nil {
			t.Error(err)
		}
		msgCh <- msg
	}()

	// Feed @-message
	var called bool
	m.OnUserMessage = func(agentName, message string) {
		called = true
		if agentName != "helper" {
			t.Fatalf("expected agent 'helper', got %q", agentName)
		}
		if message != "run date" {
			t.Fatalf("expected message 'run date', got %q", message)
		}
	}

	inner.readCh <- "@helper run date"
	inner.readCh <- "next line"

	// First ReadLine returns "" after @-dispatch (triggers coordinator result check)
	line, err := m.ReadLine()
	if err != nil {
		t.Fatal(err)
	}
	if line != "" {
		t.Fatalf("expected empty after @-dispatch, got %q", line)
	}

	// Second ReadLine returns the next actual line
	line, err = m.ReadLine()
	if err != nil {
		t.Fatal(err)
	}
	if line != "next line" {
		t.Fatalf("expected 'next line', got %q", line)
	}

	if !called {
		t.Fatal("OnUserMessage was not called")
	}

	// SubAgentIO should have received the message
	select {
	case msg := <-msgCh:
		if msg != "run date" {
			t.Fatalf("expected 'run date', got %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for subagent message")
	}
}

func TestMultiIO_AtLineShort(t *testing.T) {
	inner := newTestIO()
	m := NewMultiIO(inner)

	// Just "@" or "@name" with no message should be returned as-is
	inner.readCh <- "@"
	inner.readCh <- "@helper"
	inner.readCh <- "@ "

	line1, _ := m.ReadLine()
	if line1 != "@" {
		t.Fatalf("expected '@', got %q", line1)
	}
	line2, _ := m.ReadLine()
	if line2 != "@helper" {
		t.Fatalf("expected '@helper', got %q", line2)
	}
	line3, _ := m.ReadLine()
	if line3 != "@ " {
		t.Fatalf("expected '@ ', got %q", line3)
	}
}

func TestMultiIO_WriteToUser(t *testing.T) {
	inner := newTestIO()
	m := NewMultiIO(inner)

	if err := m.WriteToUser("helper", "task done"); err != nil {
		t.Fatal(err)
	}

	select {
	case line := <-inner.writeCh:
		expected := "[helper] task done"
		if line != expected {
			t.Fatalf("expected %q, got %q", expected, line)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for write")
	}
}

func TestMultiIO_RegisterUnregister(t *testing.T) {
	inner := newTestIO()
	m := NewMultiIO(inner)

	if names := m.AgentNames(); len(names) != 0 {
		t.Fatalf("expected 0 agents, got %d", len(names))
	}

	subIO := m.RegisterAgent("helper")
	if subIO == nil {
		t.Fatal("expected SubAgentIO, got nil")
	}

	if names := m.AgentNames(); len(names) != 1 || names[0] != "helper" {
		t.Fatalf("expected ['helper'], got %v", names)
	}

	m.UnregisterAgent("helper")
	if names := m.AgentNames(); len(names) != 0 {
		t.Fatalf("expected 0 agents, got %d", len(names))
	}
}

func TestSubAgentIO_WriteForwardsToRoot(t *testing.T) {
	inner := newTestIO()
	m := NewMultiIO(inner)
	subIO := m.RegisterAgent("helper")

	if err := subIO.WriteLine("hello"); err != nil {
		t.Fatal(err)
	}
	line := <-inner.writeCh
	if line != "[helper] hello" {
		t.Fatalf("expected '[helper] hello', got %q", line)
	}

	if err := subIO.WriteString("world"); err != nil {
		t.Fatal(err)
	}
	line = <-inner.writeCh
	if line != "[helper] world" {
		t.Fatalf("expected '[helper] world', got %q", line)
	}

	if err := subIO.Flush(); err != nil {
		t.Fatal(err)
	}
}

func TestSubAgentIO_Capabilities(t *testing.T) {
	inner := newTestIO()
	m := NewMultiIO(inner)
	subIO := m.RegisterAgent("helper")

	caps := subIO.Capabilities()
	if !caps.Streaming {
		t.Fatal("expected streaming capability from root")
	}
}

func TestSubAgentIO_Name(t *testing.T) {
	inner := newTestIO()
	m := NewMultiIO(inner)
	subIO := m.RegisterAgent("helper")

	if name := subIO.Name(); name != "[helper]" {
		t.Fatalf("expected '[helper]', got %q", name)
	}
}

func TestParseAgentMessage(t *testing.T) {
	tests := []struct {
		input     string
		name, msg string
		ok        bool
	}{
		{"@helper run date", "helper", "run date", true},
		{"@a b", "a", "b", true},
		{"@helper", "", "", false},
		{"@", "", "", false},
		{"@ ", "", "", false},
		{"hello", "", "", false},
		{"@@", "", "", false},
	}
	for _, tt := range tests {
		name, msg, ok := parseAgentMessage(tt.input)
		if ok != tt.ok || name != tt.name || msg != tt.msg {
			t.Errorf("parseAgentMessage(%q) = (%q, %q, %v), want (%q, %q, %v)",
				tt.input, name, msg, ok, tt.name, tt.msg, tt.ok)
		}
	}
}

func TestMultiIO_AtLineFullInbox(t *testing.T) {
	inner := newTestIO()
	m := NewMultiIO(inner)
	m.RegisterAgent("helper")

	// Fill the inbox (cap 64)
	for i := 0; i < 64; i++ {
		m.agents["helper"].inputCh <- "msg"
	}

	// This should not block (drops the message)
	inner.readCh <- "@helper overflow"
	inner.readCh <- "next"

	line, err := m.ReadLine()
	if err != nil {
		t.Fatal(err)
	}
	if line != "" {
		t.Fatalf("expected empty after @-dispatch, got %q", line)
	}

	line, err = m.ReadLine()
	if err != nil {
		t.Fatal(err)
	}
	if line != "next" {
		t.Fatalf("expected 'next', got %q", line)
	}
}

func TestMultiIO_ErrorPropagation(t *testing.T) {
	inner := newTestIO()
	m := NewMultiIO(inner)
	close(inner.readCh)

	_, err := m.ReadLine()
	if err == nil {
		t.Fatal("expected EOF error")
	}
}
