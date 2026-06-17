package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// External tools extend Ghost without recompiling the daemon: a JSON manifest
// (<name>.tool.json) describes the tool's name, description, input schema, and
// an executable to run. Ghost calls it like any built-in; the daemon execs the
// command with the JSON args on stdin and returns stdout. Mutating tools are
// confirmation-gated like every other mutating action.
//
// This is the "additional tools" half of Ghost extensibility (ADR 0005);
// skills are the "expertise" half.

type extToolManifest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Mutating    bool     `json:"mutating"`
	Command     []string `json:"command"` // argv; cwd = manifest dir
	TimeoutSec  int      `json:"timeoutSec"`
	InputSchema struct {
		Properties map[string]any `json:"properties"`
		Required   []string       `json:"required"`
	} `json:"inputSchema"`
}

// ExtToolInfo is the listing shape for the shell.
type ExtToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Mutating    bool   `json:"mutating"`
}

func ToolsDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ghost", "tools")
}

func systemToolsDir() string { return "/usr/share/ghost/tools" }

func loadManifests() []struct {
	m   extToolManifest
	dir string
} {
	type entry struct {
		m   extToolManifest
		dir string
	}
	byName := map[string]entry{}
	for _, dir := range []string{systemToolsDir(), ToolsDir()} {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.tool.json"))
		for _, path := range matches {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var m extToolManifest
			if json.Unmarshal(data, &m) != nil || m.Name == "" || len(m.Command) == 0 {
				continue
			}
			byName[m.Name] = entry{m: m, dir: dir}
		}
	}
	out := make([]entry, 0, len(byName))
	for _, e := range byName {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].m.Name < out[j].m.Name })
	res := make([]struct {
		m   extToolManifest
		dir string
	}, len(out))
	for i, e := range out {
		res[i] = struct {
			m   extToolManifest
			dir string
		}{e.m, e.dir}
	}
	return res
}

// ExtTools lists installed external tools (for the shell).
func ExtTools() []ExtToolInfo {
	var out []ExtToolInfo
	for _, e := range loadManifests() {
		out = append(out, ExtToolInfo{Name: e.m.Name, Description: e.m.Description, Mutating: e.m.Mutating})
	}
	return out
}

// extTools builds runnable tool entries for the agent loop.
func extTools() map[string]tool {
	tools := map[string]tool{}
	for _, e := range loadManifests() {
		m, dir := e.m, e.dir
		tools[m.Name] = tool{
			def: ToolDef{
				Name:        m.Name,
				Description: m.Description,
				Properties:  m.InputSchema.Properties,
				Required:    m.InputSchema.Required,
			},
			mutating: m.Mutating,
			run: func(args map[string]any) (string, error) {
				return runExtTool(m, dir, args)
			},
		}
	}
	return tools
}

// envKey uppercases a tool arg name and replaces non-alphanumerics with _.
func envKey(k string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(k) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}

func runExtTool(m extToolManifest, dir string, args map[string]any) (string, error) {
	timeout := time.Duration(m.TimeoutSec) * time.Second
	if timeout <= 0 || timeout > 2*time.Minute {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, m.Command[0], m.Command[1:]...)
	cmd.Dir = dir
	argsJSON, _ := json.Marshal(args)
	cmd.Stdin = bytes.NewReader(argsJSON)
	// Args offered three ways so any tool style is easy: full JSON on stdin,
	// full JSON in GHOST_TOOL_ARGS, and each scalar arg as GHOST_ARG_<KEY>
	// (so a shell script can just read "$GHOST_ARG_TEXT").
	env := append(os.Environ(), "GHOST_TOOL_ARGS="+string(argsJSON))
	for k, v := range args {
		switch v.(type) {
		case string, float64, bool, int, int64:
			env = append(env, "GHOST_ARG_"+envKey(k)+"="+fmt.Sprintf("%v", v))
		}
	}
	cmd.Env = env

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	result := strings.TrimSpace(out.String())
	if len(result) > 16000 {
		result = result[:16000] + "\n…(truncated)"
	}
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("tool timed out after %s", timeout)
	}
	if err != nil {
		if result != "" {
			return "", fmt.Errorf("%s", result)
		}
		return "", fmt.Errorf("tool failed: %w", err)
	}
	if result == "" {
		result = "(no output)"
	}
	return result, nil
}
