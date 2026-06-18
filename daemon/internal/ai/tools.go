package ai

import (
	"fmt"
	"strings"

	"github.com/ghostos/ghostd/internal/browser"
	"github.com/ghostos/ghostd/internal/fsops"
	"github.com/ghostos/ghostd/internal/gpio"
	"github.com/ghostos/ghostd/internal/system"
)

// Toolbox is Ghost's hands: the daemon's own subsystems exposed as LLM tools.
// `mutating` tools are confirmation-gated in the shell before they run.
type Toolbox struct {
	fs      *fsops.FS
	sys     *system.System
	browser *browser.Browser
	gpio    *gpio.GPIO
}

func NewToolbox(fs *fsops.FS, sys *system.System, br *browser.Browser) *Toolbox {
	return &Toolbox{fs: fs, sys: sys, browser: br, gpio: gpio.New()}
}

func intArg(args map[string]any, k string) int {
	switch v := args[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

type tool struct {
	def      ToolDef
	mutating bool
	run      func(args map[string]any) (string, error)
}

func str(args map[string]any, k string) string {
	if v, ok := args[k].(string); ok {
		return v
	}
	return ""
}

func (tb *Toolbox) tools() map[string]tool {
	return map[string]tool{
		"list_files": {
			def: ToolDef{
				Name:        "list_files",
				Description: "List the contents of a directory in the user's home. Use to explore files before acting.",
				Properties:  map[string]any{"path": map[string]any{"type": "string", "description": "Absolute path or ~ for home"}},
				Required:    []string{"path"},
			},
			run: func(a map[string]any) (string, error) {
				path, entries, err := tb.fs.List(str(a, "path"))
				if err != nil {
					return "", err
				}
				out := fmt.Sprintf("%s:\n", path)
				for _, e := range entries {
					kind := "file"
					if e.Dir {
						kind = "dir "
					}
					out += fmt.Sprintf("  [%s] %s\n", kind, e.Name)
				}
				return out, nil
			},
		},
		"read_file": {
			def: ToolDef{
				Name:        "read_file",
				Description: "Read a text file's contents.",
				Properties:  map[string]any{"path": map[string]any{"type": "string"}},
				Required:    []string{"path"},
			},
			run: func(a map[string]any) (string, error) {
				data, err := tb.fs.Read(str(a, "path"))
				if err != nil {
					return "", err
				}
				return string(data), nil
			},
		},
		"write_file": {
			mutating: true,
			def: ToolDef{
				Name:        "write_file",
				Description: "Create or overwrite a text file. Mutating — the user must confirm.",
				Properties: map[string]any{
					"path":    map[string]any{"type": "string"},
					"content": map[string]any{"type": "string"},
				},
				Required: []string{"path", "content"},
			},
			run: func(a map[string]any) (string, error) {
				if err := tb.fs.Write(str(a, "path"), []byte(str(a, "content"))); err != nil {
					return "", err
				}
				return "wrote " + str(a, "path"), nil
			},
		},
		"move_file": {
			mutating: true,
			def: ToolDef{
				Name:        "move_file",
				Description: "Move or rename a file. Mutating — the user must confirm.",
				Properties: map[string]any{
					"from": map[string]any{"type": "string"},
					"to":   map[string]any{"type": "string"},
				},
				Required: []string{"from", "to"},
			},
			run: func(a map[string]any) (string, error) {
				if err := tb.fs.Rename(str(a, "from"), str(a, "to")); err != nil {
					return "", err
				}
				return fmt.Sprintf("moved %s -> %s", str(a, "from"), str(a, "to")), nil
			},
		},
		"trash_file": {
			mutating: true,
			def: ToolDef{
				Name:        "trash_file",
				Description: "Move a file to the trash. Mutating — the user must confirm.",
				Properties:  map[string]any{"path": map[string]any{"type": "string"}},
				Required:    []string{"path"},
			},
			run: func(a map[string]any) (string, error) {
				if err := tb.fs.Trash(str(a, "path")); err != nil {
					return "", err
				}
				return "trashed " + str(a, "path"), nil
			},
		},
		"make_dir": {
			mutating: true,
			def: ToolDef{
				Name:        "make_dir",
				Description: "Create a directory. Mutating — the user must confirm.",
				Properties:  map[string]any{"path": map[string]any{"type": "string"}},
				Required:    []string{"path"},
			},
			run: func(a map[string]any) (string, error) {
				if err := tb.fs.Mkdir(str(a, "path")); err != nil {
					return "", err
				}
				return "created " + str(a, "path"), nil
			},
		},
		"open_browser": {
			mutating: true,
			def: ToolDef{
				Name:        "open_browser",
				Description: "Open a website in a new browser window. Mutating — the user must confirm.",
				Properties:  map[string]any{"url": map[string]any{"type": "string"}},
				Required:    []string{"url"},
			},
			run: func(a map[string]any) (string, error) {
				if err := tb.browser.Open(str(a, "url")); err != nil {
					return "", err
				}
				return "opened " + str(a, "url"), nil
			},
		},
		"set_volume": {
			mutating: true,
			def: ToolDef{
				Name:        "set_volume",
				Description: "Set the output volume (0-100). Mutating — the user must confirm.",
				Properties:  map[string]any{"percent": map[string]any{"type": "integer"}},
				Required:    []string{"percent"},
			},
			run: func(a map[string]any) (string, error) {
				pct := 0
				if f, ok := a["percent"].(float64); ok {
					pct = int(f)
				}
				if err := tb.sys.SetVolume(pct); err != nil {
					return "", err
				}
				return fmt.Sprintf("volume set to %d%%", pct), nil
			},
		},
		"system_status": {
			def: ToolDef{
				Name:        "system_status",
				Description: "Get hostname, platform, Wi-Fi, battery, and volume status.",
				Properties:  map[string]any{},
			},
			run: func(a map[string]any) (string, error) {
				s := tb.sys.Status()
				return fmt.Sprintf("host=%s platform=%s wifi=%v(%s) battery=%d%% volume=%d%%",
					s.Hostname, s.Platform, s.Wifi.Connected, s.Wifi.SSID, s.Battery.Percent, s.Volume.Percent), nil
			},
		},
		"gpio_list": {
			def: ToolDef{
				Name:        "gpio_list",
				Description: "List the board's GPIO lines (BCM offset, name, and whether each is driven as an output) so you can pick a pin. Read-only.",
				Properties:  map[string]any{},
			},
			run: func(a map[string]any) (string, error) {
				if !tb.gpio.Available() {
					return "no GPIO on this board", nil
				}
				lines, err := tb.gpio.Lines()
				if err != nil {
					return "", err
				}
				var b strings.Builder
				for _, l := range lines {
					state := "input"
					if l.Output {
						state = fmt.Sprintf("output=%d", l.Value)
					}
					fmt.Fprintf(&b, "GPIO%d %s %s\n", l.Offset, l.Name, state)
				}
				return b.String(), nil
			},
		},
		"gpio_read": {
			def: ToolDef{
				Name:        "gpio_read",
				Description: "Read the level (0 or 1) of a GPIO pin by BCM number. Read-only — use for buttons/sensors.",
				Properties:  map[string]any{"pin": map[string]any{"type": "integer", "description": "BCM pin number"}},
				Required:    []string{"pin"},
			},
			run: func(a map[string]any) (string, error) {
				v, err := tb.gpio.Read(intArg(a, "pin"))
				if err != nil {
					return "", err
				}
				return fmt.Sprintf("GPIO%d = %d", intArg(a, "pin"), v), nil
			},
		},
		"gpio_set": {
			mutating: true,
			def: ToolDef{
				Name:        "gpio_set",
				Description: "Drive a GPIO output pin high (1) or low (0); it stays asserted until changed. Mutating — the user must confirm. Use for LEDs/relays; blink by setting, waiting, and setting again.",
				Properties: map[string]any{
					"pin":   map[string]any{"type": "integer", "description": "BCM pin number"},
					"value": map[string]any{"type": "integer", "description": "1 = high, 0 = low"},
				},
				Required: []string{"pin", "value"},
			},
			run: func(a map[string]any) (string, error) {
				pin, val := intArg(a, "pin"), intArg(a, "value")
				if err := tb.gpio.Set(pin, val); err != nil {
					return "", err
				}
				return fmt.Sprintf("GPIO%d driven %d", pin, val), nil
			},
		},
	}
}

func (tb *Toolbox) defs() []ToolDef {
	var out []ToolDef
	for _, t := range tb.tools() {
		out = append(out, t.def)
	}
	return out
}
