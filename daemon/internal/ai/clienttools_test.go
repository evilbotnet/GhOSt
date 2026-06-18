package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ghostos/ghostd/internal/ws"
	"github.com/gorilla/websocket"
)

// TestClientToolRoundTrip exercises ADR 0006 end-to-end over a real WebSocket:
// a "shell" client registers a tool, Ghost invokes it, the client answers, and
// the invocation returns the client's output — the full app→Ghost channel.
func TestClientToolRoundTrip(t *testing.T) {
	hub := ws.NewHub()
	reg := newClientToolReg(hub)

	srv := httptest.NewServer(http.HandlerFunc(hub.ServeHTTP))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	send := func(event string, payload any) {
		conn.WriteJSON(map[string]any{"topic": "ghosttools", "event": event, "payload": payload})
	}

	// The shell subscribes, then registers its tools.
	send("subscribe", nil)
	send("register", map[string]any{"tools": []map[string]any{{
		"name":        "open_app",
		"description": "open a desktop app",
		"properties":  map[string]any{"id": map[string]any{"type": "string"}},
		"required":    []string{"id"},
		"mutating":    false,
	}}})

	// The shell answers invoke frames with a result.
	go func() {
		for {
			var env struct {
				Topic, Event string
				Payload      json.RawMessage
			}
			if conn.ReadJSON(&env) != nil {
				return
			}
			if env.Event != "invoke" {
				continue
			}
			var inv struct {
				CallID string         `json:"callId"`
				Name   string         `json:"name"`
				Args   map[string]any `json:"args"`
			}
			json.Unmarshal(env.Payload, &inv)
			send("result", map[string]any{
				"callId": inv.CallID,
				"output": "opened " + inv.Name + "(" + str(inv.Args, "id") + ")",
			})
		}
	}()

	// Wait for the register to land (it travels in over the socket).
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, ok := reg.tools()["open_app"]; ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("tool never registered")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Ghost invokes the client tool and should get the shell's output back.
	out, err := reg.invoke("open_app", map[string]any{"id": "monitor"})
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if out != "opened open_app(monitor)" {
		t.Fatalf("unexpected output: %q", out)
	}
}
