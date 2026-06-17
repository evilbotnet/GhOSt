package ai

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Skills are drop-in expertise for Ghost — a directory with a SKILL.md
// (YAML frontmatter: name, description; markdown body of instructions).
// Progressive disclosure: only the name+description sit in the system prompt;
// Ghost calls the load_skill tool to pull the full body when a task matches.
// This mirrors Anthropic Skills and keeps the base prompt cheap.

type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Dir         string `json:"-"`
	body        string
}

// SkillsDir is also seeded with defaults from /usr/share/ghost/skills by the
// image; user skills here override/extend them.
func SkillsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ghost", "skills")
}

func systemSkillsDir() string { return "/usr/share/ghost/skills" }

// LoadSkills scans both the system and user skill directories (user wins on
// name collision).
func LoadSkills() []Skill {
	byName := map[string]Skill{}
	for _, dir := range []string{systemSkillsDir(), SkillsDir()} {
		for _, sk := range scanSkills(dir) {
			byName[sk.Name] = sk
		}
	}
	out := make([]Skill, 0, len(byName))
	for _, sk := range byName {
		out = append(out, sk)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func scanSkills(dir string) []Skill {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var skills []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		name, desc, body := parseFrontmatter(string(data))
		if name == "" {
			name = e.Name()
		}
		if desc == "" {
			continue // a skill with no description is unusable to the model
		}
		skills = append(skills, Skill{Name: name, Description: desc, Dir: filepath.Join(dir, e.Name()), body: body})
	}
	return skills
}

// parseFrontmatter extracts name/description from a leading --- YAML block and
// returns the markdown body after it. Minimal by design (no YAML dependency).
func parseFrontmatter(text string) (name, desc, body string) {
	if !strings.HasPrefix(text, "---") {
		return "", "", text
	}
	rest := strings.TrimPrefix(text, "---")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", "", text
	}
	front := rest[:end]
	body = strings.TrimLeft(rest[end+4:], "\r\n")
	for _, line := range strings.Split(front, "\n") {
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(strings.Trim(strings.TrimSpace(v), `"'`))
		switch strings.TrimSpace(k) {
		case "name":
			name = v
		case "description":
			desc = v
		}
	}
	return name, desc, body
}

// skillsPromptSection lists available skills for the system prompt.
func skillsPromptSection(skills []Skill) string {
	if len(skills) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nAvailable skills — call load_skill with the skill name to read full step-by-step instructions before attempting one of these:\n")
	for _, sk := range skills {
		b.WriteString("- " + sk.Name + ": " + sk.Description + "\n")
	}
	return b.String()
}

// loadSkillTool returns the read-only tool Ghost uses to pull a skill body.
func loadSkillTool(skills []Skill) tool {
	index := map[string]Skill{}
	for _, sk := range skills {
		index[sk.Name] = sk
	}
	return tool{
		def: ToolDef{
			Name:        "load_skill",
			Description: "Read the full instructions for one of the available skills, by name. Call this before doing a task that matches a skill's description.",
			Properties:  map[string]any{"name": map[string]any{"type": "string", "description": "The skill name"}},
			Required:    []string{"name"},
		},
		run: func(args map[string]any) (string, error) {
			sk, ok := index[str(args, "name")]
			if !ok {
				return "no such skill; available: " + skillNames(skills), nil
			}
			body := sk.body
			if sk.Dir != "" {
				body += "\n\n(Skill files are in " + sk.Dir + " — read or run them with your other tools as the instructions direct.)"
			}
			return body, nil
		},
	}
}

func skillNames(skills []Skill) string {
	var names []string
	for _, sk := range skills {
		names = append(names, sk.Name)
	}
	return strings.Join(names, ", ")
}
