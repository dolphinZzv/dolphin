package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"dolphinzZ/internal/config"
)

// ToolDefinition is the public description of a tool.
type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// ToolCall is a request to execute a tool.
type ToolCall struct {
	Name      string
	Arguments json.RawMessage
}

// ToolResult is the result of a tool execution.
type ToolResult struct {
	Content string
	IsError bool
}

// Tool is the interface all MCP tools must implement.
type Tool interface {
	Definition() ToolDefinition
	Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error)
}

// Registry manages all registered MCP tools, including external server tools.
type Registry struct {
	mu      sync.RWMutex
	tools   map[string]Tool
	servers []*ServerClient
	cfg     *config.MCPConfig
}

func NewRegistry(cfg *config.Config) *Registry {
	return &Registry{
		tools:   make(map[string]Tool),
		servers: make([]*ServerClient, 0),
		cfg:     &cfg.MCP,
	}
}

func (r *Registry) Register(t Tool) {
	def := t.Definition()
	r.mu.Lock()
	r.tools[def.Name] = t
	r.mu.Unlock()
}

// LoadServers starts external MCP servers defined in config and registers their tools.
func (r *Registry) LoadServers() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for name, cfg := range r.cfg.Servers {
		client, err := NewServerClient(name, cfg)
		if err != nil {
			return fmt.Errorf("mcp server %q: %w", name, err)
		}

		defs, err := client.ListTools()
		if err != nil {
			client.Close()
			return fmt.Errorf("list tools from %q: %w", name, err)
		}

		for _, def := range defs {
			wrapper := &serverTool{
				server: client,
				def:    def,
			}
			// Always register with server:name prefix for disambiguation
			r.tools[name+":"+def.Name] = wrapper
			slog.Debug("mcp tool registered", "tool", name+":"+def.Name, "server", name)
			// Also register with bare name if no collision
			if _, exists := r.tools[def.Name]; !exists {
				r.tools[def.Name] = wrapper
				slog.Debug("mcp tool registered (bare)", "tool", def.Name, "server", name)
			}
		}

		r.servers = append(r.servers, client)
	}

	return nil
}

// CloseServers shuts down all external MCP servers.
func (r *Registry) CloseServers() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.servers {
		s.Close()
	}
	r.servers = nil
}

func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) List() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	defs := make([]ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, t.Definition())
	}
	return defs
}

func (r *Registry) Execute(ctx context.Context, name string, input json.RawMessage) (*ToolResult, error) {
	tool, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool.Execute(ctx, input)
}

// serverTool wraps an external MCP server tool for the Tool interface.
type serverTool struct {
	server *ServerClient
	def    ToolDefinition
}

func (st *serverTool) Definition() ToolDefinition {
	return st.def
}

func (st *serverTool) Execute(ctx context.Context, input json.RawMessage) (*ToolResult, error) {
	return st.server.CallTool(ctx, st.def.Name, input)
}
