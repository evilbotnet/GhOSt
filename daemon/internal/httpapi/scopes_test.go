package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghostos/ghostd/internal/osapp"
)

func TestScopeForTable(t *testing.T) {
	cases := []struct {
		method, path, want string
	}{
		{"GET", "/api/v1/fs/list", osapp.ScopeFSRead},
		{"PUT", "/api/v1/fs/write", osapp.ScopeFSWrite},
		{"POST", "/api/v1/term", osapp.ScopeTermExec},
		{"GET", "/api/v1/system/metrics", osapp.ScopeSysRead},
		{"POST", "/api/v1/system/volume", osapp.ScopeSysCtl},
		{"POST", "/api/v1/notify", osapp.ScopeNotify},
		{"POST", "/v1/chat/completions", osapp.ScopeAIGateway},
		{"DELETE", "/api/v1/osapps/x", scopeManage}, // shell-only, fails closed
		{"GET", "/api/v1/some/future/endpoint", scopeManage},
		{"GET", "/api/v1/ws", ""},
	}
	for _, c := range cases {
		if got := scopeFor(c.method, c.path); got != c.want {
			t.Errorf("scopeFor(%s %s) = %q, want %q", c.method, c.path, got, c.want)
		}
	}
}

// TestAuthScopeEnforcement drives the real auth middleware with a superuser
// session token and a scoped app token, asserting the app is confined.
func TestAuthScopeEnforcement(t *testing.T) {
	s := &Server{Token: "session-token", tokens: newTokenReg()}
	appTok := s.tokens.tokenForApp("tone.studio", []string{osapp.ScopeFSWrite})

	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	h := s.auth(ok)

	call := func(token, method, path string) int {
		req := httptest.NewRequest(method, path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	// Session token is the superuser — everything passes.
	if c := call("session-token", "PUT", "/api/v1/fs/write"); c != 200 {
		t.Errorf("session fs/write = %d, want 200", c)
	}
	if c := call("session-token", "DELETE", "/api/v1/osapps/x"); c != 200 {
		t.Errorf("session manage = %d, want 200", c)
	}

	// App with fs:home:rw: write allowed, read allowed (implied)...
	if c := call(appTok, "PUT", "/api/v1/fs/write"); c != 200 {
		t.Errorf("app fs/write = %d, want 200", c)
	}
	if c := call(appTok, "GET", "/api/v1/fs/list"); c != 200 {
		t.Errorf("app fs/list (implied ro) = %d, want 200", c)
	}
	// ...but term and management are forbidden.
	if c := call(appTok, "POST", "/api/v1/term"); c != 403 {
		t.Errorf("app term = %d, want 403", c)
	}
	if c := call(appTok, "DELETE", "/api/v1/osapps/x"); c != 403 {
		t.Errorf("app manage = %d, want 403", c)
	}

	// Unknown token is unauthorized.
	if c := call("bogus", "GET", "/api/v1/fs/list"); c != 401 {
		t.Errorf("bogus token = %d, want 401", c)
	}

	// Revoking the app kills its token.
	s.tokens.revoke("tone.studio")
	if c := call(appTok, "GET", "/api/v1/fs/list"); c != 401 {
		t.Errorf("revoked app = %d, want 401", c)
	}
}

// TestOriginRules covers the split origin policy (ADR 0009): the superuser
// session token is locked to the shell origin, while a scoped app token (which
// runs in a sandboxed opaque origin → Origin: null) is exempt but can never
// escalate beyond its grant regardless of where the request claims to come from.
func TestOriginRules(t *testing.T) {
	s := &Server{Token: "session-token", tokens: newTokenReg()}
	appTok := s.tokens.tokenForApp("tone.studio", []string{osapp.ScopeFSWrite})
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	h := s.auth(ok)

	call := func(token, method, path, origin string) int {
		req := httptest.NewRequest(method, path, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}

	// Session token: only valid from the shell's own origin.
	if c := call("session-token", "PUT", "/api/v1/fs/write", "http://127.0.0.1:7700"); c != 200 {
		t.Errorf("session @ shell origin = %d, want 200", c)
	}
	if c := call("session-token", "PUT", "/api/v1/fs/write", "https://evil.example"); c != 403 {
		t.Errorf("session @ foreign origin = %d, want 403", c)
	}
	if c := call("session-token", "PUT", "/api/v1/fs/write", "null"); c != 403 {
		t.Errorf("session @ null origin = %d, want 403", c)
	}

	// App token from an opaque origin (Origin: null) works for its scope...
	if c := call(appTok, "PUT", "/api/v1/fs/write", "null"); c != 200 {
		t.Errorf("app @ null origin (granted) = %d, want 200", c)
	}
	// ...and even from a forged foreign origin it CANNOT escalate past its grant.
	if c := call(appTok, "POST", "/api/v1/term", "https://evil.example"); c != 403 {
		t.Errorf("app escalation attempt = %d, want 403", c)
	}
	if c := call(appTok, "DELETE", "/api/v1/osapps/x", "null"); c != 403 {
		t.Errorf("app management attempt = %d, want 403", c)
	}
}
