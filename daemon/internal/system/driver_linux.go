//go:build linux

package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// linuxDriver v1 shells out to nmcli (NetworkManager) and wpctl (PipeWire);
// battery comes from /sys/class/power_supply. Replacing the shell-outs with
// native D-Bus (godbus + gonetworkmanager) is a contained later improvement.
type linuxDriver struct{}

func newDriver() Driver { return &linuxDriver{} }

func (d *linuxDriver) Status() Status {
	host, _ := os.Hostname()
	return Status{
		Hostname: host,
		Platform: "linux",
		Wifi:     d.wifiStatus(),
		Battery:  d.battery(),
		Volume:   d.volume(),
	}
}

func (d *linuxDriver) hasWifiDevice() bool {
	out, err := exec.Command("nmcli", "-t", "-f", "TYPE", "dev").Output()
	return err == nil && strings.Contains(string(out), "wifi")
}

func (d *linuxDriver) wifiStatus() WifiStatus {
	if !d.hasWifiDevice() {
		return WifiStatus{Available: false}
	}
	out, err := exec.Command("nmcli", "-t", "-f", "ACTIVE,SSID,SIGNAL", "dev", "wifi").Output()
	if err != nil {
		return WifiStatus{Available: false}
	}
	st := WifiStatus{Available: true}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) == 3 && parts[0] == "yes" {
			st.Connected = true
			st.SSID = parts[1]
			st.Signal, _ = strconv.Atoi(parts[2])
			break
		}
	}
	return st
}

func (d *linuxDriver) battery() BatteryStatus {
	matches, _ := filepath.Glob("/sys/class/power_supply/BAT*/capacity")
	if len(matches) == 0 {
		return BatteryStatus{Available: false}
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return BatteryStatus{Available: false}
	}
	pct, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	status, _ := os.ReadFile(filepath.Join(filepath.Dir(matches[0]), "status"))
	return BatteryStatus{
		Available: true,
		Charging:  strings.TrimSpace(string(status)) == "Charging",
		Percent:   pct,
	}
}

func (d *linuxDriver) volume() VolumeStatus {
	out, err := exec.Command("wpctl", "get-volume", "@DEFAULT_AUDIO_SINK@").Output()
	if err != nil {
		return VolumeStatus{}
	}
	// "Volume: 0.55 [MUTED]"
	fields := strings.Fields(string(out))
	v := VolumeStatus{Muted: strings.Contains(string(out), "MUTED")}
	if len(fields) >= 2 {
		if f, err := strconv.ParseFloat(fields[1], 64); err == nil {
			v.Percent = int(f*100 + 0.5)
		}
	}
	return v
}

func (d *linuxDriver) WifiNetworks() ([]WifiNetwork, error) {
	out, err := exec.Command("nmcli", "-t", "-f", "ACTIVE,SSID,SIGNAL,SECURITY", "dev", "wifi", "list", "--rescan", "auto").Output()
	if err != nil {
		return nil, fmt.Errorf("nmcli unavailable: %w", err)
	}
	known := d.knownConnections()
	seen := map[string]bool{}
	var nets []WifiNetwork
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 4 || parts[1] == "" || seen[parts[1]] {
			continue
		}
		seen[parts[1]] = true
		signal, _ := strconv.Atoi(parts[2])
		nets = append(nets, WifiNetwork{
			SSID:    parts[1],
			Signal:  signal,
			Secured: parts[3] != "" && parts[3] != "--",
			Known:   known[parts[1]],
			Active:  parts[0] == "yes",
		})
	}
	return nets, nil
}

func (d *linuxDriver) knownConnections() map[string]bool {
	out, err := exec.Command("nmcli", "-t", "-f", "NAME", "connection", "show").Output()
	known := map[string]bool{}
	if err != nil {
		return known
	}
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		known[name] = true
	}
	return known
}

func (d *linuxDriver) WifiConnect(ssid, password string) error {
	if ssid == "" {
		return fmt.Errorf("missing ssid")
	}
	args := []string{"dev", "wifi", "connect", ssid}
	if password != "" {
		args = append(args, "password", password)
	}
	out, err := exec.Command("nmcli", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (d *linuxDriver) SetVolume(percent int) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("volume out of range")
	}
	return exec.Command("wpctl", "set-volume", "@DEFAULT_AUDIO_SINK@",
		fmt.Sprintf("%d%%", percent)).Run()
}

func (d *linuxDriver) Screenshot(dir string) (string, error) {
	path := filepath.Join(dir, "screen-"+time.Now().Format("20060102-150405")+".png")
	out, err := exec.Command("grim", path).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("grim: %s", strings.TrimSpace(string(out)))
	}
	return path, nil
}

func (d *linuxDriver) Metrics() Metrics {
	m := Metrics{Processes: []ProcInfo{}}
	m.CPUPercent = linuxCPUPercent()

	// memory from /proc/meminfo (kB)
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		var total, avail int
		for _, line := range strings.Split(string(data), "\n") {
			f := strings.Fields(line)
			if len(f) >= 2 {
				v, _ := strconv.Atoi(f[1])
				switch f[0] {
				case "MemTotal:":
					total = v
				case "MemAvailable:":
					avail = v
				}
			}
		}
		m.MemTotalMB = total / 1024
		m.MemUsedMB = (total - avail) / 1024
	}

	used, tot := diskUsage("/")
	m.DiskUsedGB, m.DiskTotalGB = used, tot

	if data, err := os.ReadFile("/proc/uptime"); err == nil {
		var secs float64
		fmt.Sscanf(string(data), "%f", &secs)
		m.Uptime = "up " + humanDuration(time.Duration(secs)*time.Second)
	}
	if data, err := os.ReadFile("/proc/loadavg"); err == nil {
		f := strings.Fields(string(data))
		if len(f) >= 3 {
			m.Load = strings.Join(f[:3], " ")
		}
	}
	m.Processes = topProcesses()
	return m
}

func linuxCPUPercent() float64 {
	read := func() (idle, total uint64) {
		data, err := os.ReadFile("/proc/stat")
		if err != nil {
			return 0, 0
		}
		line := strings.SplitN(string(data), "\n", 2)[0]
		f := strings.Fields(line)
		for i, v := range f[1:] {
			n, _ := strconv.ParseUint(v, 10, 64)
			total += n
			if i == 3 { // idle
				idle = n
			}
		}
		return idle, total
	}
	i1, t1 := read()
	time.Sleep(200 * time.Millisecond)
	i2, t2 := read()
	if t2 == t1 {
		return 0
	}
	return (1 - float64(i2-i1)/float64(t2-t1)) * 100
}

func topProcesses() []ProcInfo {
	out, err := exec.Command("ps", "-eo", "pid,comm,%cpu,rss", "--sort=-%cpu").Output()
	if err != nil {
		return []ProcInfo{}
	}
	return parsePS(string(out), 6)
}

func (d *linuxDriver) Lock() error {
	return exec.Command("swaylock", "-f", "--color", "0b0e13").Start()
}

func (d *linuxDriver) Updates() UpdateInfo {
	info := UpdateInfo{Packages: []string{}}
	out, err := exec.Command("apt", "list", "--upgradable").Output()
	if err != nil {
		return info
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		name, _, ok := strings.Cut(line, "/")
		if !ok || strings.HasPrefix(line, "Listing") {
			continue
		}
		info.Packages = append(info.Packages, name)
	}
	info.Count = len(info.Packages)
	return info
}
