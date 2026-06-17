package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The SOUL is Ghost's personality — a name and a persona, written by the user
// when they "hatch" the assistant during onboarding (à la Hermes / OpenClaw's
// SOUL.md). It is injected at the top of the system prompt so the same OS
// assistant can be calm and terse, or warm and chatty, or whatever the user
// breathed into it. Stored at ~/.config/ghost/SOUL.md with simple frontmatter:
//
//   ---
//   name: Hermes
//   ---
//   You are witty and quick, a clever companion who...
//
// The body is free-form markdown — the persona.

type Soul struct {
	Name string `json:"name"`
	Body string `json:"body"`
}

const defaultSoulName = "Ghost"

func SoulPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ghost", "SOUL.md")
}

// LoadSoul reads SOUL.md. A missing file yields the default unhatched Ghost.
func LoadSoul() Soul {
	data, err := os.ReadFile(SoulPath())
	if err != nil {
		return Soul{Name: defaultSoulName}
	}
	name, _, body := parseFrontmatter(string(data))
	if name == "" {
		name = defaultSoulName
	}
	return Soul{Name: name, Body: strings.TrimSpace(body)}
}

// Hatched reports whether the user has given Ghost a personality yet.
func (s Soul) Hatched() bool {
	return strings.TrimSpace(s.Body) != "" || (s.Name != "" && s.Name != defaultSoulName)
}

func SaveSoul(name, body string) error {
	if strings.TrimSpace(name) == "" {
		name = defaultSoulName
	}
	dir := filepath.Dir(SoulPath())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	content := fmt.Sprintf("---\nname: %s\n---\n\n%s\n", name, strings.TrimSpace(body))
	return os.WriteFile(SoulPath(), []byte(content), 0o600)
}

// soulIdentity is the opening line of the system prompt: who the assistant is,
// plus its persona if one was hatched.
func soulIdentity(s Soul) string {
	name := s.Name
	if name == "" {
		name = defaultSoulName
	}
	var b strings.Builder
	fmt.Fprintf(&b, "You are %s, the resident assistant inside GhOSt, a web-native operating system.\n", name)
	if body := strings.TrimSpace(s.Body); body != "" {
		b.WriteString("\nYour personality — embody it consistently in how you speak and act:\n")
		b.WriteString(body + "\n")
	}
	return b.String()
}
