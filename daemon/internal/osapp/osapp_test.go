package osapp

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func makePkg(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

const manifestJSON = `{
  "id": "tone.studio", "name": "Tone Studio", "version": "0.1.0",
  "entry": "index.html", "icon": "icon.svg",
  "permissions": ["fs:home:rw", "system:read"]
}`

func TestInstallAndScopes(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s := New()

	pkg := makePkg(t, map[string]string{
		"tone.studio/manifest.json": manifestJSON,
		"tone.studio/index.html":    "<h1>Tone</h1>",
	})
	sum := sha256.Sum256(pkg)
	sha := hex.EncodeToString(sum[:])

	// Wrong hash is rejected.
	if _, err := s.Install(pkg, "deadbeef", []string{"fs:home:rw"}); err == nil {
		t.Fatal("expected hash mismatch error")
	}

	// Granting a scope the app never requested is rejected.
	if _, err := s.Install(pkg, sha, []string{"term:exec"}); err == nil {
		t.Fatal("expected error granting un-requested scope")
	}

	// Correct install: granted subset of requested.
	inst, err := s.Install(pkg, sha, []string{"fs:home:rw"})
	if err != nil {
		t.Fatalf("install: %v", err)
	}
	if inst.ID != "tone.studio" || !inst.Enabled {
		t.Fatalf("bad install result: %+v", inst)
	}
	if _, err := os.Stat(filepath.Join(s.Dir("tone.studio"), "index.html")); err != nil {
		t.Fatalf("entry not unpacked: %v", err)
	}

	// Scopes are enforced with implication: rw implies ro.
	scopes, ok := s.ScopesFor("tone.studio")
	if !ok {
		t.Fatal("ScopesFor missing")
	}
	if !Allows(scopes, "fs:home:ro") {
		t.Fatal("fs:home:rw should imply fs:home:ro")
	}
	if Allows(scopes, "term:exec") {
		t.Fatal("term:exec must not be allowed")
	}

	// Listed and then uninstalled.
	if got := s.List(); len(got) != 1 {
		t.Fatalf("expected 1 app, got %d", len(got))
	}
	if err := s.Uninstall("tone.studio"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.Dir("tone.studio")); !os.IsNotExist(err) {
		t.Fatal("app dir should be gone after uninstall")
	}
}

func TestZipSlipRejected(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s := New()
	pkg := makePkg(t, map[string]string{
		"evil.app/manifest.json":      `{"id":"evil.app","version":"1","entry":"index.html","permissions":[]}`,
		"evil.app/index.html":         "x",
		"evil.app/../../../etc/pwned": "owned",
	})
	if _, err := s.Install(pkg, "", nil); err == nil {
		t.Fatal("expected zip-slip path to be rejected")
	}
}

func TestParseManifestValidation(t *testing.T) {
	// Bad id (no dot) is rejected.
	bad := makePkg(t, map[string]string{"manifest.json": `{"id":"notrevdns","version":"1","entry":"i.html","permissions":[]}`})
	if _, err := ParseManifest(bad); err == nil {
		t.Fatal("expected invalid-id error")
	}
	// Unknown permission is rejected.
	badPerm := makePkg(t, map[string]string{"manifest.json": `{"id":"a.b","version":"1","entry":"i.html","permissions":["root:everything"]}`})
	if _, err := ParseManifest(badPerm); err == nil {
		t.Fatal("expected unknown-permission error")
	}
	// Good manifest parses.
	ok := makePkg(t, map[string]string{"a.b/manifest.json": `{"id":"a.b","version":"1","entry":"i.html","permissions":["notify"]}`})
	m, err := ParseManifest(ok)
	if err != nil || m.ID != "a.b" {
		t.Fatalf("parse good manifest: %v %+v", err, m)
	}
}
