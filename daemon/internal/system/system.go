// Package system abstracts platform state (Wi-Fi, audio, battery) behind a
// driver interface. The Linux driver talks to NetworkManager/PipeWire/UPower;
// the Darwin driver is the macOS dev-loop stand-in (partly mocked).
package system

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/ghostos/ghostd/internal/ws"
)

type WifiStatus struct {
	Available bool   `json:"available"`
	Connected bool   `json:"connected"`
	SSID      string `json:"ssid"`
	Signal    int    `json:"signal"`
}

type BatteryStatus struct {
	Available bool `json:"available"`
	Charging  bool `json:"charging"`
	Percent   int  `json:"percent"`
}

type VolumeStatus struct {
	Percent int  `json:"percent"`
	Muted   bool `json:"muted"`
}

type Status struct {
	Hostname string        `json:"hostname"`
	Platform string        `json:"platform"`
	Wifi     WifiStatus    `json:"wifi"`
	Battery  BatteryStatus `json:"battery"`
	Volume   VolumeStatus  `json:"volume"`
}

type WifiNetwork struct {
	SSID    string `json:"ssid"`
	Signal  int    `json:"signal"`
	Secured bool   `json:"secured"`
	Known   bool   `json:"known"`
	Active  bool   `json:"active"`
}

type ProcInfo struct {
	PID   int     `json:"pid"`
	Name  string  `json:"name"`
	CPU   float64 `json:"cpu"`
	MemMB float64 `json:"memMB"`
}

type Metrics struct {
	CPUPercent  float64    `json:"cpuPercent"`
	MemUsedMB   int        `json:"memUsedMB"`
	MemTotalMB  int        `json:"memTotalMB"`
	DiskUsedGB  float64    `json:"diskUsedGB"`
	DiskTotalGB float64    `json:"diskTotalGB"`
	Uptime      string     `json:"uptime"`
	Load        string     `json:"load"`
	Processes   []ProcInfo `json:"processes"`
}

type UpdateInfo struct {
	Count    int      `json:"count"`
	Packages []string `json:"packages"`
}

type Driver interface {
	Status() Status
	WifiNetworks() ([]WifiNetwork, error)
	WifiConnect(ssid, password string) error
	SetVolume(percent int) error
	Screenshot(dir string) (string, error)
	Metrics() Metrics
	Lock() error
	Updates() UpdateInfo
}

type System struct {
	driver Driver
}

func New() *System {
	return &System{driver: newDriver()}
}

func (s *System) Status() Status {
	st := s.driver.Status()
	if st.Hostname == "" {
		st.Hostname, _ = os.Hostname()
	}
	return st
}

func (s *System) WifiNetworks() ([]WifiNetwork, error) { return s.driver.WifiNetworks() }
func (s *System) WifiConnect(ssid, pw string) error    { return s.driver.WifiConnect(ssid, pw) }
func (s *System) SetVolume(p int) error                { return s.driver.SetVolume(p) }
func (s *System) Metrics() Metrics                     { return s.driver.Metrics() }
func (s *System) Lock() error                          { return s.driver.Lock() }
func (s *System) Updates() UpdateInfo                  { return s.driver.Updates() }

// Screenshot captures the screen into ~/Pictures and returns the file path.
func (s *System) Screenshot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := home + "/Pictures"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return s.driver.Screenshot(dir)
}

// PublishLoop pushes status to the `system` topic so the tray stays live.
func (s *System) PublishLoop(hub *ws.Hub, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for range t.C {
		hub.Publish("system", "status", s.Status())
	}
}

// diskUsage returns used/total GB for the filesystem at path, via df (portable
// across Linux and macOS).
func diskUsage(path string) (usedGB, totalGB float64) {
	out, err := exec.Command("df", "-kP", path).Output()
	if err != nil {
		return 0, 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return 0, 0
	}
	f := strings.Fields(lines[len(lines)-1])
	if len(f) < 4 {
		return 0, 0
	}
	totalK, _ := strconv.ParseFloat(f[1], 64)
	usedK, _ := strconv.ParseFloat(f[2], 64)
	return usedK / 1024 / 1024, totalK / 1024 / 1024
}

func humanDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	m := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%d days, %d:%02d", days, h, m)
	}
	return fmt.Sprintf("%d:%02d", h, m)
}

// parsePS parses `ps` output with columns: pid comm %cpu rss(KB). Returns the
// top n data rows (header skipped).
func parsePS(out string, n int) []ProcInfo {
	var procs []ProcInfo
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i, line := range lines {
		if i == 0 {
			continue // header
		}
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		pid, _ := strconv.Atoi(f[0])
		cpu, _ := strconv.ParseFloat(f[len(f)-2], 64)
		rssKB, _ := strconv.ParseFloat(f[len(f)-1], 64)
		name := strings.Join(f[1:len(f)-2], " ")
		if base := name[strings.LastIndex(name, "/")+1:]; base != "" {
			name = base
		}
		procs = append(procs, ProcInfo{PID: pid, Name: name, CPU: cpu, MemMB: rssKB / 1024})
		if len(procs) >= n {
			break
		}
	}
	return procs
}
