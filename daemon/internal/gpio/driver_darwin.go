//go:build darwin

package gpio

import (
	"fmt"
	"sort"
	"sync"
)

// mockDriver is the macOS dev-loop stand-in: an in-memory bank of lines so the
// Ghost tools work and can be exercised without real hardware. It models a Pi
// 400's user-facing BCM lines (2..27) and keeps their held output state.
type mockDriver struct {
	mu     sync.Mutex
	output map[int]bool
	value  map[int]int
}

func newDriver() Driver {
	return &mockDriver{output: map[int]bool{}, value: map[int]int{}}
}

// Available is true so the tools are usable in dev; callers can tell it's a
// mock from the line consumer ("mock").
func (d *mockDriver) Available() bool { return true }

func (d *mockDriver) Lines() ([]Line, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	var lines []Line
	for off := 2; off <= 27; off++ {
		l := Line{Offset: off, Name: fmt.Sprintf("GPIO%d", off)}
		if d.output[off] {
			l.Used, l.Output, l.Consumer, l.Value = true, true, "mock", d.value[off]
		}
		lines = append(lines, l)
	}
	sort.Slice(lines, func(a, b int) bool { return lines[a].Offset < lines[b].Offset })
	return lines, nil
}

func (d *mockDriver) Read(off int) (int, error) {
	if off < 2 || off > 27 {
		return 0, fmt.Errorf("no GPIO line %d", off)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.value[off], nil
}

func (d *mockDriver) Set(off, value int) error {
	if off < 2 || off > 27 {
		return fmt.Errorf("no GPIO line %d", off)
	}
	if value != 0 {
		value = 1
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.output[off] = true
	d.value[off] = value
	return nil
}

func (d *mockDriver) Release(off int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.output, off)
	delete(d.value, off)
	return nil
}
