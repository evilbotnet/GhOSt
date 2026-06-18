package gpio

import "testing"

func TestMockSetReadRelease(t *testing.T) {
	g := New()
	if !g.Available() {
		t.Fatal("mock should report available")
	}

	if err := g.Set(17, 1); err != nil {
		t.Fatalf("set: %v", err)
	}
	if v, _ := g.Read(17); v != 1 {
		t.Fatalf("read after set high = %d, want 1", v)
	}

	if err := g.Set(17, 0); err != nil {
		t.Fatalf("set low: %v", err)
	}
	if v, _ := g.Read(17); v != 0 {
		t.Fatalf("read after set low = %d, want 0", v)
	}

	// The line should show as a held output in Lines().
	lines, _ := g.Lines()
	var found bool
	for _, l := range lines {
		if l.Offset == 17 {
			found = true
			if !l.Output {
				t.Fatal("GPIO17 should report as output")
			}
		}
	}
	if !found {
		t.Fatal("GPIO17 not listed")
	}

	if err := g.Release(17); err != nil {
		t.Fatalf("release: %v", err)
	}
	lines, _ = g.Lines()
	for _, l := range lines {
		if l.Offset == 17 && l.Output {
			t.Fatal("GPIO17 still output after release")
		}
	}

	// Out-of-range pins are rejected.
	if err := g.Set(99, 1); err == nil {
		t.Fatal("expected error for out-of-range pin")
	}
}
