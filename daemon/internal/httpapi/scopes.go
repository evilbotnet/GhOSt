package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"

	"github.com/ghostos/ghostd/internal/osapp"
)

// Scoped tokens (ADR 0009). The shell boots with the session token, which
// carries the implicit superuser scope "*". Each installed .osapp is served
// with its own token carrying only its granted scopes; every API request is
// checked against the scope its path requires. This is what makes "the auth
// layer enforces permissions" literally true rather than aspirational.

type principal struct {
	appID  string
	scopes []string
}

// tokenReg maps live tokens to principals. The session token is handled
// separately (it's the superuser); this holds per-app tokens.
type tokenReg struct {
	mu       sync.Mutex
	byToken  map[string]principal
	byAppID  map[string]string // appID -> current token (one live token per app)
}

func newTokenReg() *tokenReg {
	return &tokenReg{byToken: map[string]principal{}, byAppID: map[string]string{}}
}

// tokenForApp returns a stable token for an app, minting one (with the given
// scopes) on first use and refreshing its scopes each call so re-grants apply.
func (t *tokenReg) tokenForApp(appID string, scopes []string) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	tok := t.byAppID[appID]
	if tok == "" {
		b := make([]byte, 32)
		rand.Read(b)
		tok = hex.EncodeToString(b)
		t.byAppID[appID] = tok
	}
	t.byToken[tok] = principal{appID: appID, scopes: scopes}
	return tok
}

func (t *tokenReg) lookup(tok string) (principal, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	p, ok := t.byToken[tok]
	return p, ok
}

func (t *tokenReg) revoke(appID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if tok := t.byAppID[appID]; tok != "" {
		delete(t.byToken, tok)
		delete(t.byAppID, appID)
	}
}

// scopeManage is the sentinel for shell-only endpoints (install apps, run
// setup, manage the daemon). It is NOT in osapp's grantable set, so only the
// session token ("*") satisfies it — apps can never reach these.
const scopeManage = "shell:manage"

// scopeFor returns the scope an API request requires. It fails closed: any
// path not explicitly listed requires the shell-only scope, so a new endpoint
// is locked to the shell until someone deliberately opens it to apps.
func scopeFor(method, path string) string {
	p := strings.TrimPrefix(path, "/api/v1")
	switch {
	case p == "/ws" || strings.HasPrefix(p, "/session"):
		return "" // infra: any authed principal (the shell, or an app's own ws)
	case strings.HasPrefix(p, "/v1"), strings.HasPrefix(path, "/v1"):
		return osapp.ScopeAIGateway

	case p == "/fs/write" || p == "/fs/mkdir" || p == "/fs/rename" || p == "/fs/trash":
		return osapp.ScopeFSWrite
	case strings.HasPrefix(p, "/fs/"):
		return osapp.ScopeFSRead

	case strings.HasPrefix(p, "/term"):
		return osapp.ScopeTermExec

	case p == "/system/status" || p == "/system/metrics" || p == "/system/updates" || p == "/system/wifi/networks":
		return osapp.ScopeSysRead
	case strings.HasPrefix(p, "/system/"):
		return osapp.ScopeSysCtl

	case p == "/browser/open":
		return osapp.ScopeBrowser
	case strings.HasPrefix(p, "/office/"):
		return osapp.ScopeOffice
	case strings.HasPrefix(p, "/windows"):
		return osapp.ScopeWindows
	case p == "/notify":
		return osapp.ScopeNotify
	case p == "/settings":
		return osapp.ScopeSettings
	case strings.HasPrefix(p, "/ai/"):
		return osapp.ScopeAI

	default:
		return scopeManage // /apps*, /setup/*, anything new: shell only
	}
}
