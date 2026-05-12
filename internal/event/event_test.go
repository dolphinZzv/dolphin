package event

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestEmitAndReceive(t *testing.T) {
	bus := NewEventBus(16)
	var received []Event
	var mu sync.Mutex

	bus.On(TypeToolCalled, func(ctx context.Context, evt Event) {
		mu.Lock()
		received = append(received, evt)
		mu.Unlock()
	})

	bus.Emit(context.Background(), Event{Type: TypeToolCalled, SessionID: "s1", Turn: 1})
	bus.Emit(context.Background(), Event{Type: TypeToolCompleted, SessionID: "s1", Turn: 1})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 event, got %d", len(received))
	}
	if received[0].Type != TypeToolCalled {
		t.Errorf("expected tool:called, got %s", received[0].Type)
	}
}

func TestWildcardSubscription(t *testing.T) {
	bus := NewEventBus(16)
	var mu sync.Mutex
	count := 0

	bus.On("*", func(ctx context.Context, evt Event) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	bus.Emit(context.Background(), Event{Type: TypeSessionCreated, SessionID: "s1"})
	bus.Emit(context.Background(), Event{Type: TypeToolCalled, SessionID: "s1"})
	bus.Emit(context.Background(), Event{Type: TypeError, SessionID: "s1"})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count != 3 {
		t.Errorf("wildcard should receive 3 events, got %d", count)
	}
}

func TestLogWriter(t *testing.T) {
	var buf bytes.Buffer
	bus := NewEventBus(16)
	bus.SetLogWriter(&buf)

	bus.Emit(context.Background(), Event{Type: TypeUserMessage, SessionID: "s1", Turn: 1})

	time.Sleep(100 * time.Millisecond)

	line := buf.String()
	if !strings.HasPrefix(line, `{"type":"user:message"`) {
		t.Errorf("unexpected log line: %s", line)
	}
	if !strings.Contains(line, `"session_id":"s1"`) {
		t.Errorf("missing session_id: %s", line)
	}
}

func TestWebhookDelivery(t *testing.T) {
	var received []Event
	var mu sync.Mutex
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var evt Event
		json.NewDecoder(r.Body).Decode(&evt)
		mu.Lock()
		received = append(received, evt)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	bus := NewEventBus(16)
	bus.SetWebhook(ts.URL, []Type{"*"})

	bus.Emit(context.Background(), Event{Type: TypeToolCompleted, SessionID: "s1", Turn: 2})

	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 webhook event, got %d", len(received))
	}
	if received[0].Type != TypeToolCompleted {
		t.Errorf("expected tool:completed, got %s", received[0].Type)
	}
}

func TestWebhookFilteredEvents(t *testing.T) {
	var mu sync.Mutex
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	bus := NewEventBus(16)
	bus.SetWebhook(ts.URL, []Type{TypeError})

	bus.Emit(context.Background(), Event{Type: TypeToolCalled, SessionID: "s1"})
	bus.Emit(context.Background(), Event{Type: TypeError, SessionID: "s1"})

	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if count != 1 {
		t.Errorf("expected 1 webhook call (only error), got %d", count)
	}
}

func TestHandlerPanicRecovery(t *testing.T) {
	bus := NewEventBus(16)
	var mu sync.Mutex
	ranAfter := false

	bus.On(TypeError, func(ctx context.Context, evt Event) {
		panic("oops")
	})
	bus.On(TypeError, func(ctx context.Context, evt Event) {
		mu.Lock()
		ranAfter = true
		mu.Unlock()
	})

	bus.Emit(context.Background(), Event{Type: TypeError, SessionID: "s1"})

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if !ranAfter {
		t.Error("second handler should run after first panics")
	}
}

func TestTimestampAutoSet(t *testing.T) {
	bus := NewEventBus(16)
	var received Event
	done := make(chan struct{})

	bus.On(TypeHeartbeat, func(ctx context.Context, evt Event) {
		received = evt
		close(done)
	})

	bus.Emit(context.Background(), Event{Type: TypeHeartbeat, SessionID: "s1"})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	if received.Timestamp.IsZero() {
		t.Error("timestamp should be auto-set")
	}
}

func TestEmitNonBlocking(t *testing.T) {
	bus := NewEventBus(2) // small buffer

	// Register a slow handler
	blocker := make(chan struct{})
	bus.On(TypeError, func(ctx context.Context, evt Event) {
		<-blocker
	})

	// Emit should return immediately even with slow handler
	start := time.Now()
	for i := 0; i < 10; i++ {
		bus.Emit(context.Background(), Event{Type: TypeError, SessionID: "s1"})
	}
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Emit should be non-blocking, took %v", elapsed)
	}

	close(blocker)
}
