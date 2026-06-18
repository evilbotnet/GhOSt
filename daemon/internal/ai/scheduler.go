package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/ghostos/ghostd/internal/ws"
)

// Scheduler runs proactive Ghost (ADR 0007): named prompts that fire on an
// interval or at a daily time, each producing a read-only Ghost run whose final
// message is delivered as a desktop notification. "Every morning, summarise
// what changed in Downloads"; "every 30 minutes, warn me if disk is over 90%".
//
// Schedules persist to ~/.config/ghost/schedules.json so they survive restarts.
// The loop ticks once a minute and fires anything now due.

type Schedule struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Prompt   string `json:"prompt"`
	Enabled  bool   `json:"enabled"`
	Notify   bool   `json:"notify"` // raise a desktop notification with the result

	// Exactly one cadence is set:
	Every string `json:"every,omitempty"` // Go duration, e.g. "30m", "6h"
	At    string `json:"at,omitempty"`    // daily local time "HH:MM"

	LastRun    *time.Time `json:"lastRun,omitempty"`
	NextRun    *time.Time `json:"nextRun,omitempty"`
	LastResult string     `json:"lastResult,omitempty"`
}

type Scheduler struct {
	ghost *Ghost
	hub   *ws.Hub

	mu   sync.Mutex
	list []Schedule
	stop chan struct{}
}

func NewScheduler(ghost *Ghost, hub *ws.Hub) *Scheduler {
	s := &Scheduler{ghost: ghost, hub: hub, stop: make(chan struct{})}
	s.load()
	now := time.Now()
	for i := range s.list {
		if s.list[i].NextRun == nil {
			s.list[i].schedule(now)
		}
	}
	s.save()
	return s
}

func schedulesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ghost", "schedules.json")
}

func (s *Scheduler) load() {
	data, err := os.ReadFile(schedulesPath())
	if err != nil {
		return
	}
	json.Unmarshal(data, &s.list)
}

func (s *Scheduler) save() {
	path := schedulesPath()
	os.MkdirAll(filepath.Dir(path), 0o755)
	data, _ := json.MarshalIndent(s.list, "", "  ")
	tmp := path + ".tmp"
	if os.WriteFile(tmp, data, 0o600) == nil {
		os.Rename(tmp, path)
	}
}

// schedule computes NextRun from the cadence relative to `from`.
func (sc *Schedule) schedule(from time.Time) {
	switch {
	case sc.Every != "":
		d, err := time.ParseDuration(sc.Every)
		if err != nil || d < time.Minute {
			sc.NextRun = nil
			return
		}
		next := from.Add(d)
		sc.NextRun = &next
	case sc.At != "":
		t, err := time.Parse("15:04", sc.At)
		if err != nil {
			sc.NextRun = nil
			return
		}
		next := time.Date(from.Year(), from.Month(), from.Day(), t.Hour(), t.Minute(), 0, 0, from.Location())
		if !next.After(from) {
			next = next.Add(24 * time.Hour)
		}
		sc.NextRun = &next
	default:
		sc.NextRun = nil
	}
}

// Start runs the tick loop until Stop. Call in a goroutine.
func (s *Scheduler) Start() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.tick(time.Now())
		}
	}
}

func (s *Scheduler) Stop() { close(s.stop) }

func (s *Scheduler) tick(now time.Time) {
	s.mu.Lock()
	var due []int
	for i := range s.list {
		sc := &s.list[i]
		if sc.Enabled && sc.NextRun != nil && !sc.NextRun.After(now) {
			due = append(due, i)
		}
	}
	// Snapshot prompts/ids so we don't hold the lock across the LLM run.
	type job struct {
		idx          int
		id, name, pr string
		notify       bool
	}
	var jobs []job
	for _, i := range due {
		sc := &s.list[i]
		jobs = append(jobs, job{i, sc.ID, sc.Name, sc.Prompt, sc.Notify})
		sc.schedule(now) // reschedule immediately so a slow run can't double-fire
		sc.LastRun = ptr(now)
	}
	if len(jobs) > 0 {
		s.save()
	}
	s.mu.Unlock()

	for _, j := range jobs {
		go s.fire(j.idx, j.id, j.name, j.pr, j.notify)
	}
}

func (s *Scheduler) fire(idx int, id, name, prompt string, notify bool) {
	result := s.ghost.RunScheduled(prompt)

	s.mu.Lock()
	// Re-find by id (indices may have shifted via CRUD during the run).
	for i := range s.list {
		if s.list[i].ID == id {
			s.list[i].LastResult = result
			break
		}
	}
	s.save()
	s.mu.Unlock()

	if notify && result != "" {
		s.hub.Publish("notify", "show", map[string]string{
			"title": name, "body": result, "kind": "ghost",
		})
	}
	s.hub.Publish("schedules", "fired", map[string]string{"id": id, "result": result})
}

// --- CRUD (the shell + setup drive these) ---

func (s *Scheduler) List() []Schedule {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Schedule, len(s.list))
	copy(out, s.list)
	sort.Slice(out, func(a, b int) bool { return out[a].Name < out[b].Name })
	return out
}

// Save adds a new schedule (blank ID) or updates an existing one by ID, then
// recomputes its next run. Returns the stored schedule.
func (s *Scheduler) Save(in Schedule) Schedule {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if in.ID == "" {
		in.ID = newID()
		in.schedule(now)
		s.list = append(s.list, in)
		s.save()
		return in
	}
	for i := range s.list {
		if s.list[i].ID == in.ID {
			in.LastRun = s.list[i].LastRun
			in.LastResult = s.list[i].LastResult
			in.schedule(now)
			s.list[i] = in
			s.save()
			return in
		}
	}
	// Unknown ID: treat as new with the given ID.
	in.schedule(now)
	s.list = append(s.list, in)
	s.save()
	return in
}

func (s *Scheduler) Remove(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.list[:0]
	for _, sc := range s.list {
		if sc.ID != id {
			out = append(out, sc)
		}
	}
	s.list = out
	s.save()
}

// RunNow fires a schedule immediately (out of band) and returns its result.
func (s *Scheduler) RunNow(id string) (string, bool) {
	s.mu.Lock()
	var sc *Schedule
	for i := range s.list {
		if s.list[i].ID == id {
			sc = &s.list[i]
			break
		}
	}
	if sc == nil {
		s.mu.Unlock()
		return "", false
	}
	name, prompt, notify := sc.Name, sc.Prompt, sc.Notify
	sc.LastRun = ptr(time.Now())
	s.save()
	s.mu.Unlock()

	result := s.ghost.RunScheduled(prompt)
	s.mu.Lock()
	for i := range s.list {
		if s.list[i].ID == id {
			s.list[i].LastResult = result
			break
		}
	}
	s.save()
	s.mu.Unlock()
	if notify && result != "" {
		s.hub.Publish("notify", "show", map[string]string{"title": name, "body": result, "kind": "ghost"})
	}
	return result, true
}

func ptr(t time.Time) *time.Time { return &t }
