// Package system abstracts platform state (Wi-Fi, audio, battery) behind a
// driver interface. The Linux driver talks to NetworkManager/PipeWire/UPower;
// the Darwin driver is the macOS dev-loop stand-in (partly mocked).
package system

import (
	"os"
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

type Driver interface {
	Status() Status
	WifiNetworks() ([]WifiNetwork, error)
	WifiConnect(ssid, password string) error
	SetVolume(percent int) error
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

// PublishLoop pushes status to the `system` topic so the tray stays live.
func (s *System) PublishLoop(hub *ws.Hub, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for range t.C {
		hub.Publish("system", "status", s.Status())
	}
}
