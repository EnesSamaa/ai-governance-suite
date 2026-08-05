package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

// Event is an MCP event delivered through an SSE stream.
type Event struct {
	ID    string
	Type  string
	Data  string
	Retry int
}

// Broker owns active subscribers and broadcasts events without blocking publishers.
type Broker struct {
	mu          sync.RWMutex
	subscribers map[uint64]chan Event
	nextID      atomic.Uint64
}

func NewBroker() *Broker {
	return &Broker{subscribers: make(map[uint64]chan Event)}
}

func (b *Broker) Subscribe(buffer int) (<-chan Event, func()) {
	if buffer < 1 {
		buffer = 1
	}
	id := b.nextID.Add(1)
	channel := make(chan Event, buffer)
	b.mu.Lock()
	b.subscribers[id] = channel
	b.mu.Unlock()
	var once sync.Once
	return channel, func() {
		once.Do(func() {
			b.mu.Lock()
			if current, ok := b.subscribers[id]; ok {
				delete(b.subscribers, id)
				close(current)
			}
			b.mu.Unlock()
		})
	}
}

// Publish broadcasts an event. Slow clients drop the newest event rather than
// blocking an MCP request path.
func (b *Broker) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, subscriber := range b.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (b *Broker) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// ServeHTTP exposes the broker as an SSE endpoint.
func (b *Broker) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		http.Error(writer, "SSE requires GET", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := writer.(http.Flusher)
	if !ok {
		http.Error(writer, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	channel, unsubscribe := b.Subscribe(32)
	defer unsubscribe()
	writer.WriteHeader(http.StatusOK)
	flusher.Flush()
	for {
		select {
		case <-request.Context().Done():
			return
		case event, open := <-channel:
			if !open {
				return
			}
			if _, err := fmt.Fprint(writer, FormatEvent(event)); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// FormatEvent serializes a safe SSE frame. Newlines become separate data lines.
func FormatEvent(event Event) string {
	var frame strings.Builder
	if event.ID != "" {
		fmt.Fprintf(&frame, "id: %s\n", event.ID)
	}
	if event.Type != "" {
		fmt.Fprintf(&frame, "event: %s\n", event.Type)
	}
	if event.Retry > 0 {
		fmt.Fprintf(&frame, "retry: %d\n", event.Retry)
	}
	for _, line := range strings.Split(event.Data, "\n") {
		fmt.Fprintf(&frame, "data: %s\n", line)
	}
	frame.WriteString("\n")
	return frame.String()
}

func main() {
	broker := NewBroker()
	mux := http.NewServeMux()
	mux.Handle("/mcp/events", broker)
	_ = http.ListenAndServe(":8080", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/mcp/events" {
			mux.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	}))
}

// WaitForEvent is useful to consumers that need cancellation-aware reads.
func WaitForEvent(ctx context.Context, events <-chan Event) (Event, bool) {
	select {
	case <-ctx.Done():
		return Event{}, false
	case event, ok := <-events:
		return event, ok
	}
}
