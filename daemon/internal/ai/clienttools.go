package ai

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ghostos/ghostd/internal/ws"
)

// Client tools let a *running app* expose tools to Ghost (ADR 0006). An app —
// the shell today, an .osapp tomorrow — registers tool definitions over the
// WS topic "ghosttools"; Ghost merges them into its loop. When Ghost calls one,
// the daemon emits an "invoke" to the app and awaits its "result" — the same
// request/response-over-WS pattern as the confirmation gate. Mutating client
// tools are still Allow/Deny-gated like everything else.
//
// v1 assumes a single registering client (the shell). Multi-client routing
// (which app owns which tool) is a clean later addition.

type clientToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Properties  map[string]any `json:"properties"`
	Required    []string       `json:"required"`
	Mutating    bool           `json:"mutating"`
}

type ctResult struct {
	output string
	err    string
}

type clientToolReg struct {
	hub     *ws.Hub
	mu      sync.Mutex
	defs    map[string]clientToolDef
	pending map[string]chan ctResult
}

func newClientToolReg(hub *ws.Hub) *clientToolReg {
	r := &clientToolReg{
		hub:     hub,
		defs:    map[string]clientToolDef{},
		pending: map[string]chan ctResult{},
	}
	hub.HandlePrefix("ghosttools", r.handle)
	return r
}

func (r *clientToolReg) handle(topic, event string, payload json.RawMessage) {
	switch event {
	case "register":
		var p struct {
			Tools []clientToolDef `json:"tools"`
		}
		if json.Unmarshal(payload, &p) != nil {
			return
		}
		r.mu.Lock()
		r.defs = map[string]clientToolDef{}
		for _, t := range p.Tools {
			if t.Name != "" {
				r.defs[t.Name] = t
			}
		}
		r.mu.Unlock()
	case "result":
		var p struct {
			CallID string `json:"callId"`
			Output string `json:"output"`
			Error  string `json:"error"`
		}
		if json.Unmarshal(payload, &p) != nil {
			return
		}
		r.mu.Lock()
		ch := r.pending[p.CallID]
		r.mu.Unlock()
		if ch != nil {
			ch <- ctResult{output: p.Output, err: p.Error}
		}
	}
}

// tools returns the currently-registered client tools as runnable Ghost tools.
func (r *clientToolReg) tools() map[string]tool {
	r.mu.Lock()
	defs := make([]clientToolDef, 0, len(r.defs))
	for _, d := range r.defs {
		defs = append(defs, d)
	}
	r.mu.Unlock()

	out := map[string]tool{}
	for _, d := range defs {
		d := d
		props := d.Properties
		if props == nil {
			props = map[string]any{}
		}
		out[d.Name] = tool{
			def: ToolDef{
				Name:        d.Name,
				Description: "[app] " + d.Description,
				Properties:  props,
				Required:    d.Required,
			},
			mutating: d.Mutating,
			run: func(args map[string]any) (string, error) {
				return r.invoke(d.Name, args)
			},
		}
	}
	return out
}

// newID returns a short random hex id, used for WS callIds and session ids.
func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (r *clientToolReg) invoke(name string, args map[string]any) (string, error) {
	callID := newID()

	ch := make(chan ctResult, 1)
	r.mu.Lock()
	r.pending[callID] = ch
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		delete(r.pending, callID)
		r.mu.Unlock()
	}()

	r.hub.Publish("ghosttools", "invoke", map[string]any{
		"callId": callID, "name": name, "args": args,
	})

	select {
	case res := <-ch:
		if res.err != "" {
			return "", fmt.Errorf("%s", res.err)
		}
		if res.output == "" {
			return "(done)", nil
		}
		return res.output, nil
	case <-time.After(30 * time.Second):
		return "", fmt.Errorf("app tool %q did not respond", name)
	}
}
