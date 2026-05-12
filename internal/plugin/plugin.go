// Package plugin provides the plugin system for Dolphin.
// Plugins register hooks (synchronous interception) and event handlers
// (asynchronous notification) via a simple Register(reg) method.
package plugin

import (
	"dolphin/internal/event"
	"dolphin/internal/hook"
)

// Plugin is the only interface a plugin must implement.
type Plugin interface {
	Name() string
	Register(reg *Registry)
}

// Registry exposes hook and event registration during plugin activation.
type Registry struct {
	hooks  map[hook.Point][]hook.Registration
	events map[event.Type][]event.Handler

	HooksAdded  int
	EventsAdded int
}

// NewRegistry returns an empty plugin Registry.
func NewRegistry() *Registry {
	return &Registry{
		hooks:  make(map[hook.Point][]hook.Registration),
		events: make(map[event.Type][]event.Handler),
	}
}

// AddHook registers a hook for the given point with priority (lower = earlier).
func (r *Registry) AddHook(point hook.Point, priority int, h hook.Handler) {
	r.hooks[point] = append(r.hooks[point], hook.Registration{Priority: priority, Handler: h})
	r.HooksAdded++
}

// AddEvent registers an event handler. Use "*" for all events.
func (r *Registry) AddEvent(t event.Type, h event.Handler) {
	r.events[t] = append(r.events[t], h)
	r.EventsAdded++
}

// ApplyTo copies all registered hooks and events into the target hook.Registry
// and event.EventBus. Called by Manager after all plugins have registered.
func (r *Registry) ApplyTo(hooks *hook.Registry, bus *event.EventBus) {
	for point, regs := range r.hooks {
		for _, reg := range regs {
			hooks.Register(point, reg.Priority, reg.Handler)
		}
	}
	for evtType, handlers := range r.events {
		for _, h := range handlers {
			bus.On(evtType, h)
		}
	}
}
