// Package devkit installs terminal AI coding agents (pi, herdr) and wires them
// to GhOSt's model gateway (ADR 0003). The work is done by the user-level
// /usr/local/bin/ghost-install-devkit script; this package triggers it and
// reports status. ghostd runs as the ghost user, so no privilege is needed.
package devkit

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const scriptPath = "/usr/local/bin/ghost-install-devkit"

func home() string { h, _ := os.UserHomeDir(); return h }

func statePath() string { return filepath.Join(home(), ".config", "ghost", "devkit.state") }

type Status struct {
	NodePresent bool     `json:"nodePresent"` // Node.js available (prereq)
	State       string   `json:"state"`       // "" | "installing" | "ok" | "failed: …"
	Tools       []string `json:"tools"`       // installed wrappers found on PATH
	Available   bool     `json:"available"`   // the installer script is present (false on dev hosts)
}

// Get reports the current devkit state.
func Get() Status {
	s := Status{}
	if _, err := exec.LookPath("node"); err == nil {
		s.NodePresent = true
	}
	if _, err := os.Stat(scriptPath); err == nil {
		s.Available = true
	}
	if data, err := os.ReadFile(statePath()); err == nil {
		s.State = strings.TrimSpace(string(data))
	}
	bin := filepath.Join(home(), ".local", "bin")
	for _, t := range []string{"pi", "herdr"} {
		if fi, err := os.Stat(filepath.Join(bin, t)); err == nil && fi.Mode()&0o111 != 0 {
			s.Tools = append(s.Tools, t)
		}
	}
	return s
}

// Install kicks off the installer in the background (npm installs take minutes);
// callers poll Get() for progress. Returns an error if it can't be started.
func Install() error {
	if _, err := os.Stat(scriptPath); err != nil {
		return errNoInstaller
	}
	_ = os.WriteFile(statePath(), []byte("installing"), 0o600)
	cmd := exec.Command("bash", scriptPath)
	// Detach: we don't wait. The script updates devkit.state as it progresses.
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }() // reap without blocking
	return nil
}

type devkitError string

func (e devkitError) Error() string { return string(e) }

const errNoInstaller = devkitError("the devkit installer isn't present on this host")
