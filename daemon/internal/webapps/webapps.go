// Package webapps implements Layer 1 of the app platform (ADR 0001): any URL
// installed as a first-class app that opens as its own chromeless browser
// window with its own taskbar entry. Stored as JSON in the user's config.
package webapps

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type App struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Icon string `json:"icon"` // a glyph name from the shell icon set
}

type Store struct {
	mu   sync.Mutex
	path string
}

func New() *Store {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "ghost")
	os.MkdirAll(dir, 0o700)
	return &Store{path: filepath.Join(dir, "webapps.json")}
}

func (s *Store) load() []App {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil
	}
	var apps []App
	json.Unmarshal(data, &apps)
	return apps
}

func (s *Store) save(apps []App) error {
	data, _ := json.MarshalIndent(apps, "", "  ")
	return os.WriteFile(s.path, data, 0o600)
}

func (s *Store) List() []App {
	s.mu.Lock()
	defer s.mu.Unlock()
	apps := s.load()
	if apps == nil {
		apps = []App{}
	}
	return apps
}

func (s *Store) Install(name, rawURL, icon string) (App, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return App{}, fmt.Errorf("a web app needs an http(s) URL")
	}
	if name == "" {
		name = u.Hostname()
	}
	if icon == "" {
		icon = "browser"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	apps := s.load()
	app := App{
		ID:   fmt.Sprintf("web-%d", time.Now().UnixNano()),
		Name: name,
		URL:  u.String(),
		Icon: icon,
	}
	apps = append(apps, app)
	return app, s.save(apps)
}

func (s *Store) Uninstall(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	apps := s.load()
	out := apps[:0]
	for _, a := range apps {
		if a.ID != id {
			out = append(out, a)
		}
	}
	return s.save(out)
}

func (s *Store) URLFor(id string) (string, bool) {
	for _, a := range s.List() {
		if a.ID == id {
			return a.URL, true
		}
	}
	return "", false
}

// IconForURL guesses a sensible glyph from the hostname.
func IconForURL(raw string) string {
	u, _ := url.Parse(raw)
	host := ""
	if u != nil {
		host = strings.ToLower(u.Hostname())
	}
	switch {
	case strings.Contains(host, "github"):
		return "editor"
	case strings.Contains(host, "mail") || strings.Contains(host, "proton"):
		return "file"
	case strings.Contains(host, "music") || strings.Contains(host, "spotify"):
		return "volume"
	default:
		return "browser"
	}
}
