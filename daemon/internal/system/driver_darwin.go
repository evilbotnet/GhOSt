//go:build darwin

package system

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// darwinDriver backs the macOS dev loop: volume is real (osascript),
// Wi-Fi is mocked (the macOS CLI for Wi-Fi is gone post-airport),
// battery is real when pmset is available.
type darwinDriver struct {
	mu     sync.Mutex
	volume int
}

func newDriver() Driver {
	d := &darwinDriver{volume: 50}
	if out, err := exec.Command("osascript", "-e", "output volume of (get volume settings)").Output(); err == nil {
		if v, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
			d.volume = v
		}
	}
	return d
}

func (d *darwinDriver) Status() Status {
	host, _ := os.Hostname()
	d.mu.Lock()
	vol := d.volume
	d.mu.Unlock()
	return Status{
		Hostname: strings.TrimSuffix(host, ".local"),
		Platform: "darwin (dev)",
		Wifi:     WifiStatus{Available: true, Connected: true, SSID: "dev-loopback", Signal: 82},
		Battery:  d.battery(),
		Volume:   VolumeStatus{Percent: vol},
	}
}

func (d *darwinDriver) battery() BatteryStatus {
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil || !strings.Contains(string(out), "%") {
		return BatteryStatus{Available: false}
	}
	text := string(out)
	pct := 0
	if i := strings.Index(text, "%"); i > 0 {
		j := i
		for j > 0 && text[j-1] >= '0' && text[j-1] <= '9' {
			j--
		}
		pct, _ = strconv.Atoi(text[j:i])
	}
	return BatteryStatus{
		Available: true,
		Charging:  strings.Contains(text, "AC Power"),
		Percent:   pct,
	}
}

func (d *darwinDriver) WifiNetworks() ([]WifiNetwork, error) {
	// Mock list for UI development; the Linux driver is the real one.
	return []WifiNetwork{
		{SSID: "dev-loopback", Signal: 82, Secured: true, Known: true, Active: true},
		{SSID: "Neighbornet-5G", Signal: 64, Secured: true},
		{SSID: "CoffeeShopGuest", Signal: 41, Secured: false},
		{SSID: "PrinterSetup-8C2A", Signal: 18, Secured: false},
	}, nil
}

func (d *darwinDriver) WifiConnect(ssid, password string) error {
	if ssid == "" {
		return fmt.Errorf("missing ssid")
	}
	return nil // mock: pretend it worked
}

func (d *darwinDriver) SetVolume(percent int) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("volume out of range")
	}
	d.mu.Lock()
	d.volume = percent
	d.mu.Unlock()
	return exec.Command("osascript", "-e", fmt.Sprintf("set volume output volume %d", percent)).Run()
}

func (d *darwinDriver) Screenshot(dir string) (string, error) {
	path := dir + "/screen-" + time.Now().Format("20060102-150405") + ".png"
	if err := exec.Command("screencapture", "-x", path).Run(); err != nil {
		return "", err
	}
	return path, nil
}

func (d *darwinDriver) Metrics() Metrics {
	m := Metrics{Processes: topProcesses()}
	// overall CPU: sum of per-process %cpu (rough, fine for the dev loop)
	for _, p := range m.Processes {
		_ = p
	}
	if out, err := exec.Command("sh", "-c", "ps -A -o %cpu | awk '{s+=$1} END {print s}'").Output(); err == nil {
		m.CPUPercent, _ = strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	}
	if out, err := exec.Command("sysctl", "-n", "hw.memsize").Output(); err == nil {
		bytes, _ := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		m.MemTotalMB = int(bytes / 1024 / 1024)
	}
	// used memory ~ total - free (approx via vm_stat page count)
	if out, err := exec.Command("sh", "-c", "vm_stat | awk '/Pages free/{f=$3} /Pages inactive/{i=$3} END{print f+i}'").Output(); err == nil {
		freePages, _ := strconv.ParseInt(strings.TrimSpace(strings.Trim(string(out), ".")), 10, 64)
		freeMB := int(freePages * 16384 / 1024 / 1024) // 16K page size on arm64
		if m.MemTotalMB > 0 {
			m.MemUsedMB = m.MemTotalMB - freeMB
		}
	}
	m.DiskUsedGB, m.DiskTotalGB = diskUsage("/")
	if out, err := exec.Command("uptime").Output(); err == nil {
		s := string(out)
		if i := strings.Index(s, "load averages:"); i >= 0 {
			m.Load = strings.TrimSpace(s[i+len("load averages:"):])
		}
	}
	m.Uptime = "dev"
	return m
}

func topProcesses() []ProcInfo {
	out, err := exec.Command("ps", "-Ao", "pid,comm,%cpu,rss", "-r").Output()
	if err != nil {
		return []ProcInfo{}
	}
	return parsePS(string(out), 6)
}
