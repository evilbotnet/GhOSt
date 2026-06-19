// Package ai is Ghost — the resident assistant whose tool surface is the OS
// API itself (ADR 0002). The daemon hosts the agent loop; every mutating
// tool call is confirmation-gated through the shell.
package ai

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Provider struct {
	Type    string `toml:"type"` // "openai-compatible" | "anthropic"
	URL     string `toml:"url"`
	Model   string `toml:"model"`
	KeyFile string `toml:"key_file"`
}

type Routing struct {
	Intent   string `toml:"intent"`
	Agent    string `toml:"agent"`
	Fallback string `toml:"fallback"`
}

type Config struct {
	Enabled   bool
	Providers map[string]Provider
	Routing   Routing
}

type configFile struct {
	AI struct {
		Enabled   bool                `toml:"enabled"`
		Providers map[string]Provider `toml:"providers"`
		Routing   Routing             `toml:"routing"`
	} `toml:"ai"`
}

func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ghost", "ai.toml")
}

// LoadConfig re-reads ai.toml on every call so wizard/Settings edits apply
// without a daemon restart.
func LoadConfig() Config {
	var f configFile
	if _, err := toml.DecodeFile(ConfigPath(), &f); err != nil {
		return Config{}
	}
	return Config{Enabled: f.AI.Enabled, Providers: f.AI.Providers, Routing: f.AI.Routing}
}

// AgentProvider resolves the routing.agent provider (the tier that runs the
// multi-step tool loop — see ADR 0002).
func (c Config) AgentProvider() (name string, p Provider, ok bool) {
	if !c.Enabled || c.Routing.Agent == "" {
		return "", Provider{}, false
	}
	p, ok = c.Providers[c.Routing.Agent]
	return c.Routing.Agent, p, ok
}

// IntentProvider resolves the routing.intent provider — the local command tier
// that turns one utterance into one tool call (ADR 0002).
func (c Config) IntentProvider() (name string, p Provider, ok bool) {
	if !c.Enabled || c.Routing.Intent == "" {
		return "", Provider{}, false
	}
	p, ok = c.Providers[c.Routing.Intent]
	return c.Routing.Intent, p, ok
}

// FallbackProvider resolves the routing.fallback provider, used when the agent
// tier is unreachable (ADR 0002).
func (c Config) FallbackProvider() (name string, p Provider, ok bool) {
	if !c.Enabled || c.Routing.Fallback == "" {
		return "", Provider{}, false
	}
	p, ok = c.Providers[c.Routing.Fallback]
	return c.Routing.Fallback, p, ok
}

// NamedProvider resolves an explicit provider by name (for "ask <provider>"
// overrides).
func (c Config) NamedProvider(name string) (Provider, bool) {
	if !c.Enabled {
		return Provider{}, false
	}
	p, ok := c.Providers[name]
	return p, ok
}

// fullConfig is the single in-memory shape of ai.toml. All writers go through
// loadFull/saveFull so the wizard, Settings, and the Hub's MCP management never
// clobber each other's sections.
type fullConfig struct {
	AI struct {
		Enabled    bool                `toml:"enabled"`
		Providers  map[string]Provider `toml:"providers,omitempty"`
		Routing    Routing             `toml:"routing"`
		MCPServers []mcpServerConfig   `toml:"mcp_servers,omitempty"`
	} `toml:"ai"`
}

func loadFull() fullConfig {
	var f fullConfig
	toml.DecodeFile(ConfigPath(), &f)
	if f.AI.Providers == nil {
		f.AI.Providers = map[string]Provider{}
	}
	return f
}

func saveFull(f fullConfig) error {
	if err := os.MkdirAll(filepath.Dir(ConfigPath()), 0o700); err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString("# Ghost AI config — managed by ghostd (setup wizard, Settings, Hub)\n")
	buf.WriteString("# see docs/decisions/0002-ghost-ai-assistant.md and 0005-ghost-extensibility.md\n\n")
	if err := toml.NewEncoder(&buf).Encode(f); err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), buf.Bytes(), 0o600)
}

// SaveRouting writes provider + routing config, preserving MCP servers.
func SaveRouting(enabled bool, providers map[string]Provider, routing Routing) error {
	f := loadFull()
	f.AI.Enabled = enabled
	f.AI.Providers = providers
	f.AI.Routing = routing
	return saveFull(f)
}

// AddMCPServer adds or replaces an MCP server, preserving everything else.
func AddMCPServer(name, transport string, command []string, url string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("server name required")
	}
	if transport == "" {
		transport = "stdio"
	}
	srv := mcpServerConfig{Name: name, Transport: transport, Command: command, URL: url, Enabled: true}
	f := loadFull()
	for i, s := range f.AI.MCPServers {
		if s.Name == name {
			f.AI.MCPServers[i] = srv
			return saveFull(f)
		}
	}
	f.AI.MCPServers = append(f.AI.MCPServers, srv)
	return saveFull(f)
}

// RemoveMCPServer drops an MCP server by name, preserving everything else.
func RemoveMCPServer(name string) error {
	f := loadFull()
	out := f.AI.MCPServers[:0]
	for _, s := range f.AI.MCPServers {
		if s.Name != name {
			out = append(out, s)
		}
	}
	f.AI.MCPServers = out
	return saveFull(f)
}

func (p Provider) Key() string {
	if p.KeyFile == "" {
		return ""
	}
	path := p.KeyFile
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[2:])
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
