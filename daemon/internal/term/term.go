// Package term manages pty sessions, bridged to the shell over the WS hub
// on topics "term.<id>".
package term

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"

	"github.com/ghostos/ghostd/internal/ws"
)

type session struct {
	id  string
	pty *os.File
	cmd *exec.Cmd
}

type Manager struct {
	mu       sync.Mutex
	sessions map[string]*session
	hub      *ws.Hub
}

func NewManager(hub *ws.Hub) *Manager {
	m := &Manager{sessions: make(map[string]*session), hub: hub}
	hub.HandlePrefix("term.", m.handleEvent)
	return m
}

func (m *Manager) Create(cols, rows int) (string, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	cmd := exec.Command(shell, "-l")
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	f, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: uint16(cols), Rows: uint16(rows),
	})
	if err != nil {
		return "", err
	}

	b := make([]byte, 8)
	rand.Read(b)
	id := hex.EncodeToString(b)
	s := &session{id: id, pty: f, cmd: cmd}

	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()

	go m.pump(s)
	return id, nil
}

func (m *Manager) pump(s *session) {
	topic := "term." + s.id
	buf := make([]byte, 16384)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			m.hub.Publish(topic, "data", string(buf[:n]))
		}
		if err != nil {
			break
		}
	}
	m.hub.Publish(topic, "exit", nil)
	m.Close(s.id)
}

func (m *Manager) Close(id string) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	delete(m.sessions, id)
	m.mu.Unlock()
	if !ok {
		return
	}
	s.pty.Close()
	if s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	s.cmd.Wait()
}

func (m *Manager) handleEvent(topic, event string, payload json.RawMessage) {
	id := topic[len("term."):]
	m.mu.Lock()
	s, ok := m.sessions[id]
	m.mu.Unlock()
	if !ok {
		return
	}
	switch event {
	case "input":
		var data string
		if json.Unmarshal(payload, &data) == nil {
			s.pty.Write([]byte(data))
		}
	case "resize":
		var size struct {
			Cols uint16 `json:"cols"`
			Rows uint16 `json:"rows"`
		}
		if json.Unmarshal(payload, &size) == nil {
			pty.Setsize(s.pty, &pty.Winsize{Cols: size.Cols, Rows: size.Rows})
		}
	}
}
