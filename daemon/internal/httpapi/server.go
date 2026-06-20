// Package httpapi is the REST surface + auth middleware + static serving.
//
// Auth model: every /api request needs the per-session bearer token (or
// ?token= for the WebSocket). The daemon serves the built shell and injects
// the token into index.html for its own origin only; a malicious website in a
// browsing window can neither read the token nor pass the Origin check.
package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/ghostos/ghostd/internal/admin"
	"github.com/ghostos/ghostd/internal/ai"
	"github.com/ghostos/ghostd/internal/backup"
	"github.com/ghostos/ghostd/internal/browser"
	"github.com/ghostos/ghostd/internal/fsops"
	"github.com/ghostos/ghostd/internal/kv"
	"github.com/ghostos/ghostd/internal/office"
	"github.com/ghostos/ghostd/internal/osapp"
	"github.com/ghostos/ghostd/internal/setup"
	"github.com/ghostos/ghostd/internal/store"
	"github.com/ghostos/ghostd/internal/system"
	"github.com/ghostos/ghostd/internal/term"
	"github.com/ghostos/ghostd/internal/webapps"
	"github.com/ghostos/ghostd/internal/windows"
	"github.com/ghostos/ghostd/internal/ws"
)

type Server struct {
	Token     string
	Dev       bool
	StaticDir string
	Hub       *ws.Hub
	Files     *fsops.FS
	Terms     *term.Manager
	System    *system.System
	Browser   *browser.Browser
	Windows   *windows.Manager
	Office    *office.Manager
	Setup     *setup.Manager
	Ghost     *ai.Ghost
	Scheduler *ai.Scheduler
	WebApps   *webapps.Store
	OSApps    *osapp.Store
	Store     *store.Store
	Settings  *kv.Store
	Gateway   *ai.Gateway

	tokens *tokenReg // per-app scoped tokens (ADR 0009); lazily initialized
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	if s.tokens == nil {
		s.tokens = newTokenReg()
	}

	r.Route("/api/v1", func(api chi.Router) {
		// Dev-only bootstrap: lets the Vite-served shell fetch the token.
		if s.Dev {
			api.Get("/session/dev-token", func(w http.ResponseWriter, r *http.Request) {
				writeJSON(w, map[string]string{"token": s.Token})
			})
		}

		api.Group(func(authed chi.Router) {
			authed.Use(s.auth)

			authed.Get("/ws", s.Hub.ServeHTTP)

			authed.Get("/fs/home", s.fsHome)
			authed.Get("/fs/list", s.fsList)
			authed.Get("/fs/read", s.fsRead)
			authed.Get("/fs/raw", s.fsRaw)
			authed.Put("/fs/write", s.fsWrite)
			authed.Post("/fs/mkdir", s.fsMkdir)
			authed.Post("/fs/rename", s.fsRename)
			authed.Post("/fs/trash", s.fsTrash)

			authed.Post("/term", s.termCreate)
			authed.Delete("/term/{id}", s.termClose)

			authed.Get("/system/status", s.systemStatus)
			authed.Get("/system/metrics", s.systemMetrics)
			authed.Post("/system/lock", s.systemLock)
			authed.Get("/system/updates", s.systemUpdates)
			authed.Get("/system/wifi/networks", s.wifiNetworks)
			authed.Post("/system/wifi/connect", s.wifiConnect)
			authed.Post("/system/volume", s.setVolume)

			authed.Post("/browser/open", s.browserOpen)
			authed.Get("/office/status", s.officeStatus)
			authed.Post("/office/open", s.officeOpen)
			authed.Post("/office/close", s.officeClose)
			authed.Post("/office/launch", s.officeLaunch)

			authed.Get("/windows", s.windowsList)
			authed.Post("/windows/action", s.windowsAction)

			authed.Post("/system/screenshot", s.screenshot)

			authed.Get("/ai/status", s.aiStatus)
			authed.Get("/ai/skills", s.aiSkills)
			authed.Get("/ai/tools", s.aiTools)
			authed.Get("/ai/soul", s.aiSoul)
			authed.Get("/ai/mcp", s.aiMCP)
			authed.Get("/ai/memory", s.memoryList)
			authed.Post("/ai/memory", s.memorySave)
			authed.Delete("/ai/memory/{name}", s.memoryRemove)
			authed.Get("/ai/schedules", s.schedulesList)
			authed.Post("/ai/schedules", s.scheduleSave)
			authed.Delete("/ai/schedules/{id}", s.scheduleRemove)
			authed.Post("/ai/schedules/{id}/run", s.scheduleRun)
			authed.Post("/setup/soul", s.setupSoul)
			authed.Post("/setup/mcp", s.mcpAdd)
			authed.Delete("/setup/mcp/{name}", s.mcpRemove)

			authed.Get("/settings", s.settingsGet)
			authed.Put("/settings", s.settingsPut)
			authed.Post("/notify", s.notify)

			authed.Get("/backup/export", s.backupExport)
			authed.Post("/backup/import", s.backupImport)

			authed.Get("/apps", s.appsList)
			authed.Post("/apps/install", s.appsInstall)
			authed.Post("/apps/launch", s.appsLaunch)
			authed.Delete("/apps/{id}", s.appsUninstall)

			authed.Get("/osapps", s.osappsList)
			authed.Delete("/osapps/{id}", s.osappUninstall)

			authed.Get("/store", s.storeCatalog)
			authed.Put("/store/config", s.storeConfig)
			authed.Post("/store/install", s.storeInstall)

			authed.Get("/setup/status", s.setupStatus)
			authed.Get("/setup/timezones", s.setupTimezones)
			authed.Post("/setup/password", s.setupPassword)
			authed.Post("/setup/timezone", s.setupTimezone)
			authed.Post("/setup/hostname", s.setupHostname)
			authed.Post("/setup/ai", s.setupAI)
			authed.Post("/setup/complete", s.setupComplete)
		})
	})

	// OpenAI-compatible model gateway (ADR 0003): /v1/* proxies to the user's
	// configured model. Token-gated like the rest of the API; tools set the
	// GhOSt session token as their OpenAI API key.
	r.Route("/v1", func(v chi.Router) {
		v.Use(s.auth)
		v.Handle("/*", s.Gateway)
	})

	// Installed .osapp packages are served at /apps/<id>/ (ADR 0009), each with
	// its own scoped token injected — distinct from the shell's static files.
	if s.OSApps != nil {
		r.Handle("/apps/{id}/*", http.HandlerFunc(s.serveApp))
		r.Handle("/apps/{id}", http.HandlerFunc(s.serveApp))
	}

	if s.StaticDir != "" {
		r.Get("/*", s.serveShell)
	}
	return r
}

// auth validates the bearer token, enforces the Origin header for the superuser
// session token, and checks the scope the request path requires (ADR 0009).
//
// Two principals, two origin rules:
//   - The session token is the superuser ("*"). It is only honoured from the
//     shell's own origin — defense against CSRF / DNS-rebinding, since a
//     browsing window shares the daemon's host.
//   - A per-app token carries only its granted scopes and *is* the capability
//     (unguessable, unforgeable). Each app runs in a sandboxed opaque origin
//     (its requests carry Origin: null), so app tokens are not origin-gated;
//     the token + scope check is the boundary. A malicious website cannot
//     obtain an app token, and even if it did it would be scope-limited.
func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		} else {
			token = r.URL.Query().Get("token") // WebSocket
		}

		var scopes []string
		isSession := subtle.ConstantTimeCompare([]byte(token), []byte(s.Token)) == 1
		if isSession {
			scopes = []string{osapp.ScopeAll}
		} else if p, ok := s.tokens.lookup(token); ok {
			scopes = p.scopes
		} else {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// The all-powerful session token must come from the shell origin; app
		// tokens are exempt (they run in opaque origins and are scope-bound).
		if isSession {
			if origin := r.Header.Get("Origin"); origin != "" && !s.originAllowed(origin) {
				http.Error(w, "forbidden origin", http.StatusForbidden)
				return
			}
		}

		if required := scopeFor(r.Method, r.URL.Path); !osapp.Allows(scopes, required) {
			http.Error(w, "forbidden: this app lacks the "+required+" permission", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) originAllowed(origin string) bool {
	allowed := []string{"http://127.0.0.1:7700", "http://localhost:7700"}
	if s.Dev {
		allowed = append(allowed, "http://localhost:5173", "http://127.0.0.1:5173")
	}
	for _, a := range allowed {
		if origin == a {
			return true
		}
	}
	return false
}

// serveShell serves the built shell, injecting the session token into
// index.html so the app boots authenticated (same-origin only).
func (s *Server) serveShell(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.StaticDir, filepath.Clean(r.URL.Path))
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		path = filepath.Join(s.StaticDir, "index.html")
	}
	if filepath.Base(path) == "index.html" {
		data, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, "shell not built", http.StatusInternalServerError)
			return
		}
		inject := fmt.Sprintf("<script>window.__GHOST_TOKEN__=%q;</script>", s.Token)
		html := strings.Replace(string(data), "<head>", "<head>"+inject, 1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write([]byte(html))
		return
	}
	http.ServeFile(w, r, path)
}

// serveApp serves an installed .osapp's files from its install dir at
// /apps/<id>/, injecting the app's own scoped token into its entry HTML so it
// boots authenticated with exactly the permissions it was granted (ADR 0009).
func (s *Server) serveApp(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	inst, ok := s.OSApps.Get(id)
	if !ok || !inst.Enabled {
		http.NotFound(w, r)
		return
	}
	base := s.OSApps.Dir(id)

	rel := chi.URLParam(r, "*")
	if rel == "" || rel == "/" {
		rel = inst.Entry
	}
	full := filepath.Join(base, filepath.Clean("/"+rel))
	if full != base && !strings.HasPrefix(full, base+string(os.PathSeparator)) {
		http.NotFound(w, r)
		return
	}
	if info, err := os.Stat(full); err != nil || info.IsDir() {
		full = filepath.Join(base, filepath.Clean("/"+inst.Entry))
	}

	// Inject the scoped token only into the entry HTML.
	if full == filepath.Join(base, filepath.Clean("/"+inst.Entry)) {
		data, err := os.ReadFile(full)
		if err != nil {
			http.Error(w, "app entry missing", http.StatusInternalServerError)
			return
		}
		tok := s.tokens.tokenForApp(id, inst.Granted)
		inject := fmt.Sprintf("<script>window.__GHOST_TOKEN__=%q;window.__GHOST_APP__=%q;</script>", tok, id)
		html := string(data)
		if strings.Contains(html, "<head>") {
			html = strings.Replace(html, "<head>", "<head>"+inject, 1)
		} else {
			html = inject + html
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		// Daemon-enforced isolation (ADR 0009): the CSP sandbox directive forces
		// the browser to give this document a unique *opaque* origin — no
		// allow-same-origin — so it cannot reach the shell's window or read its
		// token, and the shell cannot read into it. This holds even if the shell
		// loaded it top-level or forgot the iframe sandbox attribute. frame-
		// ancestors limits who may frame it to the shell's own origin.
		w.Header().Set("Content-Security-Policy", "sandbox allow-scripts allow-forms allow-modals; frame-ancestors 'self'")
		w.Write([]byte(html))
		return
	}
	http.ServeFile(w, r, full)
}

func (s *Server) osappsList(w http.ResponseWriter, r *http.Request) {
	list := s.OSApps.List()
	if list == nil {
		list = []osapp.Installed{}
	}
	writeJSON(w, list)
}

func (s *Server) osappUninstall(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.OSApps.Uninstall(id); err != nil {
		writeErr(w, err)
		return
	}
	s.tokens.revoke(id)
	writeJSON(w, map[string]bool{"ok": true})
}

// storeCatalog returns the store config plus the verified catalog (or the
// reason it couldn't be fetched/verified, so the Hub can guide the user).
func (s *Server) storeCatalog(w http.ResponseWriter, r *http.Request) {
	cfg := s.Store.Config()
	resp := map[string]any{"configured": cfg.IndexURL != "" && cfg.PublicKey != "", "url": cfg.IndexURL}
	idx, err := s.Store.Catalog()
	if err != nil {
		resp["error"] = err.Error()
		resp["entries"] = []store.Entry{}
	} else {
		resp["entries"] = idx.Entries
		resp["generated"] = idx.Generated
	}
	writeJSON(w, resp)
}

func (s *Server) storeConfig(w http.ResponseWriter, r *http.Request) {
	var c store.Config
	if !readJSON(w, r, &c) {
		return
	}
	if err := s.Store.Configure(c); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) storeInstall(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID      string   `json:"id"`
		Granted []string `json:"granted"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Store.Install(req.ID, req.Granted); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// ---- handlers ----

func (s *Server) fsHome(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"path": s.Files.Home()})
}

func (s *Server) fsList(w http.ResponseWriter, r *http.Request) {
	path, entries, err := s.Files.List(r.URL.Query().Get("path"))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]any{"path": path, "entries": entries})
}

func (s *Server) fsRead(w http.ResponseWriter, r *http.Request) {
	data, err := s.Files.Read(r.URL.Query().Get("path"))
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write(data)
}

// fsRaw streams a file's bytes with content-type + range support (images,
// PDFs) — confined to the allowed roots. Auth is via the bearer token or, for
// <img>/<iframe> tags that can't set headers, the ?token= query param handled
// by the auth middleware.
func (s *Server) fsRaw(w http.ResponseWriter, r *http.Request) {
	path, err := s.Files.RawPath(r.URL.Query().Get("path"))
	if err != nil {
		writeErr(w, err)
		return
	}
	http.ServeFile(w, r, path)
}

func (s *Server) systemMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.System.Metrics())
}

func (s *Server) systemLock(w http.ResponseWriter, r *http.Request) {
	if err := s.System.Lock(); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) systemUpdates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.System.Updates())
}

func (s *Server) mcpAdd(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Command, Transport, URL string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := ai.AddMCPServer(req.Name, req.Transport, strings.Fields(req.Command), req.URL); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) mcpRemove(w http.ResponseWriter, r *http.Request) {
	if err := ai.RemoveMCPServer(chi.URLParam(r, "name")); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) settingsGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.Settings.All())
}

func (s *Server) settingsPut(w http.ResponseWriter, r *http.Request) {
	var req struct{ Key, Value string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Settings.Set(req.Key, req.Value); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// notify raises a desktop notification (also reachable by server-side events).
func (s *Server) notify(w http.ResponseWriter, r *http.Request) {
	var req struct{ Title, Body, Kind string }
	if !readJSON(w, r, &req) {
		return
	}
	s.Hub.Publish("notify", "show", map[string]string{
		"title": req.Title, "body": req.Body, "kind": req.Kind,
	})
	writeJSON(w, map[string]bool{"ok": true})
}

// backupExport streams a .tar.gz of all GhOSt state as a download. Shell-only
// (the manage scope); the ?token query param lets the browser download link
// authenticate without a header.
func (s *Server) backupExport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", `attachment; filename="ghost-backup.tar.gz"`)
	// If Export fails mid-stream the download is truncated; the client validates
	// on import, so a bad export can't silently restore.
	_ = backup.Export(w)
}

func (s *Server) backupImport(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	if err := backup.Import(r.Body); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) fsWrite(w http.ResponseWriter, r *http.Request) {
	var req struct{ Path, Content string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Files.Write(req.Path, []byte(req.Content)); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) fsMkdir(w http.ResponseWriter, r *http.Request) {
	var req struct{ Path string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Files.Mkdir(req.Path); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) fsRename(w http.ResponseWriter, r *http.Request) {
	var req struct{ From, To string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Files.Rename(req.From, req.To); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) fsTrash(w http.ResponseWriter, r *http.Request) {
	var req struct{ Path string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Files.Trash(req.Path); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) termCreate(w http.ResponseWriter, r *http.Request) {
	var req struct{ Cols, Rows int }
	if !readJSON(w, r, &req) {
		return
	}
	if req.Cols <= 0 {
		req.Cols = 80
	}
	if req.Rows <= 0 {
		req.Rows = 24
	}
	id, err := s.Terms.Create(req.Cols, req.Rows)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]string{"id": id})
}

func (s *Server) termClose(w http.ResponseWriter, r *http.Request) {
	s.Terms.Close(chi.URLParam(r, "id"))
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) systemStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.System.Status())
}

func (s *Server) wifiNetworks(w http.ResponseWriter, r *http.Request) {
	nets, err := s.System.WifiNetworks()
	if err != nil {
		writeErr(w, err)
		return
	}
	if nets == nil {
		nets = []system.WifiNetwork{}
	}
	writeJSON(w, nets)
}

func (s *Server) wifiConnect(w http.ResponseWriter, r *http.Request) {
	var req struct{ SSID, Password string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.System.WifiConnect(req.SSID, req.Password); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) setVolume(w http.ResponseWriter, r *http.Request) {
	var req struct{ Percent int }
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.System.SetVolume(req.Percent); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) browserOpen(w http.ResponseWriter, r *http.Request) {
	var req struct{ URL string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Browser.Open(req.URL); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) windowsList(w http.ResponseWriter, r *http.Request) {
	if !s.Windows.Available() {
		writeJSON(w, []windows.Toplevel{})
		return
	}
	tops, err := s.Windows.List()
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, tops)
}

func (s *Server) windowsAction(w http.ResponseWriter, r *http.Request) {
	var req struct{ AppID, Title, Action string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := s.Windows.Act(req.Action, req.AppID, req.Title); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) officeStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"available": s.Office.Available(),
		"url":       s.Office.URL(),
		"running":   s.Office.Running(),
	})
}

func (s *Server) officeOpen(w http.ResponseWriter, r *http.Request) {
	if !s.Office.Available() {
		http.Error(w, "office not configured", http.StatusNotFound)
		return
	}
	s.Office.Open()
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) officeClose(w http.ResponseWriter, r *http.Request) {
	s.Office.Close()
	writeJSON(w, map[string]bool{"ok": true})
}

// officeLaunch opens CryptPad as a native chromeless app window — its CSP
// (frame-ancestors 'self') forbids iframing from the shell origin.
func (s *Server) officeLaunch(w http.ResponseWriter, r *http.Request) {
	if !s.Office.Available() {
		http.Error(w, "office not configured", http.StatusNotFound)
		return
	}
	if err := s.Browser.OpenApp(s.Office.URL() + "/drive/"); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) screenshot(w http.ResponseWriter, r *http.Request) {
	path, err := s.System.Screenshot()
	if err != nil {
		writeErr(w, err)
		return
	}
	s.Hub.Publish("notify", "show", map[string]string{
		"title": "Screenshot saved", "body": filepath.Base(path), "kind": "success",
	})
	writeJSON(w, map[string]string{"path": path})
}

func (s *Server) aiStatus(w http.ResponseWriter, r *http.Request) {
	configured, provider := s.Ghost.Configured()
	soul := s.Ghost.Soul()
	writeJSON(w, map[string]any{
		"configured": configured,
		"provider":   provider,
		"name":       soul.Name,
		"hatched":    soul.Hatched(),
	})
}

func (s *Server) aiSoul(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.Ghost.Soul())
}

func (s *Server) aiMCP(w http.ResponseWriter, r *http.Request) {
	servers := s.Ghost.MCPServers()
	if servers == nil {
		servers = []ai.MCPServerInfo{}
	}
	writeJSON(w, servers)
}

func (s *Server) setupSoul(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Body string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := ai.SaveSoul(req.Name, req.Body); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) aiSkills(w http.ResponseWriter, r *http.Request) {
	skills := s.Ghost.Skills()
	if skills == nil {
		skills = []ai.Skill{}
	}
	writeJSON(w, skills)
}

func (s *Server) aiTools(w http.ResponseWriter, r *http.Request) {
	tools := s.Ghost.Tools()
	if tools == nil {
		tools = []ai.ExtToolInfo{}
	}
	writeJSON(w, tools)
}

func (s *Server) memoryList(w http.ResponseWriter, r *http.Request) {
	mems := s.Ghost.Memories()
	if mems == nil {
		mems = []ai.Memory{}
	}
	writeJSON(w, mems)
}

func (s *Server) memorySave(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, Description, Body string }
	if !readJSON(w, r, &req) {
		return
	}
	m, err := ai.SaveMemory(req.Name, req.Description, req.Body)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, m)
}

func (s *Server) memoryRemove(w http.ResponseWriter, r *http.Request) {
	if err := ai.DeleteMemory(chi.URLParam(r, "name")); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) schedulesList(w http.ResponseWriter, r *http.Request) {
	list := s.Scheduler.List()
	if list == nil {
		list = []ai.Schedule{}
	}
	writeJSON(w, list)
}

func (s *Server) scheduleSave(w http.ResponseWriter, r *http.Request) {
	var sc ai.Schedule
	if !readJSON(w, r, &sc) {
		return
	}
	writeJSON(w, s.Scheduler.Save(sc))
}

func (s *Server) scheduleRemove(w http.ResponseWriter, r *http.Request) {
	s.Scheduler.Remove(chi.URLParam(r, "id"))
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) scheduleRun(w http.ResponseWriter, r *http.Request) {
	result, ok := s.Scheduler.RunNow(chi.URLParam(r, "id"))
	if !ok {
		http.Error(w, `{"error":"no such schedule"}`, http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]string{"result": result})
}

func (s *Server) appsList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.WebApps.List())
}

func (s *Server) appsInstall(w http.ResponseWriter, r *http.Request) {
	var req struct{ Name, URL, Icon string }
	if !readJSON(w, r, &req) {
		return
	}
	if req.Icon == "" {
		req.Icon = webapps.IconForURL(req.URL)
	}
	app, err := s.WebApps.Install(req.Name, req.URL, req.Icon)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, app)
}

func (s *Server) appsLaunch(w http.ResponseWriter, r *http.Request) {
	var req struct{ ID string }
	if !readJSON(w, r, &req) {
		return
	}
	url, ok := s.WebApps.URLFor(req.ID)
	if !ok {
		http.Error(w, "no such app", http.StatusNotFound)
		return
	}
	if err := s.Browser.OpenApp(url); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) appsUninstall(w http.ResponseWriter, r *http.Request) {
	if err := s.WebApps.Uninstall(chi.URLParam(r, "id")); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// ---- first-boot setup ----

func (s *Server) setupStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]bool{"needed": s.Setup.Needed()})
}

func (s *Server) setupTimezones(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, s.Setup.Timezones())
}

// setupPassword sets the kiosk user's password and unlocks sudo for them —
// the moment the machine becomes the user's (root stays locked; sudo asks
// for this password).
func (s *Server) setupPassword(w http.ResponseWriter, r *http.Request) {
	var req struct{ Password string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := admin.Call(admin.Request{Action: "set-password", User: "ghost", Password: req.Password}); err != nil {
		writeErr(w, err)
		return
	}
	if err := admin.Call(admin.Request{Action: "enable-sudo"}); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) setupTimezone(w http.ResponseWriter, r *http.Request) {
	var req struct{ Timezone string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := admin.Call(admin.Request{Action: "set-timezone", Timezone: req.Timezone}); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) setupHostname(w http.ResponseWriter, r *http.Request) {
	var req struct{ Hostname string }
	if !readJSON(w, r, &req) {
		return
	}
	if err := admin.Call(admin.Request{Action: "set-hostname", Hostname: req.Hostname}); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) setupAI(w http.ResponseWriter, r *http.Request) {
	var cfg setup.AIConfig
	if !readJSON(w, r, &cfg) {
		return
	}
	if err := s.Setup.SaveAI(cfg); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *Server) setupComplete(w http.ResponseWriter, r *http.Request) {
	if err := s.Setup.Complete(); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, map[string]bool{"ok": true})
}

// ---- helpers ----

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeErr(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case os.IsNotExist(err):
		status = http.StatusNotFound
	case os.IsPermission(err), err == fsops.ErrOutsideRoot:
		status = http.StatusForbidden
	}
	http.Error(w, err.Error(), status)
}
