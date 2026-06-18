//go:build linux

package gpio

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// libgpiodDriver shells out to libgpiod's CLI (gpioinfo/gpioget/gpioset). It's
// dependency-light (no cgo), and the tools ship on Raspberry Pi OS.
//
// Holding an output: gpioset releases the line when it exits, so a one-shot
// can't keep an LED lit. We instead keep a long-lived `gpioset --mode=signal`
// process per asserted line and replace it on the next Set — giving persistent,
// intuitive output that survives until changed or released.
type libgpiodDriver struct {
	chip string
	mu   sync.Mutex
	held map[int]*exec.Cmd // offset -> running gpioset holding it high/low
}

func newDriver() Driver {
	return &libgpiodDriver{chip: "gpiochip0", held: map[int]*exec.Cmd{}}
}

func (d *libgpiodDriver) Available() bool {
	_, err := exec.LookPath("gpioinfo")
	return err == nil
}

func (d *libgpiodDriver) Lines() ([]Line, error) {
	out, err := exec.Command("gpioinfo", d.chip).Output()
	if err != nil {
		return nil, fmt.Errorf("gpioinfo: %w", err)
	}
	var lines []Line
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		// e.g.: 	line  17:      "GPIO17"       "ghostd"  output  active-high
		f := strings.Fields(sc.Text())
		if len(f) < 3 || f[0] != "line" {
			continue
		}
		off, err := strconv.Atoi(strings.TrimSuffix(f[1], ":"))
		if err != nil {
			continue
		}
		l := Line{Offset: off, Name: strings.Trim(f[2], `"`)}
		rest := sc.Text()
		l.Used = strings.Contains(rest, "[used]") || strings.Contains(rest, "\"ghostd\"")
		l.Output = strings.Contains(rest, "output")
		lines = append(lines, l)
	}
	d.mu.Lock()
	for off := range d.held {
		for i := range lines {
			if lines[i].Offset == off {
				lines[i].Output, lines[i].Used = true, true
			}
		}
	}
	d.mu.Unlock()
	return lines, nil
}

func (d *libgpiodDriver) Read(off int) (int, error) {
	out, err := exec.Command("gpioget", d.chip, strconv.Itoa(off)).Output()
	if err != nil {
		return 0, fmt.Errorf("gpioget: %w", err)
	}
	if strings.TrimSpace(string(out)) == "1" {
		return 1, nil
	}
	return 0, nil
}

func (d *libgpiodDriver) Set(off, value int) error {
	if value != 0 {
		value = 1
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if old := d.held[off]; old != nil && old.Process != nil {
		old.Process.Kill()
		old.Wait()
		delete(d.held, off)
	}
	// --mode=signal holds the line until the process is signalled/killed.
	cmd := exec.Command("gpioset", "--mode=signal", d.chip, fmt.Sprintf("%d=%d", off, value))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("gpioset: %w", err)
	}
	d.held[off] = cmd
	return nil
}

func (d *libgpiodDriver) Release(off int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if cmd := d.held[off]; cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
		cmd.Wait()
		delete(d.held, off)
	}
	return nil
}
