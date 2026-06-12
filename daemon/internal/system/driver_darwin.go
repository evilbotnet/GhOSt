//go:build darwin

package system

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
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
