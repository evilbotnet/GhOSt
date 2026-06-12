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

	"github.com/ghostos/ghostd/internal/browser"
	"github.com/ghostos/ghostd/internal/fsops"
	"github.com/ghostos/ghostd/internal/office"
	"github.com/ghostos/ghostd/internal/system"
	"github.com/ghostos/ghostd/internal/term"
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
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

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
			authed.Put("/fs/write", s.fsWrite)
			authed.Post("/fs/mkdir", s.fsMkdir)
			authed.Post("/fs/rename", s.fsRename)
			authed.Post("/fs/trash", s.fsTrash)

			authed.Post("/term", s.termCreate)
			authed.Delete("/term/{id}", s.termClose)

			authed.Get("/system/status", s.systemStatus)
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
		})
	})

	if s.StaticDir != "" {
		r.Get("/*", s.serveShell)
	}
	return r
}

// auth validates the bearer token and, defense-in-depth, the Origin header.
func (s *Server) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		} else {
			token = r.URL.Query().Get("token") // WebSocket
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(s.Token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && !s.originAllowed(origin) {
			http.Error(w, "forbidden origin", http.StatusForbidden)
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
	writeJSON(w, map[string]string{"path": path})
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
