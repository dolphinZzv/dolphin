package health

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCheckerOK(t *testing.T) {
	c := NewChecker("test", func(ctx context.Context) error {
		return nil
	})
	cs := c.Check(context.Background())
	if cs.Status != StatusOK {
		t.Errorf("expected ok, got %s", cs.Status)
	}
	if cs.Name != "test" {
		t.Errorf("expected name 'test', got %s", cs.Name)
	}
}

func TestCheckerError(t *testing.T) {
	c := NewChecker("failing", func(ctx context.Context) error {
		return errors.New("something went wrong")
	})
	cs := c.Check(context.Background())
	if cs.Status != StatusError {
		t.Errorf("expected error, got %s", cs.Status)
	}
	if cs.Message != "something went wrong" {
		t.Errorf("expected message, got %s", cs.Message)
	}
}

func TestCheckAllOK(t *testing.T) {
	checkers := []Checker{
		NewChecker("a", func(ctx context.Context) error { return nil }),
		NewChecker("b", func(ctx context.Context) error { return nil }),
	}
	result := CheckAll(context.Background(), checkers)
	if result.Status != StatusOK {
		t.Errorf("expected ok, got %s", result.Status)
	}
	if len(result.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(result.Components))
	}
}

func TestCheckAllDegraded(t *testing.T) {
	checkers := []Checker{
		NewChecker("a", func(ctx context.Context) error { return nil }),
		NewChecker("b", func(ctx context.Context) error { return errors.New("fail") }),
	}
	result := CheckAll(context.Background(), checkers)
	if result.Status != StatusDegraded {
		t.Errorf("expected degraded, got %s", result.Status)
	}
}

func TestHandler(t *testing.T) {
	handler := Handler(
		NewChecker("ok", func(ctx context.Context) error { return nil }),
		NewChecker("err", func(ctx context.Context) error { return errors.New("boom") }),
	)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var result Result
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Status != StatusDegraded {
		t.Errorf("expected degraded, got %s", result.Status)
	}
	if len(result.Components) != 2 {
		t.Errorf("expected 2 components, got %d", len(result.Components))
	}
}
