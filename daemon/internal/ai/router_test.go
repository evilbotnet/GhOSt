package ai

import "testing"

func TestMatchCommand(t *testing.T) {
	cases := []struct {
		in      string
		tool    string
		wantArg map[string]any
		ok      bool
	}{
		{"volume 40", "set_volume", map[string]any{"percent": float64(40)}, true},
		{"set volume to 75%", "set_volume", map[string]any{"percent": float64(75)}, true},
		{"VOLUME 200", "set_volume", map[string]any{"percent": float64(100)}, true}, // clamped
		{"mute", "set_volume", map[string]any{"percent": float64(0)}, true},
		{"lock", "lock_screen", map[string]any{}, true},
		{"lock the screen", "lock_screen", map[string]any{}, true},
		{"open https://example.com", "open_browser", map[string]any{"url": "https://example.com"}, true},
		{"go to news.ycombinator.com", "open_browser", map[string]any{"url": "news.ycombinator.com"}, true},
		{"status", "system_status", map[string]any{}, true},
		// Multi-step / ambiguous → no rule (must go to the agent tier).
		{"organize my downloads into folders", "", nil, false},
		{"what's the weather and set volume to 30", "", nil, false},
		{"open the file where I wrote about the pi", "", nil, false},
	}
	for _, c := range cases {
		tool, args, ok := matchCommand(c.in)
		if ok != c.ok {
			t.Errorf("%q: ok=%v want %v", c.in, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if tool != c.tool {
			t.Errorf("%q: tool=%q want %q", c.in, tool, c.tool)
		}
		for k, v := range c.wantArg {
			if args[k] != v {
				t.Errorf("%q: arg[%s]=%v want %v", c.in, k, args[k], v)
			}
		}
	}
}

func TestParseOverride(t *testing.T) {
	cases := []struct {
		in, prov, rest string
	}{
		{"ask cloud summarize this", "cloud", "summarize this"},
		{"ask local: volume 40", "local", "volume 40"},
		{"ask LAN, tidy downloads", "lan", "tidy downloads"},
		{"just a normal request", "", "just a normal request"},
		{"asked nicely about things", "", "asked nicely about things"}, // not an override
	}
	for _, c := range cases {
		prov, rest := parseOverride(c.in)
		if prov != c.prov || rest != c.rest {
			t.Errorf("%q: got (%q,%q) want (%q,%q)", c.in, prov, rest, c.prov, c.rest)
		}
	}
}
