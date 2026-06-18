package osapp

// Permission scopes (ADR 0009). These are the vocabulary a .osapp manifest may
// request and the user may grant; ghostd's auth layer enforces them per request
// (see httpapi). The built-in shell holds the implicit superuser scope "*".
//
// Management scopes (installing apps, running setup) are intentionally NOT in
// this grantable set, so a third-party manifest can neither request nor be
// granted them — only the shell's session token ("*") reaches those endpoints.
const (
	ScopeFSRead    = "fs:home:ro"
	ScopeFSWrite   = "fs:home:rw"
	ScopeTermExec  = "term:exec"
	ScopeSysRead   = "system:read"
	ScopeSysCtl    = "system:control"
	ScopeWindows   = "windows"
	ScopeBrowser   = "browser"
	ScopeOffice    = "office"
	ScopeNotify    = "notify"
	ScopeAI        = "ai"         // talk to Ghost / read AI status
	ScopeAIGateway = "ai:gateway" // use the /v1 model gateway
	ScopeSettings  = "settings"

	// ScopeAll is the shell's implicit superuser scope — never granted to a
	// third-party app, only held by the session token.
	ScopeAll = "*"
)

var grantable = map[string]bool{
	ScopeFSRead: true, ScopeFSWrite: true, ScopeTermExec: true,
	ScopeSysRead: true, ScopeSysCtl: true, ScopeWindows: true,
	ScopeBrowser: true, ScopeOffice: true, ScopeNotify: true,
	ScopeAI: true, ScopeAIGateway: true, ScopeSettings: true,
}

// ScopeKnown reports whether a scope is a valid, grantable permission.
func ScopeKnown(s string) bool { return grantable[s] }

// Allows reports whether a set of granted scopes satisfies a required scope,
// honouring implications: "*" allows anything; write implies read; control
// implies read.
func Allows(granted []string, required string) bool {
	if required == "" {
		return true
	}
	for _, g := range granted {
		if g == ScopeAll || g == required {
			return true
		}
		if g == ScopeFSWrite && required == ScopeFSRead {
			return true
		}
		if g == ScopeSysCtl && required == ScopeSysRead {
			return true
		}
	}
	return false
}
