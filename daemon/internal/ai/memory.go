package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Cross-session memory: facts Ghost carries between conversations — the user's
// preferences and durable context. Each memory is a markdown file in
// ~/.config/ghost/memory/<name>.md (frontmatter: description; body: the fact).
//
// Unlike skills (progressively disclosed via a tool), memories are short and
// always relevant, so the full set is injected straight into the system prompt
// — a "recall" tool the model might forget to call would defeat the point.
// Writing/forgetting goes through `remember`/`forget`, which are mutating and
// therefore confirmation-gated: nothing is remembered without the user's OK.

type Memory struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Body        string `json:"body"`
}

func MemoryDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ghost", "memory")
}

var memSlug = regexp.MustCompile(`[^a-z0-9]+`)

// slugifyMemory makes a safe, stable filename stem from a memory name.
func slugifyMemory(name string) string {
	s := memSlug.ReplaceAllString(strings.ToLower(strings.TrimSpace(name)), "-")
	return strings.Trim(s, "-")
}

func LoadMemories() []Memory {
	entries, err := os.ReadDir(MemoryDir())
	if err != nil {
		return nil
	}
	var mems []Memory
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(MemoryDir(), e.Name()))
		if err != nil {
			continue
		}
		name, desc, body := parseFrontmatter(string(data))
		if name == "" {
			name = strings.TrimSuffix(e.Name(), ".md")
		}
		body = strings.TrimSpace(body)
		if body == "" {
			continue
		}
		mems = append(mems, Memory{Name: name, Description: desc, Body: body})
	}
	sort.Slice(mems, func(i, j int) bool { return mems[i].Name < mems[j].Name })
	return mems
}

// SaveMemory writes (or overwrites) a memory. Returns the stored memory.
func SaveMemory(name, description, body string) (Memory, error) {
	slug := slugifyMemory(name)
	if slug == "" {
		return Memory{}, fmt.Errorf("a memory needs a name")
	}
	if strings.TrimSpace(body) == "" {
		return Memory{}, fmt.Errorf("a memory needs content")
	}
	if err := os.MkdirAll(MemoryDir(), 0o755); err != nil {
		return Memory{}, err
	}
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: " + name + "\n")
	if description != "" {
		b.WriteString("description: " + description + "\n")
	}
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimSpace(body) + "\n")
	path := filepath.Join(MemoryDir(), slug+".md")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return Memory{}, err
	}
	return Memory{Name: name, Description: description, Body: strings.TrimSpace(body)}, nil
}

func DeleteMemory(name string) error {
	slug := slugifyMemory(name)
	if slug == "" {
		return fmt.Errorf("invalid name")
	}
	err := os.Remove(filepath.Join(MemoryDir(), slug+".md"))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// memoryPromptSection injects the full memory set into the system prompt.
func memoryPromptSection(mems []Memory) string {
	if len(mems) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nWhat you remember about this user and their setup (persisted across sessions — use it naturally; update it with the remember tool when you learn something durable, and only when it'll matter later):\n")
	for _, m := range mems {
		b.WriteString("- " + m.Name + ": " + m.Body + "\n")
	}
	return b.String()
}

// memoryTools returns the remember/forget tools (both mutating → gated).
func memoryTools() map[string]tool {
	return map[string]tool{
		"remember": {
			mutating: true,
			def: ToolDef{
				Name:        "remember",
				Description: "Persist a durable fact about the user or their setup across sessions (a preference, their name, ongoing context). Mutating — the user confirms. Use a short stable name; calling it again with the same name updates that memory. Only remember things that will matter later, not one-off conversation details.",
				Properties: map[string]any{
					"name":        map[string]any{"type": "string", "description": "short stable name, e.g. 'preferred-editor'"},
					"content":     map[string]any{"type": "string", "description": "the fact to remember, concise"},
					"description": map[string]any{"type": "string", "description": "optional one-line summary"},
				},
				Required: []string{"name", "content"},
			},
			run: func(a map[string]any) (string, error) {
				m, err := SaveMemory(str(a, "name"), str(a, "description"), str(a, "content"))
				if err != nil {
					return "", err
				}
				return "remembered: " + m.Name, nil
			},
		},
		"forget": {
			mutating: true,
			def: ToolDef{
				Name:        "forget",
				Description: "Delete a remembered fact by name. Mutating — the user confirms.",
				Properties:  map[string]any{"name": map[string]any{"type": "string", "description": "the memory's name"}},
				Required:    []string{"name"},
			},
			run: func(a map[string]any) (string, error) {
				if err := DeleteMemory(str(a, "name")); err != nil {
					return "", err
				}
				return "forgot: " + str(a, "name"), nil
			},
		},
	}
}
