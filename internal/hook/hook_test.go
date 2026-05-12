package hook

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestRegistryPriorityOrder(t *testing.T) {
	r := NewRegistry()
	var order []int

	r.Register(PointBeforeTool, 10, func(ctx context.Context, hc *Context) error { order = append(order, 10); return nil })
	r.Register(PointBeforeTool, 5, func(ctx context.Context, hc *Context) error { order = append(order, 5); return nil })
	r.Register(PointBeforeTool, 1, func(ctx context.Context, hc *Context) error { order = append(order, 1); return nil })

	err := r.Fire(context.Background(), PointBeforeTool, &Context{SessionID: "s1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 3 || order[0] != 1 || order[1] != 5 || order[2] != 10 {
		t.Errorf("expected [1 5 10], got %v", order)
	}
}

func TestRegistryAbortOnError(t *testing.T) {
	r := NewRegistry()
	var ran bool

	r.Register(PointUserInput, 0, func(ctx context.Context, hc *Context) error {
		return errors.New("reject")
	})
	r.Register(PointUserInput, 1, func(ctx context.Context, hc *Context) error {
		ran = true
		return nil
	})

	err := r.Fire(context.Background(), PointUserInput, &Context{SessionID: "s1"})
	if err == nil || err.Error() != "reject" {
		t.Fatalf("expected 'reject' error, got: %v", err)
	}
	if ran {
		t.Error("second handler should not have run after abort")
	}
}

func TestRegistryNoAbortOnNonAbortable(t *testing.T) {
	r := NewRegistry()
	var ranAll bool

	r.Register(PointAfterLLM, 0, func(ctx context.Context, hc *Context) error {
		return errors.New("ignored")
	})
	r.Register(PointAfterLLM, 1, func(ctx context.Context, hc *Context) error {
		ranAll = true
		return nil
	})

	err := r.Fire(context.Background(), PointAfterLLM, &Context{SessionID: "s1"})
	if err != nil {
		t.Fatalf("non-abortable point should not return error, got: %v", err)
	}
	if !ranAll {
		t.Error("all handlers should run at non-abortable points")
	}
}

func TestRegistryUserInputRewrite(t *testing.T) {
	r := NewRegistry()

	r.Register(PointUserInput, 0, func(ctx context.Context, hc *Context) error {
		hc.UserInput = "rewritten by hook"
		return nil
	})

	hc := &Context{SessionID: "s1", UserInput: "original"}
	err := r.Fire(context.Background(), PointUserInput, hc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hc.UserInput != "rewritten by hook" {
		t.Errorf("expected 'rewritten by hook', got %q", hc.UserInput)
	}
}

func TestRegistryToolArgsRewrite(t *testing.T) {
	r := NewRegistry()

	r.Register(PointBeforeTool, 0, func(ctx context.Context, hc *Context) error {
		hc.ToolArgs = []byte(`{"sanitized":true}`)
		return nil
	})

	hc := &Context{SessionID: "s1", ToolArgs: []byte(`{"input":"danger"}`)}
	err := r.Fire(context.Background(), PointBeforeTool, hc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(hc.ToolArgs) != `{"sanitized":true}` {
		t.Errorf("expected sanitized args, got: %s", string(hc.ToolArgs))
	}
}

func TestRegistryValuesSharedAcrossPoints(t *testing.T) {
	r := NewRegistry()

	r.Register(PointBeforeTool, 0, func(ctx context.Context, hc *Context) error {
		hc.Values["start_ns"] = int64(1000)
		return nil
	})
	r.Register(PointAfterTool, 0, func(ctx context.Context, hc *Context) error {
		if start, ok := hc.Values["start_ns"]; !ok || start.(int64) != 1000 {
			t.Errorf("expected start_ns=1000 in Values, got: %v", hc.Values)
		}
		return nil
	})

	hc := &Context{SessionID: "s1", Values: make(map[string]any)}
	r.Fire(context.Background(), PointBeforeTool, hc)
	r.Fire(context.Background(), PointAfterTool, hc)
}

func TestPointAbortable(t *testing.T) {
	abortable := []Point{PointUserInput, PointBeforeLLM, PointBeforeTool}
	for _, p := range abortable {
		if !p.Abortable() {
			t.Errorf("%s should be abortable", p)
		}
	}
	nonAbort := []Point{PointSessionStart, PointSessionEnd, PointAfterLLM, PointAfterTool, PointBeforeResponse, PointOnError}
	for _, p := range nonAbort {
		if p.Abortable() {
			t.Errorf("%s should NOT be abortable", p)
		}
	}
}

func TestHasAny(t *testing.T) {
	r := NewRegistry()
	if r.HasAny(PointBeforeTool) {
		t.Error("empty registry should return false")
	}
	r.Register(PointAfterLLM, 0, func(ctx context.Context, hc *Context) error { return nil })
	if !r.HasAny(PointAfterLLM) {
		t.Error("registered point should return true")
	}
	if r.HasAny(PointBeforeTool) {
		t.Error("unregistered point should return false")
	}
	if !r.HasAny(PointBeforeTool, PointAfterLLM) {
		t.Error("HasAny with one match should return true")
	}
}

func TestConcurrentRegisterAndFire(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup

	// Concurrent registrations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(pri int) {
			defer wg.Done()
			r.Register(PointBeforeTool, pri, func(ctx context.Context, hc *Context) error { return nil })
		}(i)
	}

	// Concurrent fires
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Fire(context.Background(), PointBeforeTool, &Context{SessionID: "s1"})
		}()
	}

	wg.Wait()
}
