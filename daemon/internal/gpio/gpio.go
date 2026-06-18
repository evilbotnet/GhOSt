// Package gpio exposes the board's GPIO lines behind a driver interface, so
// Ghost (and apps) can read inputs and drive outputs — "blink the LED when CI
// goes green", "is the button pressed?". The Linux driver shells out to
// libgpiod (gpioinfo/gpioget/gpioset), present on Raspberry Pi OS; the Darwin
// driver is an in-memory mock for the macOS dev loop and any board without GPIO
// (the VM), so the tools always exist and degrade gracefully.
//
// Lines are addressed by BCM offset on the default chip (gpiochip0 on the Pi
// 400). Outputs are *held* — a set stays asserted until changed or released —
// which is the intuitive behaviour and what blinking needs.
package gpio

type Line struct {
	Offset   int    `json:"offset"`
	Name     string `json:"name"`
	Consumer string `json:"consumer,omitempty"`
	Used     bool   `json:"used"`
	Output   bool   `json:"output"`
	Value    int    `json:"value"`
}

type Driver interface {
	Available() bool
	Lines() ([]Line, error)
	Read(offset int) (int, error)
	Set(offset, value int) error // drive as a held output
	Release(offset int) error    // hand the line back (revert to input)
}

type GPIO struct{ driver Driver }

func New() *GPIO { return &GPIO{driver: newDriver()} }

func (g *GPIO) Available() bool          { return g.driver.Available() }
func (g *GPIO) Lines() ([]Line, error)   { return g.driver.Lines() }
func (g *GPIO) Read(off int) (int, error) { return g.driver.Read(off) }
func (g *GPIO) Set(off, v int) error     { return g.driver.Set(off, v) }
func (g *GPIO) Release(off int) error    { return g.driver.Release(off) }
