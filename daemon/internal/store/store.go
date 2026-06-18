// Package store is the GhOSt registry client (ADR 0009): it fetches a signed
// index from a git-backed store, verifies its Ed25519 signature against a
// pinned publisher key, and installs the catalog's apps, skills, tools, and
// MCP servers. No GhOSt-operated backend — the index is just data in a repo,
// mirrorable and forkable; trust rides the signature plus a per-artifact hash.
package store

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ghostos/ghostd/internal/ai"
	"github.com/ghostos/ghostd/internal/osapp"
)

// Entry is one catalog item. Type-specific fields are populated per Type.
type Entry struct {
	Type        string   `json:"type"` // app | skill | tool | mcp
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Icon        string   `json:"icon,omitempty"`
	Author      string   `json:"author,omitempty"`

	// app | skill | tool: a downloadable package, hash-pinned.
	Download string `json:"download,omitempty"`
	SHA256   string `json:"sha256,omitempty"`

	// app: the permissions the package will request (shown before install).
	Permissions []string `json:"permissions,omitempty"`

	// mcp: how to launch the server (no download — config only).
	Transport string   `json:"transport,omitempty"`
	Command   []string `json:"command,omitempty"`
	URL       string   `json:"url,omitempty"`
}

// Index is the signed catalog.
type Index struct {
	Generated string  `json:"generated"`
	Entries   []Entry `json:"entries"`
}

// Config points at a store and pins the key its index must be signed with.
type Config struct {
	IndexURL  string `json:"indexURL"`  // https raw URL or local path to index.json
	PublicKey string `json:"publicKey"` // base64 Ed25519 public key
}

type Store struct {
	osapps *osapp.Store
	mu     sync.Mutex
	cfg    Config
}

func New(osapps *osapp.Store) *Store {
	s := &Store{osapps: osapps}
	s.cfg = loadConfig()
	return s
}

func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ghost", "store.json")
}

func loadConfig() Config {
	var c Config
	if data, err := os.ReadFile(configPath()); err == nil {
		json.Unmarshal(data, &c)
	}
	return c
}

// Configure sets and persists the store URL + pinned key.
func (s *Store) Configure(c Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	os.MkdirAll(filepath.Dir(configPath()), 0o755)
	data, _ := json.MarshalIndent(c, "", "  ")
	if err := os.WriteFile(configPath(), data, 0o600); err != nil {
		return err
	}
	s.cfg = c
	return nil
}

func (s *Store) Config() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

// Catalog fetches the index, verifies its signature, and returns the entries.
func (s *Store) Catalog() (Index, error) {
	cfg := s.Config()
	if cfg.IndexURL == "" || cfg.PublicKey == "" {
		return Index{}, fmt.Errorf("no store configured — set a URL and pinned key in the Hub")
	}
	pub, err := base64.StdEncoding.DecodeString(strings.TrimSpace(cfg.PublicKey))
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return Index{}, fmt.Errorf("store public key is invalid")
	}
	indexBytes, err := fetch(cfg.IndexURL)
	if err != nil {
		return Index{}, fmt.Errorf("fetch index: %w", err)
	}
	sigBytes, err := fetch(cfg.IndexURL + ".sig")
	if err != nil {
		return Index{}, fmt.Errorf("fetch signature: %w", err)
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(sigBytes)))
	if err != nil {
		return Index{}, fmt.Errorf("signature is not valid base64")
	}
	if !ed25519.Verify(pub, indexBytes, sig) {
		return Index{}, fmt.Errorf("index signature does not match the pinned key — refusing to trust it")
	}
	var idx Index
	if err := json.Unmarshal(indexBytes, &idx); err != nil {
		return Index{}, fmt.Errorf("index is not valid JSON: %w", err)
	}
	return idx, nil
}

// Install installs the catalog entry with the given id (re-fetching + verifying
// the index first, so we only ever act on signed data). granted is the scope
// subset the user approved (apps only).
func (s *Store) Install(id string, granted []string) error {
	idx, err := s.Catalog()
	if err != nil {
		return err
	}
	var e *Entry
	for i := range idx.Entries {
		if idx.Entries[i].ID == id {
			e = &idx.Entries[i]
			break
		}
	}
	if e == nil {
		return fmt.Errorf("no catalog entry %q", id)
	}

	switch e.Type {
	case "app":
		pkg, err := s.download(e)
		if err != nil {
			return err
		}
		_, err = s.osapps.Install(pkg, e.SHA256, granted)
		return err
	case "skill":
		return s.unpackInto(e, "skills")
	case "tool":
		return s.unpackInto(e, "tools")
	case "mcp":
		return ai.AddMCPServer(e.ID, e.Transport, e.Command, e.URL)
	default:
		return fmt.Errorf("unknown entry type %q", e.Type)
	}
}

// download fetches a package and verifies its hash against the (already
// signature-verified) index entry.
func (s *Store) download(e *Entry) ([]byte, error) {
	if e.Download == "" {
		return nil, fmt.Errorf("entry %q has no download", e.ID)
	}
	pkg, err := fetch(e.Download)
	if err != nil {
		return nil, err
	}
	if e.SHA256 != "" {
		sum := sha256.Sum256(pkg)
		if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, e.SHA256) {
			return nil, fmt.Errorf("package hash mismatch for %q", e.ID)
		}
	}
	return pkg, nil
}

func (s *Store) unpackInto(e *Entry, subdir string) error {
	pkg, err := s.download(e)
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	return osapp.SafeUnzip(pkg, filepath.Join(home, ".config", "ghost", subdir))
}

// fetch reads bytes from an https URL or a local file path.
func fetch(loc string) ([]byte, error) {
	if strings.HasPrefix(loc, "http://") || strings.HasPrefix(loc, "https://") {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(loc)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GET %s: %s", loc, resp.Status)
		}
		return io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	}
	return os.ReadFile(strings.TrimPrefix(loc, "file://"))
}
