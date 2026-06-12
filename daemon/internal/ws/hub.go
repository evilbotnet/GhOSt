// Package ws implements the single multiplexed WebSocket.
// Envelope: {"topic": "...", "event": "...", "payload": ...}.
// Clients subscribe per topic; subsystems publish to topics and may register
// prefix handlers for client->server events (e.g. "term." input).
package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

type Envelope struct {
	Topic   string          `json:"topic"`
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type outEnvelope struct {
	Topic   string `json:"topic"`
	Event   string `json:"event"`
	Payload any    `json:"payload,omitempty"`
}

// Handler receives client->server events for a topic prefix.
type Handler func(topic, event string, payload json.RawMessage)

type client struct {
	conn   *websocket.Conn
	send   chan []byte
	topics map[string]bool
	mu     sync.Mutex
}

type Hub struct {
	mu       sync.RWMutex
	clients  map[*client]bool
	handlers map[string]Handler // key: topic prefix
	upgrader websocket.Upgrader
}

func NewHub() *Hub {
	return &Hub{
		clients:  make(map[*client]bool),
		handlers: make(map[string]Handler),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			// Origin is validated by the HTTP auth middleware before upgrade.
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// HandlePrefix registers a handler for client events on topics with the prefix.
func (h *Hub) HandlePrefix(prefix string, fn Handler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[prefix] = fn
}

// Publish sends an event to every client subscribed to the topic.
func (h *Hub) Publish(topic, event string, payload any) {
	data, err := json.Marshal(outEnvelope{Topic: topic, Event: event, Payload: payload})
	if err != nil {
		log.Printf("ws marshal: %v", err)
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		c.mu.Lock()
		subscribed := c.topics[topic]
		if !subscribed {
			// prefix subscriptions, e.g. client subscribed to "term."
			for t := range c.topics {
				if strings.HasSuffix(t, ".") && strings.HasPrefix(topic, t) {
					subscribed = true
					break
				}
			}
		}
		c.mu.Unlock()
		if subscribed {
			select {
			case c.send <- data:
			default: // slow client: drop frame rather than block the publisher
			}
		}
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := &client{
		conn:   conn,
		send:   make(chan []byte, 256),
		topics: make(map[string]bool),
	}
	h.mu.Lock()
	h.clients[c] = true
	h.mu.Unlock()

	go c.writeLoop()
	h.readLoop(c)
}

func (h *Hub) readLoop(c *client) {
	defer func() {
		h.mu.Lock()
		delete(h.clients, c)
		h.mu.Unlock()
		close(c.send)
		c.conn.Close()
	}()
	c.conn.SetReadLimit(1 << 20)
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		var env Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		switch env.Event {
		case "subscribe":
			c.mu.Lock()
			c.topics[env.Topic] = true
			c.mu.Unlock()
		case "unsubscribe":
			c.mu.Lock()
			delete(c.topics, env.Topic)
			c.mu.Unlock()
		default:
			h.dispatch(env)
		}
	}
}

func (h *Hub) dispatch(env Envelope) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for prefix, fn := range h.handlers {
		if strings.HasPrefix(env.Topic, prefix) {
			fn(env.Topic, env.Event, env.Payload)
			return
		}
	}
}

func (c *client) writeLoop() {
	for data := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}
	}
}
