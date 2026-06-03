package community

import (
	"encoding/json"
	"sync"
)

// Hub maintains connected WebSocket clients and broadcasts messages.
type Hub struct {
	mu      sync.RWMutex
	clients map[chan []byte]uint // channel → userID
}

func NewHub() *Hub {
	return &Hub{clients: make(map[chan []byte]uint)}
}

func (h *Hub) Subscribe(userID uint) chan []byte {
	ch := make(chan []byte, 32)
	h.mu.Lock()
	h.clients[ch] = userID
	h.mu.Unlock()
	return ch
}

func (h *Hub) Unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

// OnlineIDs returns the set of user IDs currently connected via WebSocket.
func (h *Hub) OnlineIDs() map[uint]bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make(map[uint]bool, len(h.clients))
	for _, uid := range h.clients {
		out[uid] = true
	}
	return out
}

func (h *Hub) Broadcast(event WSEvent) {
	b, _ := json.Marshal(event)
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- b:
		default:
		}
	}
}
