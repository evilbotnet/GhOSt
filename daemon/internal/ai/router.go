package ai

import (
	"regexp"
	"strconv"
	"strings"
)

// The router (ADR 0002) decides which *kind* of work a request is, with
// deterministic rules — not a model judging the route. Three outcomes:
//
//   - Command tier, rules: the utterance maps to one known OS action. Resolved
//     here with zero LLM — instant and fully offline ("volume 40", "lock").
//   - Command tier, intent model: rules missed and the only configured brain is
//     a local intent model → one constrained single-shot tool call.
//   - Agent tier: anything multi-step → the confirmation-gated tool loop on the
//     configured LAN/cloud provider, with fallback.
//
// Keeping the route deterministic (not model-judged) makes it cheap,
// predictable, and auditable, exactly as the ADR argues.

// commandRule maps a matched utterance to a single tool call.
type commandRule struct {
	re   *regexp.Regexp
	tool string
	args func(m []string) map[string]any
}

// commandRules are the tier-0 deterministic mappings. They target the daemon's
// own tools, so they work with no model configured and never leave the device.
var commandRules = []commandRule{
	{
		regexp.MustCompile(`(?i)^\s*(?:set\s+)?volume\s+(?:to\s+)?(\d{1,3})\s*%?\s*$`),
		"set_volume",
		func(m []string) map[string]any {
			v, _ := strconv.Atoi(m[1])
			if v > 100 {
				v = 100
			}
			return map[string]any{"percent": float64(v)}
		},
	},
	{
		regexp.MustCompile(`(?i)^\s*(?:mute|silence)(?:\s+(?:the\s+)?(?:volume|sound|audio))?\s*$`),
		"set_volume",
		func(m []string) map[string]any { return map[string]any{"percent": float64(0)} },
	},
	{
		regexp.MustCompile(`(?i)^\s*lock(?:\s+(?:the\s+)?screen)?\s*$`),
		"lock_screen",
		func(m []string) map[string]any { return map[string]any{} },
	},
	{
		regexp.MustCompile(`(?i)^\s*(?:open|go\s+to|browse\s+to|visit)\s+(https?://\S+|[\w-]+(?:\.[\w-]+)+\S*)\s*$`),
		"open_browser",
		func(m []string) map[string]any { return map[string]any{"url": m[1]} },
	},
	{
		regexp.MustCompile(`(?i)^\s*(?:system\s+)?status\b.*$`),
		"system_status",
		func(m []string) map[string]any { return map[string]any{} },
	},
}

// matchCommand returns the tool call for a tier-0 rule match, if any.
func matchCommand(prompt string) (tool string, args map[string]any, ok bool) {
	for _, r := range commandRules {
		if m := r.re.FindStringSubmatch(prompt); m != nil {
			return r.tool, r.args(m), true
		}
	}
	return "", nil, false
}

var askRe = regexp.MustCompile(`(?i)^\s*ask\s+([\w-]+)[\s:,]+(.+)$`)

// parseOverride extracts an explicit "ask <provider> <request>" pin. Returns the
// provider name (lowercased) and the remaining request, or "" if no override.
func parseOverride(prompt string) (provider, rest string) {
	if m := askRe.FindStringSubmatch(prompt); m != nil {
		return strings.ToLower(m[1]), strings.TrimSpace(m[2])
	}
	return "", prompt
}
