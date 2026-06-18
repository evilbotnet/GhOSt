// ghostd — the GhOSt system daemon.
//
// Serves the built shell, exposes the system API (filesystem, terminal,
// system status, browser windows) over localhost HTTP + one WebSocket,
// authenticated with a per-session bearer token.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ghostos/ghostd/internal/admin"
	"github.com/ghostos/ghostd/internal/ai"
	"github.com/ghostos/ghostd/internal/browser"
	"github.com/ghostos/ghostd/internal/fsops"
	"github.com/ghostos/ghostd/internal/httpapi"
	"github.com/ghostos/ghostd/internal/kv"
	"github.com/ghostos/ghostd/internal/office"
	"github.com/ghostos/ghostd/internal/setup"
	"github.com/ghostos/ghostd/internal/system"
	"github.com/ghostos/ghostd/internal/term"
	"github.com/ghostos/ghostd/internal/webapps"
	"github.com/ghostos/ghostd/internal/windows"
	"github.com/ghostos/ghostd/internal/ws"
)

func main() {
	// `ghostd helper` is the privileged sidecar (ghost-admin.service, root).
	if len(os.Args) > 1 && os.Args[1] == "helper" {
		log.Fatal(admin.RunHelper())
	}

	listen := flag.String("listen", "127.0.0.1:7700", "address to bind (localhost only)")
	tokenFile := flag.String("token-file", "", "path to session token file (created if missing)")
	staticDir := flag.String("static", "", "directory with the built shell to serve")
	dev := flag.Bool("dev", false, "dev mode: expose /session/dev-token and allow the Vite origin")
	flag.Parse()

	if !strings.HasPrefix(*listen, "127.0.0.1:") && !strings.HasPrefix(*listen, "localhost:") {
		log.Fatalf("refusing to bind non-loopback address %q", *listen)
	}

	token, err := loadOrCreateToken(*tokenFile)
	if err != nil {
		log.Fatalf("token: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("home: %v", err)
	}

	hub := ws.NewHub()
	files := fsops.New([]string{home, "/media"})
	terms := term.NewManager(hub)
	sys := system.New()
	winMgr := windows.NewManager(hub)
	br := browser.New()
	ghost := ai.NewGhost(hub, ai.NewToolbox(files, sys, br))
	scheduler := ai.NewScheduler(ghost, hub)
	go scheduler.Start()
	go sys.PublishLoop(hub, 5*time.Second)

	srv := &httpapi.Server{
		Token:     token,
		Dev:       *dev,
		StaticDir: *staticDir,
		Hub:       hub,
		Files:     files,
		Terms:     terms,
		System:    sys,
		Browser:   br,
		Windows:   winMgr,
		Setup:     setup.New(),
		Ghost:     ghost,
		Scheduler: scheduler,
		WebApps:   webapps.New(),
		Settings:  kv.New(),
		Gateway:   ai.NewGateway(),
		Office: office.New(os.Getenv("GHOST_OFFICE_URL"), func() bool {
			tops, err := winMgr.List()
			if err != nil {
				return false
			}
			for _, t := range tops {
				if strings.HasPrefix(t.AppID, "chrome-localhost") {
					return true
				}
			}
			return false
		}),
	}

	log.Printf("ghostd listening on http://%s (dev=%v, static=%q)", *listen, *dev, *staticDir)
	if err := http.ListenAndServe(*listen, srv.Router()); err != nil {
		log.Fatal(err)
	}
}

func loadOrCreateToken(path string) (string, error) {
	if path == "" {
		// ephemeral token (still printed nowhere; dev endpoint hands it out)
		b := make([]byte, 32)
		rand.Read(b)
		return hex.EncodeToString(b), nil
	}
	if data, err := os.ReadFile(path); err == nil {
		if t := strings.TrimSpace(string(data)); t != "" {
			return t, nil
		}
	}
	b := make([]byte, 32)
	rand.Read(b)
	t := hex.EncodeToString(b)
	if err := os.WriteFile(path, []byte(t+"\n"), 0o600); err != nil {
		return "", err
	}
	return t, nil
}
