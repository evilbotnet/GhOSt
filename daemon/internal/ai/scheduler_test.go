package ai

import (
	"testing"
	"time"
)

func TestScheduleNextRun(t *testing.T) {
	base := time.Date(2026, 6, 17, 9, 30, 0, 0, time.UTC)

	// Interval: next run is base + duration.
	iv := Schedule{Every: "30m"}
	iv.schedule(base)
	if iv.NextRun == nil || !iv.NextRun.Equal(base.Add(30*time.Minute)) {
		t.Fatalf("interval next = %v, want %v", iv.NextRun, base.Add(30*time.Minute))
	}

	// Too-short interval is rejected (guards against a busy-loop).
	bad := Schedule{Every: "10s"}
	bad.schedule(base)
	if bad.NextRun != nil {
		t.Fatalf("sub-minute interval should not schedule, got %v", bad.NextRun)
	}

	// Daily, time later today → today.
	later := Schedule{At: "18:00"}
	later.schedule(base)
	want := time.Date(2026, 6, 17, 18, 0, 0, 0, time.UTC)
	if later.NextRun == nil || !later.NextRun.Equal(want) {
		t.Fatalf("daily(later) = %v, want %v", later.NextRun, want)
	}

	// Daily, time already passed today → tomorrow.
	passed := Schedule{At: "08:00"}
	passed.schedule(base)
	want = time.Date(2026, 6, 18, 8, 0, 0, 0, time.UTC)
	if passed.NextRun == nil || !passed.NextRun.Equal(want) {
		t.Fatalf("daily(passed) = %v, want %v", passed.NextRun, want)
	}
}
