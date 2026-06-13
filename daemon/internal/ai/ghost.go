package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ghostos/ghostd/internal/ws"
)

const systemPrompt = `You are Ghost, the resident assistant inside GhOSt, a web-native operating system. Your tools ARE the operating system's API — files, the browser, system settings. You act on the user's own machine.

Be concise and direct. When the user asks you to do something, use your tools to actually do it rather than explaining how they could. Explore with read-only tools (list_files, read_file, system_status) freely. For anything that changes the system, the user will be shown a confirmation card before your action runs — so propose the action; don't ask permission in text.

If a tool returns an error, adapt or report it plainly. When the task is done, give a one-line summary of what you did.`

// Ghost runs the confirmation-gated agent loop in the daemon and streams the
// trace to the shell over WS topic ai.<session>.
type Ghost struct {
	hub     *ws.Hub
	toolbox *Toolbox

	mu       sync.Mutex
	sessions map[string]*aiSession
}

type aiSession struct {
	id      string
	history []Msg
	pending map[string]chan bool // confirmation callId -> decision
}

func NewGhost(hub *ws.Hub, toolbox *Toolbox) *Ghost {
	g := &Ghost{hub: hub, toolbox: toolbox, sessions: map[string]*aiSession{}}
	hub.HandlePrefix("ai.", g.handleEvent)
	return g
}

// Configured reports whether an agent-tier provider is set up (Settings/tray
// surface this so the panel can prompt the user to configure Ghost).
func (g *Ghost) Configured() (bool, string) {
	name, _, ok := LoadConfig().AgentProvider()
	return ok, name
}

func newLLM(p Provider) (LLM, error) {
	switch p.Type {
	case "anthropic":
		return newAnthropic(p), nil
	case "openai-compatible":
		return newOpenAI(p), nil
	}
	return nil, fmt.Errorf("unknown provider type %q", p.Type)
}

func (g *Ghost) session(id string) *aiSession {
	g.mu.Lock()
	defer g.mu.Unlock()
	s := g.sessions[id]
	if s == nil {
		s = &aiSession{id: id, pending: map[string]chan bool{}}
		g.sessions[id] = s
	}
	return s
}

func (g *Ghost) emit(id, event string, payload any) {
	g.hub.Publish("ai."+id, event, payload)
}

func (g *Ghost) handleEvent(topic, event string, payload json.RawMessage) {
	id := topic[len("ai."):]
	switch event {
	case "prompt":
		var p struct{ Text string }
		json.Unmarshal(payload, &p)
		go g.run(id, p.Text)
	case "confirm":
		var c struct {
			CallID string `json:"callId"`
			Allow  bool   `json:"allow"`
		}
		json.Unmarshal(payload, &c)
		s := g.session(id)
		s.mu(func() {
			if ch := s.pending[c.CallID]; ch != nil {
				ch <- c.Allow
			}
		})
	}
}

// mu is a tiny helper so we don't hold Ghost.mu during the loop.
func (s *aiSession) mu(fn func()) {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	fn()
}

var sessionMu sync.Mutex

func (g *Ghost) run(id, prompt string) {
	name, p, ok := LoadConfig().AgentProvider()
	if !ok {
		g.emit(id, "error", map[string]string{"message": "Ghost isn't configured yet. Settings → Ghost, or re-run setup."})
		return
	}
	llm, err := newLLM(p)
	if err != nil {
		g.emit(id, "error", map[string]string{"message": err.Error()})
		return
	}

	s := g.session(id)
	s.history = append(s.history, Msg{Role: "user", Text: prompt})
	g.emit(id, "provenance", map[string]string{"provider": name, "model": p.Model})

	tools := g.toolbox.tools()
	defs := g.toolbox.defs()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for step := 0; step < 12; step++ {
		g.emit(id, "thinking", nil)
		turn, err := llm.Chat(ctx, systemPrompt, s.history, defs)
		if err != nil {
			g.emit(id, "error", map[string]string{"message": err.Error()})
			return
		}
		if turn.Text != "" {
			g.emit(id, "message", map[string]string{"text": turn.Text})
		}
		s.history = append(s.history, Msg{Role: "assistant", Text: turn.Text, Calls: turn.Calls})

		if len(turn.Calls) == 0 {
			g.emit(id, "done", nil)
			return
		}

		var results []ToolResult
		for _, call := range turn.Calls {
			t, known := tools[call.Name]
			if !known {
				results = append(results, ToolResult{CallID: call.ID, Content: "unknown tool", IsError: true})
				continue
			}
			if t.mutating && !g.confirm(s, call) {
				g.emit(id, "tool_denied", map[string]string{"name": call.Name})
				results = append(results, ToolResult{CallID: call.ID, Content: "user declined this action", IsError: true})
				continue
			}
			g.emit(id, "tool_run", map[string]any{"name": call.Name, "args": call.Args})
			out, err := t.run(call.Args)
			if err != nil {
				g.emit(id, "tool_result", map[string]any{"name": call.Name, "error": err.Error()})
				results = append(results, ToolResult{CallID: call.ID, Content: err.Error(), IsError: true})
			} else {
				g.emit(id, "tool_result", map[string]any{"name": call.Name, "output": out})
				results = append(results, ToolResult{CallID: call.ID, Content: out})
			}
		}
		s.history = append(s.history, Msg{Role: "tool", Results: results})
	}
	g.emit(id, "done", map[string]string{"note": "stopped after 12 steps"})
}

// confirm shows a confirmation card in the shell and blocks until the user
// decides (or 2 minutes pass — default deny).
func (g *Ghost) confirm(s *aiSession, call ToolCall) bool {
	ch := make(chan bool, 1)
	s.mu(func() { s.pending[call.ID] = ch })
	defer s.mu(func() { delete(s.pending, call.ID) })

	g.emit(s.id, "confirm_request", map[string]any{
		"callId": call.ID, "name": call.Name, "args": call.Args,
	})
	select {
	case ok := <-ch:
		return ok
	case <-time.After(2 * time.Minute):
		return false
	}
}
