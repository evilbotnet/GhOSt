// Package browser opens native browsing windows above the shell.
// Linux: a new window of the running Chromium instance (shared profile, so
// it lands in the existing process). macOS dev loop: the default browser.
package browser

import (
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

type Browser struct{}

func New() *Browser { return &Browser{} }

func (b *Browser) Open(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return fmt.Errorf("refusing to open non-http(s) url")
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", u.String()).Start()
	default:
		// Same profile dir as the kiosk session (see os/overlay autostart):
		// Chromium forwards this to the already-running instance.
		return exec.Command("chromium",
			"--user-data-dir=/home/ghost/.config/ghost-browser",
			"--new-window", u.String()).Start()
	}
}
