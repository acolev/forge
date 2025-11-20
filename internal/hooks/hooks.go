package hooks

import (
	"context"
	"sync"
)

// Event represents a generic hook/event that can be emitted and handled by plugins.
// Example: hooks.Emit(ctx, hooks.Event{Name: "project.create.before"})
type Event struct {
	Name    string
	Payload any
}

// Handler is implemented by plugins that want to handle events.
type Handler interface {
	Handle(ctx context.Context, Name string, Payload *any) error
}

var (
	mu       sync.RWMutex
	handlers []Handler
)

// Register registers a handler to receive emitted events.
func Register(h Handler) {
	if h == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	handlers = append(handlers, h)
}

// Emit emits an event to all registered handlers. If a handler returns an error,
// Emit stops and returns that error.
func Emit(ctx context.Context, e Event) error {
	mu.RLock()
	hs := append([]Handler(nil), handlers...)
	mu.RUnlock()

	for _, h := range hs {
		if err := h.Handle(ctx, e.Name, &e.Payload); err != nil {
			return err
		}
	}
	return nil
}
