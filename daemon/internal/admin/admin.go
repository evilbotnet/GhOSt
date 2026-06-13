// Package admin is GhOSt's privileged helper — the OS equivalent of
// systemd-timedated. ghostd runs unprivileged as the kiosk user; the few
// operations that genuinely need root (setting a password, granting sudo,
// timezone/hostname) go through a root-owned unix socket that only the
// ghost user can reach. The helper validates every input and knows only
// these four verbs — it is not a general root shell.
package admin

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const SocketPath = "/run/ghost/admin.sock"

type Request struct {
	Action   string `json:"action"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Timezone string `json:"timezone,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

type Response struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

var (
	tzRe   = regexp.MustCompile(`^[A-Za-z0-9_+\-/]{1,64}$`)
	hostRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
)

// RunHelper serves the admin socket. Must run as root (ghost-admin.service).
func RunHelper() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("the helper must run as root")
	}
	if err := os.MkdirAll("/run/ghost", 0o755); err != nil {
		return err
	}
	os.Remove(SocketPath)
	l, err := net.Listen("unix", SocketPath)
	if err != nil {
		return err
	}
	// Only root and the ghost group may talk to the helper.
	if g, err := user.Lookup("ghost"); err == nil {
		gid, _ := strconv.Atoi(g.Gid)
		os.Chown(SocketPath, 0, gid)
	}
	if err := os.Chmod(SocketPath, 0o660); err != nil {
		return err
	}
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	var req Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		json.NewEncoder(conn).Encode(Response{Error: "bad request"})
		return
	}
	if err := execute(req); err != nil {
		json.NewEncoder(conn).Encode(Response{Error: err.Error()})
		return
	}
	json.NewEncoder(conn).Encode(Response{OK: true})
}

func execute(req Request) error {
	switch req.Action {
	case "set-password":
		// The kiosk user only — this is "set *my* password", not user admin.
		if req.User != "ghost" {
			return fmt.Errorf("can only set the ghost user's password")
		}
		if len(req.Password) < 4 {
			return fmt.Errorf("password too short")
		}
		cmd := exec.Command("chpasswd")
		cmd.Stdin = strings.NewReader("ghost:" + req.Password + "\n")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("chpasswd: %s", strings.TrimSpace(string(out)))
		}
		return nil

	case "enable-sudo":
		// Password-required sudo (the user just set one in the wizard).
		return os.WriteFile("/etc/sudoers.d/010-ghost",
			[]byte("ghost ALL=(ALL:ALL) ALL\n"), 0o440)

	case "set-timezone":
		if !tzRe.MatchString(req.Timezone) || strings.Contains(req.Timezone, "..") {
			return fmt.Errorf("invalid timezone")
		}
		if out, err := exec.Command("timedatectl", "set-timezone", req.Timezone).CombinedOutput(); err != nil {
			return fmt.Errorf("%s", strings.TrimSpace(string(out)))
		}
		return nil

	case "set-hostname":
		if !hostRe.MatchString(req.Hostname) {
			return fmt.Errorf("invalid hostname")
		}
		if out, err := exec.Command("hostnamectl", "set-hostname", req.Hostname).CombinedOutput(); err != nil {
			return fmt.Errorf("%s", strings.TrimSpace(string(out)))
		}
		return nil
	}
	return fmt.Errorf("unknown action %q", req.Action)
}

// Call sends one request to the helper (from unprivileged ghostd).
func Call(req Request) error {
	conn, err := net.DialTimeout("unix", SocketPath, 2*time.Second)
	if err != nil {
		return fmt.Errorf("admin helper unavailable: %w", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(15 * time.Second))
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return err
	}
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return err
	}
	if !resp.OK {
		return fmt.Errorf("%s", resp.Error)
	}
	return nil
}
