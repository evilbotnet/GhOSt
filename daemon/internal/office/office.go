// Package office manages the local CryptPad instance. CryptPad is a Node
// process (~250 MB), so it runs on demand: started when the Office app opens,
// stopped ten minutes after the last Office window closes.
package office

import (
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const unit = "ghost-cryptpad.service"
const idleStop = 10 * time.Minute

type Manager struct {
	url string // e.g. http://localhost:3000 ("" = office not configured)
	// windowPresent reports whether a native Office window is still open
	// (CryptPad runs in a chromium --app window; see ADR/plan: its CSP
	// forbids iframing from the shell origin).
	windowPresent func() bool

	mu        sync.Mutex
	openCount int
	lastClose time.Time
}

func New(url string, windowPresent func() bool) *Manager {
	if windowPresent == nil {
		windowPresent = func() bool { return false }
	}
	m := &Manager{url: url, windowPresent: windowPresent}
	if url != "" {
		go m.idleLoop()
		// CryptPad needs a second browser origin for its sandbox iframe, but
		// modern CryptPad only listens on one port and expects a reverse
		// proxy. ghostd IS that proxy: a dumb TCP forward, e.g.
		//   GHOST_OFFICE_SAFE_PROXY=127.0.0.1:3001=127.0.0.1:3000
		if spec := os.Getenv("GHOST_OFFICE_SAFE_PROXY"); spec != "" {
			if listen, target, ok := strings.Cut(spec, "="); ok {
				go safeOriginProxy(listen, target)
			}
		}
	}
	return m
}

func safeOriginProxy(listen, target string) {
	l, err := net.Listen("tcp", listen)
	if err != nil {
		return // port taken (another ghostd) or misconfigured — office still works unproxied
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		go func() {
			defer conn.Close()
			up, err := net.Dial("tcp", target)
			if err != nil {
				return
			}
			defer up.Close()
			go io.Copy(up, conn)
			io.Copy(conn, up)
		}()
	}
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
		m.mu.Unlock()
		if idle && !m.windowPresent() {
			m.mu.Lock()
			m.lastClose = time.Time{}
			m.mu.Unlock()
			exec.Command("systemctl", "--user", "stop", unit).Run()
		}
	}
}
