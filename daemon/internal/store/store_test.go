package store

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ghostos/ghostd/internal/osapp"
)

func appPkg(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	mf, _ := zw.Create("tone.studio/manifest.json")
	mf.Write([]byte(`{"id":"tone.studio","name":"Tone","version":"0.1.0","entry":"index.html","permissions":["fs:home:rw"]}`))
	idx, _ := zw.Create("tone.studio/index.html")
	idx.Write([]byte("<h1>Tone</h1>"))
	zw.Close()
	return buf.Bytes()
}

func TestStoreSignedInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// Write the package and reference it (hash-pinned) from the index.
	pkg := appPkg(t)
	pkgPath := filepath.Join(home, "tone.osapp")
	os.WriteFile(pkgPath, pkg, 0o644)
	sum := sha256.Sum256(pkg)

	idx := Index{
		Generated: "2026-06-18",
		Entries: []Entry{{
			Type: "app", ID: "tone.studio", Name: "Tone", Version: "0.1.0",
			Download: "file://" + pkgPath, SHA256: hex.EncodeToString(sum[:]),
			Permissions: []string{osapp.ScopeFSWrite},
		}},
	}
	indexBytes, _ := json.Marshal(idx)
	indexPath := filepath.Join(home, "index.json")
	os.WriteFile(indexPath, indexBytes, 0o644)
	sig := ed25519.Sign(priv, indexBytes)
	os.WriteFile(indexPath+".sig", []byte(base64.StdEncoding.EncodeToString(sig)), 0o644)

	osapps := osapp.New()
	s := New(osapps)
	if err := s.Configure(Config{IndexURL: indexPath, PublicKey: base64.StdEncoding.EncodeToString(pub)}); err != nil {
		t.Fatal(err)
	}

	// Catalog verifies the signature and returns the entry.
	cat, err := s.Catalog()
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	if len(cat.Entries) != 1 || cat.Entries[0].ID != "tone.studio" {
		t.Fatalf("unexpected catalog: %+v", cat)
	}

	// Install downloads, hash-checks, and installs the app with the grant.
	if err := s.Install("tone.studio", []string{osapp.ScopeFSWrite}); err != nil {
		t.Fatalf("install: %v", err)
	}
	if _, ok := osapps.Get("tone.studio"); !ok {
		t.Fatal("app not installed")
	}

	// A different key must be rejected.
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	s.Configure(Config{IndexURL: indexPath, PublicKey: base64.StdEncoding.EncodeToString(otherPub)})
	if _, err := s.Catalog(); err == nil {
		t.Fatal("expected signature verification to fail with the wrong key")
	}

	// A tampered index (valid old signature, changed bytes) must be rejected.
	os.WriteFile(indexPath, append(indexBytes, ' '), 0o644)
	s.Configure(Config{IndexURL: indexPath, PublicKey: base64.StdEncoding.EncodeToString(pub)})
	if _, err := s.Catalog(); err == nil {
		t.Fatal("expected tampered index to fail verification")
	}
}
