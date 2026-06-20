// Package backup exports and restores all of GhOSt's own state — config,
// skills, tools, memory, soul, schedules, the installed .osapp packages and
// their grants, web apps, store config, settings — as a single .tar.gz. It is
// deliberately scoped to GhOSt's directories (~/.config/ghost and
// ~/.local/share/ghost), NOT the whole home, so a backup is small and portable;
// the user's documents are theirs to back up separately.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// roots are the GhOSt directories included in a backup, as paths relative to
// $HOME (so the archive is home-relative and restores anywhere).
func roots() []string {
	return []string{
		filepath.Join(".config", "ghost"),
		filepath.Join(".local", "share", "ghost"),
	}
}

func home() string {
	h, _ := os.UserHomeDir()
	return h
}

// Export writes a gzip-compressed tar of the GhOSt directories to w.
func Export(w io.Writer) error {
	gz := gzip.NewWriter(w)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	h := home()
	for _, rel := range roots() {
		base := filepath.Join(h, rel)
		if _, err := os.Stat(base); err != nil {
			continue // a dir that doesn't exist yet is simply absent from the backup
		}
		err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil // tar regular files only; dirs are implied by their paths
			}
			if !info.Mode().IsRegular() {
				return nil // skip symlinks/sockets
			}
			arcName, err := filepath.Rel(h, path)
			if err != nil {
				return err
			}
			hdr := &tar.Header{
				Name:    filepath.ToSlash(arcName),
				Mode:    int64(info.Mode().Perm()),
				Size:    info.Size(),
				ModTime: info.ModTime(),
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = io.Copy(tw, f)
			return err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Import restores a backup produced by Export into the user's home. Entries are
// confined to GhOSt's directories (path-traversal-safe); anything outside them
// is refused, so a malformed or hostile archive can't escape.
func Import(r io.Reader) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("not a gzip archive: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	h := home()
	allowed := roots()
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.Clean(filepath.FromSlash(hdr.Name))
		if filepath.IsAbs(name) || strings.HasPrefix(name, "..") {
			return fmt.Errorf("unsafe path in backup: %q", hdr.Name)
		}
		if !underAllowedRoot(name, allowed) {
			return fmt.Errorf("backup entry outside GhOSt directories: %q", hdr.Name)
		}
		dst := filepath.Join(h, name)
		if !strings.HasPrefix(dst, h+string(os.PathSeparator)) {
			return fmt.Errorf("path escapes home: %q", hdr.Name)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode)&0o777)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, io.LimitReader(tr, 256<<20))
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func underAllowedRoot(name string, allowed []string) bool {
	for _, root := range allowed {
		if name == root || strings.HasPrefix(name, root+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}
