// Package osapp implements Layer 2 of the app platform (ADR 0001 / ADR 0009):
// installable .osapp packages — a zip rooted at the app id, holding a
// manifest.json and the app's files. ghostd verifies the package hash, unpacks
// it under ~/.local/share/ghost/apps/<id>/, records the granted permission
// scopes, and serves it at /apps/<id>/ with a scoped token. Built-in shell
// apps are just apps with all scopes; third-party apps get only what their
// manifest declares and the user grants.
package osapp

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Manifest is manifest.json inside a package (see ADR 0009).
type Manifest struct {
	ID          string     `json:"id"`   // reverse-DNS, immutable identity
	Name        string     `json:"name"`
	Version     string     `json:"version"`
	Entry       string     `json:"entry"` // e.g. index.html
	Icon        string     `json:"icon"`
	Window      Window     `json:"window"`
	Permissions []string   `json:"permissions"`         // requested scopes
	GhostTools  []ToolDecl `json:"ghostTools,omitempty"` // tools the app teaches Ghost
	Author      string     `json:"author,omitempty"`
	License     string     `json:"license,omitempty"`
	Source      string     `json:"source,omitempty"`
}

type Window struct {
	W   int  `json:"w"`
	H   int  `json:"h"`
	Min *struct {
		W int `json:"w"`
		H int `json:"h"`
	} `json:"min,omitempty"`
}

type ToolDecl struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Mutating    bool   `json:"mutating"`
}

// Installed is a manifest plus install-time state.
type Installed struct {
	Manifest
	Granted []string `json:"granted"` // scopes the user approved (subset of Permissions)
	Enabled bool     `json:"enabled"`
}

var idRe = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9-]+)+$`) // at least one dot, reverse-DNS-ish

type Store struct {
	mu     sync.Mutex
	dir    string // ~/.local/share/ghost/apps
	grants string // grants.json
}

func New() *Store {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".local", "share", "ghost", "apps")
	os.MkdirAll(dir, 0o755)
	return &Store{dir: dir, grants: filepath.Join(dir, "grants.json")}
}

// Dir returns the on-disk directory for an installed app (for static serving).
func (s *Store) Dir(id string) string { return filepath.Join(s.dir, id) }

type grant struct {
	Granted []string `json:"granted"`
	Enabled bool     `json:"enabled"`
}

func (s *Store) loadGrants() map[string]grant {
	m := map[string]grant{}
	if data, err := os.ReadFile(s.grants); err == nil {
		json.Unmarshal(data, &m)
	}
	return m
}

func (s *Store) saveGrants(m map[string]grant) {
	data, _ := json.MarshalIndent(m, "", "  ")
	tmp := s.grants + ".tmp"
	if os.WriteFile(tmp, data, 0o600) == nil {
		os.Rename(tmp, s.grants)
	}
}

// ParseManifest reads and validates the manifest from a .osapp zip's bytes
// without installing — used to drive the permission prompt before the grant.
func ParseManifest(pkg []byte) (Manifest, error) {
	zr, err := zip.NewReader(bytes.NewReader(pkg), int64(len(pkg)))
	if err != nil {
		return Manifest{}, fmt.Errorf("not a valid .osapp zip: %w", err)
	}
	m, _, err := readManifest(zr)
	return m, err
}

// readManifest finds <id>/manifest.json (or manifest.json at root) and returns
// it plus the path prefix all entries must share.
func readManifest(zr *zip.Reader) (Manifest, string, error) {
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "manifest.json") && strings.Count(strings.Trim(f.Name, "/"), "/") <= 1 {
			rc, err := f.Open()
			if err != nil {
				return Manifest{}, "", err
			}
			defer rc.Close()
			data, err := io.ReadAll(io.LimitReader(rc, 1<<20))
			if err != nil {
				return Manifest{}, "", err
			}
			var m Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				return Manifest{}, "", fmt.Errorf("manifest.json: %w", err)
			}
			if err := m.validate(); err != nil {
				return Manifest{}, "", err
			}
			prefix := strings.TrimSuffix(f.Name, "manifest.json") // "" or "<id>/"
			return m, prefix, nil
		}
	}
	return Manifest{}, "", fmt.Errorf("manifest.json not found in package")
}

func (m Manifest) validate() error {
	if !idRe.MatchString(m.ID) {
		return fmt.Errorf("invalid app id %q (want reverse-DNS, e.g. tone.studio)", m.ID)
	}
	if m.Version == "" {
		return fmt.Errorf("manifest missing version")
	}
	if m.Entry == "" {
		return fmt.Errorf("manifest missing entry")
	}
	for _, p := range m.Permissions {
		if !ScopeKnown(p) {
			return fmt.Errorf("unknown permission %q", p)
		}
	}
	return nil
}

// Install verifies the package hash, unpacks it atomically, and records the
// granted scopes. expectedSHA is hex sha256 (from a store index); pass "" to
// skip the check (sideload). granted is the subset of requested permissions the
// user approved.
func (s *Store) Install(pkg []byte, expectedSHA string, granted []string) (Installed, error) {
	if expectedSHA != "" {
		sum := sha256.Sum256(pkg)
		if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, expectedSHA) {
			return Installed{}, fmt.Errorf("package hash mismatch: got %s, want %s", got, expectedSHA)
		}
	}
	zr, err := zip.NewReader(bytes.NewReader(pkg), int64(len(pkg)))
	if err != nil {
		return Installed{}, fmt.Errorf("not a valid .osapp zip: %w", err)
	}
	m, prefix, err := readManifest(zr)
	if err != nil {
		return Installed{}, err
	}
	// Granted must be a subset of the manifest's requested permissions.
	for _, g := range granted {
		if !contains(m.Permissions, g) {
			return Installed{}, fmt.Errorf("granted scope %q was not requested by the app", g)
		}
	}

	final := s.Dir(m.ID)
	incoming := final + ".incoming"
	os.RemoveAll(incoming)
	if err := unpack(zr, prefix, incoming); err != nil {
		os.RemoveAll(incoming)
		return Installed{}, err
	}
	// Entry must exist.
	if _, err := os.Stat(filepath.Join(incoming, filepath.Clean(m.Entry))); err != nil {
		os.RemoveAll(incoming)
		return Installed{}, fmt.Errorf("entry %q missing from package", m.Entry)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	os.RemoveAll(final)
	if err := os.Rename(incoming, final); err != nil {
		os.RemoveAll(incoming)
		return Installed{}, err
	}
	g := s.loadGrants()
	g[m.ID] = grant{Granted: granted, Enabled: true}
	s.saveGrants(g)
	return Installed{Manifest: m, Granted: granted, Enabled: true}, nil
}

// SafeUnzip unpacks every entry of a zip into dst with the same zip-slip guards
// as app install — used by the store for skill/tool packages.
func SafeUnzip(data []byte, dst string) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("not a valid zip: %w", err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	return unpack(zr, "", dst)
}

// unpack writes every zip entry under prefix into dst, guarding against
// zip-slip (.. or absolute paths escaping dst).
func unpack(zr *zip.Reader, prefix, dst string) error {
	for _, f := range zr.File {
		name := f.Name
		if prefix != "" {
			if !strings.HasPrefix(name, prefix) {
				continue // ignore stray top-level entries outside the app dir
			}
			name = strings.TrimPrefix(name, prefix)
		}
		if name == "" || strings.HasSuffix(name, "/") {
			continue // directory entry
		}
		if strings.Contains(name, "\\") || hasDotDot(name) || filepath.IsAbs(name) {
			return fmt.Errorf("unsafe path in package: %q", f.Name)
		}
		rel := filepath.Clean(name)
		if rel == "" || rel == "." {
			return fmt.Errorf("unsafe path in package: %q", f.Name)
		}
		out := filepath.Join(dst, rel)
		if !strings.HasPrefix(out, dst+string(os.PathSeparator)) {
			return fmt.Errorf("path escapes app dir: %q", f.Name)
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(w, io.LimitReader(rc, 64<<20))
		w.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// List returns installed apps merged with their grants.
func (s *Store) List() []Installed {
	s.mu.Lock()
	defer s.mu.Unlock()
	grants := s.loadGrants()
	entries, _ := os.ReadDir(s.dir)
	var out []Installed
	for _, e := range entries {
		if !e.IsDir() || strings.HasSuffix(e.Name(), ".incoming") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.dir, e.Name(), "manifest.json"))
		if err != nil {
			continue
		}
		var m Manifest
		if json.Unmarshal(data, &m) != nil {
			continue
		}
		g := grants[m.ID]
		out = append(out, Installed{Manifest: m, Granted: g.Granted, Enabled: g.Enabled})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Name < out[b].Name })
	return out
}

// Get returns a single installed app.
func (s *Store) Get(id string) (Installed, bool) {
	for _, a := range s.List() {
		if a.ID == id {
			return a, true
		}
	}
	return Installed{}, false
}

// ScopesFor returns the granted scopes for an installed, enabled app.
func (s *Store) ScopesFor(id string) ([]string, bool) {
	a, ok := s.Get(id)
	if !ok || !a.Enabled {
		return nil, false
	}
	return a.Granted, true
}

func (s *Store) Uninstall(id string) error {
	if !idRe.MatchString(id) {
		return fmt.Errorf("invalid id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	os.RemoveAll(s.Dir(id))
	g := s.loadGrants()
	delete(g, id)
	s.saveGrants(g)
	return nil
}

// hasDotDot reports whether any path segment is "..".
func hasDotDot(name string) bool {
	for _, seg := range strings.Split(name, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}
