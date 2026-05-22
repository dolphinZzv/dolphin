package subsystem

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"dolphin/internal/mcp"
)

// ---- test helpers ----

type testProvider struct {
	name      string
	contextMD string
	toolDefs  []ToolDef
}

func (p *testProvider) Name() string        { return p.name }
func (p *testProvider) ContextMD() string   { return p.contextMD }
func (p *testProvider) ToolDefs() []ToolDef { return p.toolDefs }

// resetProviders clears the global registry for testing.
func resetProviders() {
	mu.Lock()
	defer mu.Unlock()
	providers = nil
}

func TestRegisterAndContextMD(t *testing.T) {
	resetProviders()
	defer resetProviders()

	Register(&testProvider{name: "alpha", contextMD: "## Alpha\ncontent a"})
	Register(&testProvider{name: "beta", contextMD: "## Beta\ncontent b"})

	md := ContextMD()
	if md == "" {
		t.Fatal("expected non-empty ContextMD")
	}
	if md != "## Alpha\ncontent a\n\n## Beta\ncontent b" {
		t.Errorf("unexpected ContextMD:\n%s", md)
	}
}

func TestContextMD_Empty(t *testing.T) {
	resetProviders()
	defer resetProviders()

	if md := ContextMD(); md != "" {
		t.Errorf("expected empty, got %q", md)
	}
}

func TestContextMD_SkipEmpty(t *testing.T) {
	resetProviders()
	defer resetProviders()

	Register(&testProvider{name: "empty", contextMD: ""})
	Register(&testProvider{name: "full", contextMD: "## Full\nok"})

	md := ContextMD()
	if md != "## Full\nok" {
		t.Errorf("expected only full provider, got %q", md)
	}
}

func TestToolDefs_Aggregation(t *testing.T) {
	resetProviders()
	defer resetProviders()

	handler := func(_ context.Context, _ json.RawMessage) (*mcp.ToolResult, error) {
		return &mcp.ToolResult{Content: "ok"}, nil
	}

	Register(&testProvider{
		name: "a",
		toolDefs: []ToolDef{
			{Name: "tool_a1", Handler: handler},
			{Name: "tool_a2", SelfEvolution: true, Handler: handler},
		},
	})
	Register(&testProvider{
		name: "b",
		toolDefs: []ToolDef{
			{Name: "tool_b1", Handler: handler},
		},
	})

	defs := ToolDefs()
	if len(defs) != 3 {
		t.Fatalf("expected 3 tool defs, got %d", len(defs))
	}
	if defs[0].Name != "tool_a1" || defs[1].Name != "tool_a2" || defs[2].Name != "tool_b1" {
		t.Errorf("unexpected tool order: [%s, %s, %s]", defs[0].Name, defs[1].Name, defs[2].Name)
	}
	if !defs[1].SelfEvolution {
		t.Error("tool_a2 should have SelfEvolution=true")
	}
	if defs[2].SelfEvolution {
		t.Error("tool_b1 should have SelfEvolution=false")
	}
}

func TestToolDefs_Empty(t *testing.T) {
	resetProviders()
	defer resetProviders()

	if defs := ToolDefs(); defs != nil {
		t.Errorf("expected nil, got %v", defs)
	}
}

func TestRegister_DuplicatePanic(t *testing.T) {
	resetProviders()
	defer resetProviders()

	Register(&testProvider{name: "dup", contextMD: "x"})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate register")
		}
	}()
	Register(&testProvider{name: "dup", contextMD: "y"})
}

func TestConcurrentReads(t *testing.T) {
	resetProviders()
	defer resetProviders()

	Register(&testProvider{name: "conc", contextMD: "## Conc\nok", toolDefs: []ToolDef{
		{Name: "tool_c", Handler: func(_ context.Context, _ json.RawMessage) (*mcp.ToolResult, error) {
			return &mcp.ToolResult{Content: "ok"}, nil
		}},
	}})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ContextMD()
			_ = ToolDefs()
		}()
	}
	wg.Wait()
}
