// Package windows tracks native Wayland toplevels (browser windows, apt-
// installed GUI apps) via wlrctl's foreign-toplevel-management client and
// publishes them on the `windows` WS topic so the shell taskbar can show and
// control them. v1 polls; a native Go Wayland client is the contained later
// upgrade (plan risk #2).
package windows

import (
	"fmt"
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ghostos/ghostd/internal/ws"
)

type Toplevel struct {
	AppID string `json:"appId"`
	Title string `json:"title"`
}

type Manager struct {
	mu        sync.Mutex
	available bool
	last      []Toplevel
}

// shellAppPrefix identifies the shell's own Chromium --app window, which is
// the desktop itself and must not appear in the taskbar.
const shellAppPrefix = "chrome-127.0.0.1"

func NewManager(hub *ws.Hub) *Manager {
	m := &Manager{}
	if _, err := exec.LookPath("wlrctl"); err == nil {
		m.available = true
		go m.pollLoop(hub)
	}
	return m
}

func (m *Manager) Available() bool { return m.available }

func (m *Manager) pollLoop(hub *ws.Hub) {
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()
	for range t.C {
		tops, err := m.List()
		if err != nil {
			continue
		}
		m.mu.Lock()
		changed := !reflect.DeepEqual(tops, m.last)
		m.last = tops
		m.mu.Unlock()
		if changed {
			hub.Publish("windows", "list", tops)
		}
	}
}

func (m *Manager) List() ([]Toplevel, error) {
	out, err := exec.Command("wlrctl", "toplevel", "list").Output()
	if err != nil {
		return nil, err
	}
	tops := []Toplevel{}
	for _, line := range strings.Split(strings.TrimRight(string(out), "\n"), "\n") {
		appID, title, ok := strings.Cut(line, ": ")
		if !ok || strings.HasPrefix(appID, shellAppPrefix) {
			continue
		}
		tops = append(tops, Toplevel{AppID: appID, Title: title})
	}
	sort.Slice(tops, func(i, j int) bool {
		if tops[i].AppID != tops[j].AppID {
			return tops[i].AppID < tops[j].AppID
		}
		return tops[i].Title < tops[j].Title
	})
	return tops, nil
}

// Act runs focus/minimize/maximize/close against windows matching appID
// (and title when given — wlrctl matches are exact strings).
func (m *Manager) Act(action, appID, title string) error {
	switch action {
	case "focus", "minimize", "maximize", "close":
	default:
		return fmt.Errorf("unknown window action %q", action)
	}
	if !m.available {
		return fmt.Errorf("window control unavailable (no wlrctl)")
	}
	args := []string{"toplevel", action}
	if appID != "" {
		args = append(args, "app_id:"+appID)
	}
	if title != "" {
		args = append(args, "title:"+title)
	}
	out, err := exec.Command("wlrctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("wlrctl: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
