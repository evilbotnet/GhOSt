package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExportImportRoundTrip(t *testing.T) {
	// Source home with some GhOSt state.
	src := t.TempDir()
	t.Setenv("HOME", src)
	write := func(rel, content string) {
		p := filepath.Join(src, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(content), 0o600)
	}
	write(".config/ghost/ai.toml", "[ai]\nenabled = true\n")
	write(".config/ghost/memory/preferred-units.md", "metric")
	write(".local/share/ghost/apps/tone.studio/manifest.json", `{"id":"tone.studio"}`)
	write(".config/other-app/secret", "should NOT be backed up") // outside GhOSt dirs

	var buf bytes.Buffer
	if err := Export(&buf); err != nil {
		t.Fatalf("export: %v", err)
	}

	// The archive must contain GhOSt files and exclude unrelated ones.
	names := tarNames(t, buf.Bytes())
	if !names[".config/ghost/ai.toml"] || !names[".config/ghost/memory/preferred-units.md"] ||
		!names[".local/share/ghost/apps/tone.studio/manifest.json"] {
		t.Fatalf("missing expected entries: %v", names)
	}
	for n := range names {
		if filepath.HasPrefix(n, ".config/other-app") {
			t.Fatalf("backup leaked a non-GhOSt path: %s", n)
		}
	}

	// Restore into a fresh home.
	dst := t.TempDir()
	t.Setenv("HOME", dst)
	if err := Import(bytes.NewReader(buf.Bytes())); err != nil {
		t.Fatalf("import: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dst, ".config/ghost/ai.toml"))
	if err != nil || string(got) != "[ai]\nenabled = true\n" {
		t.Fatalf("restored ai.toml = %q, err %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dst, ".local/share/ghost/apps/tone.studio/manifest.json")); err != nil {
		t.Fatalf("osapp not restored: %v", err)
	}
}

func TestImportRejectsTraversal(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	// Craft a malicious archive with a path-traversal entry.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	body := []byte("pwned")
	tw.WriteHeader(&tar.Header{Name: "../../../etc/ghost-pwned", Mode: 0o600, Size: int64(len(body)), Typeflag: tar.TypeReg})
	tw.Write(body)
	tw.Close()
	gz.Close()

	if err := Import(bytes.NewReader(buf.Bytes())); err == nil {
		t.Fatal("expected traversal entry to be rejected")
	}
}

func tarNames(t *testing.T, data []byte) map[string]bool {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	names := map[string]bool{}
	for {
		h, err := tr.Next()
		if err != nil {
			break
		}
		names[h.Name] = true
	}
	return names
}
