package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMemoryRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Empty content is rejected.
	if _, err := SaveMemory("x", "", "  "); err == nil {
		t.Fatal("expected error for empty content")
	}

	m, err := SaveMemory("Preferred Editor", "editor choice", "Prefers the built-in Editor over vim.")
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if m.Name != "Preferred Editor" {
		t.Fatalf("name = %q", m.Name)
	}
	// Stored under a slugified filename.
	if _, err := os.Stat(filepath.Join(MemoryDir(), "preferred-editor.md")); err != nil {
		t.Fatalf("slug file missing: %v", err)
	}

	mems := LoadMemories()
	if len(mems) != 1 || mems[0].Body != "Prefers the built-in Editor over vim." {
		t.Fatalf("load: %+v", mems)
	}

	// The full body is injected into the prompt (not gated behind recall).
	sec := memoryPromptSection(mems)
	if !strings.Contains(sec, "Prefers the built-in Editor over vim.") {
		t.Fatalf("prompt section missing body: %q", sec)
	}

	// Same name updates in place (no duplicate).
	if _, err := SaveMemory("Preferred Editor", "", "Now prefers vim."); err != nil {
		t.Fatal(err)
	}
	mems = LoadMemories()
	if len(mems) != 1 || mems[0].Body != "Now prefers vim." {
		t.Fatalf("update failed: %+v", mems)
	}

	// Forget removes it; forgetting a missing memory is a no-op.
	if err := DeleteMemory("Preferred Editor"); err != nil {
		t.Fatal(err)
	}
	if len(LoadMemories()) != 0 {
		t.Fatal("memory not deleted")
	}
	if err := DeleteMemory("nonexistent"); err != nil {
		t.Fatalf("deleting missing memory should be a no-op, got %v", err)
	}

	// Empty prompt section when there are no memories.
	if memoryPromptSection(nil) != "" {
		t.Fatal("expected empty section for no memories")
	}
}
