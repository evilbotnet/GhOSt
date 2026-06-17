// Package kv is a tiny string key/value store for shell preferences (theme,
// etc.), persisted to ~/.config/ghost/settings.json.
package kv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	mu   sync.Mutex
	path string
}

func New() *Store {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "ghost")
	os.MkdirAll(dir, 0o700)
	return &Store{path: filepath.Join(dir, "settings.json")}
}

func (s *Store) load() map[string]string {
	m := map[string]string{}
	if data, err := os.ReadFile(s.path); err == nil {
		json.Unmarshal(data, &m)
	}
	return m
}

func (s *Store) All() map[string]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

func (s *Store) Set(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.load()
	m[key] = value
	data, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(s.path, data, 0o600)
}
