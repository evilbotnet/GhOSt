// Package browser opens native browsing windows above the shell.
// Linux: new windows of the running Chromium instance (shared profile, so
// they land in the existing process). macOS dev loop: the default browser.
package browser

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type Browser struct{}

func New() *Browser { return &Browser{} }

func profileDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ghost-browser")
}

func check(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("refusing to open non-http(s) url")
	}
	return u.String(), nil
}

// Open opens a normal tabbed browser window.
func (b *Browser) Open(raw string) error {
	u, err := check(raw)
	if err != nil {
		return err
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", u).Start()
	}
	return exec.Command("chromium",
		"--user-data-dir="+profileDir(), "--new-window", u).Start()
}

// OpenApp opens a chromeless app window — used for local web apps like
// CryptPad that look like native apps but can't be iframed (CSP).
func (b *Browser) OpenApp(raw string) error {
	u, err := check(raw)
	if err != nil {
		return err
	}
	if runtime.GOOS == "darwin" {
		return exec.Command("open", u).Start()
	}
	return exec.Command("chromium",
		"--user-data-dir="+profileDir(), "--app="+u).Start()
}
