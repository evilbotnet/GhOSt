// Package office manages the local CryptPad instance. CryptPad is a Node
// process (~250 MB), so it runs on demand: started when the Office app opens,
// stopped ten minutes after the last Office window closes.
package office

import (
	"net/http"
	"os/exec"
	"sync"
	"time"
)

const unit = "ghost-cryptpad.service"
const idleStop = 10 * time.Minute

type Manager struct {
	url string // e.g. http://localhost:3000 ("" = office not configured)

	mu        sync.Mutex
	openCount int
	lastClose time.Time
}

func New(url string) *Manager {
	m := &Manager{url: url}
	if url != "" {
		go m.idleLoop()
	}
	return m
}

func (m *Manager) Available() bool { return m.url != "" }
func (m *Manager) URL() string     { return m.url }

// Running probes CryptPad itself — truthful regardless of how it was started.
func (m *Manager) Running() bool {
	if m.url == "" {
		return false
	}
	c := http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := c.Get(m.url)
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

// Open marks an Office window opened and ensures the service is starting.
func (m *Manager) Open() {
	m.mu.Lock()
	m.openCount++
	m.mu.Unlock()
	// Fire and forget: on dev hosts without the unit this fails harmlessly
	// (Running() stays false and the shell shows the placeholder).
	exec.Command("systemctl", "--user", "start", unit).Run()
}

func (m *Manager) Close() {
	m.mu.Lock()
	if m.openCount > 0 {
		m.openCount--
	}
	if m.openCount == 0 {
		m.lastClose = time.Now()
	}
	m.mu.Unlock()
}

func (m *Manager) idleLoop() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		m.mu.Lock()
		idle := m.openCount == 0 && !m.lastClose.IsZero() && time.Since(m.lastClose) > idleStop
		if idle {
			m.lastClose = time.Time{}
		}
		m.mu.Unlock()
		if idle {
			exec.Command("systemctl", "--user", "stop", unit).Run()
		}
	}
}
