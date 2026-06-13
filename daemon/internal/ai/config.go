// Package ai is Ghost — the resident assistant whose tool surface is the OS
// API itself (ADR 0002). The daemon hosts the agent loop; every mutating
// tool call is confirmation-gated through the shell.
package ai

import (
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
