// Package setup backs the first-boot wizard: setup state, timezone catalog,
// and writing the Ghost AI routing config (ADR 0002).
package setup

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Manager struct {
	configDir string // ~/.config/ghost
}

func New() *Manager {
	home, _ := os.UserHomeDir()
	return &Manager{configDir: filepath.Join(home, ".config", "ghost")}
}

func (m *Manager) doneFlag() string { return filepath.Join(m.configDir, "setup-done") }

func (m *Manager) Needed() bool {
	_, err := os.Stat(m.doneFlag())
	return os.IsNotExist(err)
}

func (m *Manager) Complete() error {
	if err := os.MkdirAll(m.configDir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(m.doneFlag(), []byte("setup completed\n"), 0o600)
}

// Timezones returns the system list, with a small fallback for dev hosts.
func (m *Manager) Timezones() []string {
	out, err := exec.Command("timedatectl", "list-timezones").Output()
	if err != nil {
		return []string{
			"UTC", "America/New_York", "America/Chicago", "America/Denver",
			"America/Los_Angeles", "Europe/London", "Europe/Berlin",
			"Asia/Tokyo", "Australia/Sydney",
		}
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

// AIConfig is the wizard's view of Ghost routing (ADR 0002).
type AIConfig struct {
	Mode  string `json:"mode"` // off | local | lan | cloud
	URL   string `json:"url"`  // lan/local OpenAI-compatible endpoint
	Model string `json:"model"`
	Key   string `json:"key"` // cloud API key (stored 0600, never echoed)
}

// SaveAI writes ~/.config/ghost/ai.toml (+ key file). Ghost (Phase 7) reads
// this; saving it at first boot means the router is configured before the
// assistant ships.
func (m *Manager) SaveAI(c AIConfig) error {
	if err := os.MkdirAll(m.configDir, 0o700); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("# Ghost AI routing — see docs/decisions/0002-ghost-ai-assistant.md\n")
	fmt.Fprintf(&b, "[ai]\nenabled = %v\n\n", c.Mode != "off" && c.Mode != "")
	switch c.Mode {
	case "local", "lan":
		name := c.Mode
		fmt.Fprintf(&b, "[ai.providers.%s]\ntype = \"openai-compatible\"\nurl = %q\nmodel = %q\n\n", name, c.URL, c.Model)
		fmt.Fprintf(&b, "[ai.routing]\nintent = %q\nagent = %q\nfallback = \"\"\n", name, name)
	case "cloud":
		model := c.Model
		if model == "" {
			model = "claude-opus-4-8"
		}
		fmt.Fprintf(&b, "[ai.providers.cloud]\ntype = \"anthropic\"\nmodel = %q\nkey_file = \"~/.config/ghost/anthropic.key\"\n\n", model)
		b.WriteString("[ai.routing]\nintent = \"cloud\"\nagent = \"cloud\"\nfallback = \"\"\n")
		if c.Key != "" {
			if err := os.WriteFile(filepath.Join(m.configDir, "anthropic.key"), []byte(c.Key+"\n"), 0o600); err != nil {
				return err
			}
		}
	default:
		b.WriteString("[ai.routing]\nintent = \"\"\nagent = \"\"\nfallback = \"\"\n")
	}
	return os.WriteFile(filepath.Join(m.configDir, "ai.toml"), []byte(b.String()), 0o600)
}
