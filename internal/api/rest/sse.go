package rest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// SSEHub manages real-time broadcast connections for web clients
type SSEHub struct {
	mu         sync.Mutex
	clients    map[chan []byte]bool
	register   chan chan []byte
	unregister chan chan []byte
	broadcast  chan []byte
	quit       chan struct{}
}

// NewSSEHub initializes a concurrent event broadcaster
func NewSSEHub() *SSEHub {
	hub := &SSEHub{
		clients:    make(map[chan []byte]bool),
		register:   make(chan chan []byte),
		unregister: make(chan chan []byte),
		broadcast:  make(chan []byte, 1024),
		quit:       make(chan struct{}),
	}

	go hub.run()
	return hub
}

func (h *SSEHub) run() {
	for {
		select {
		case <-h.quit:
			h.mu.Lock()
			for ch := range h.clients {
				close(ch)
			}
			h.clients = make(map[chan []byte]bool)
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client)
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client <- message:
				default:
					// Slow client: drop message to prevent blocking
				}
			}
			h.mu.Unlock()
		}
	}
}

// BroadcastEvent formats an event payload and sends it to all active SSE subscribers
func (h *SSEHub) BroadcastEvent(eventType string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	msg := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(data))
	select {
	case h.broadcast <- []byte(msg):
	default:
	}
}

// ServeHTTP handles incoming HTTP streaming requests from browsers
func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan []byte, 64)
	h.register <- clientChan

	defer func() {
		h.unregister <- clientChan
	}()

	// Send initial connected handshake
	initMsg := fmt.Sprintf("event: connected\ndata: {\"timestamp\":\"%s\"}\n\n", time.Now().UTC().Format(time.RFC3339))
	_, _ = w.Write([]byte(initMsg))
	flusher.Flush()

	notify := r.Context().Done()
	keepAliveTicker := time.NewTicker(15 * time.Second)
	defer keepAliveTicker.Stop()

	for {
		select {
		case <-notify:
			return
		case <-keepAliveTicker.C:
			_, _ = w.Write([]byte(": ping\n\n"))
			flusher.Flush()
		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			if _, err := w.Write(msg); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// Close gracefully closes the SSE broadcaster
func (h *SSEHub) Close() {
	close(h.quit)
}
