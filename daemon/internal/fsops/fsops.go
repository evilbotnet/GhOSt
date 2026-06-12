// Package fsops implements home-confined filesystem operations for the shell.
// Every path is canonicalized and must resolve inside one of the allowed
// roots; deletes go to a trash directory instead of unlinking.
package fsops

import (
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var ErrOutsideRoot = errors.New("path outside allowed roots")

type Entry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Dir      bool   `json:"dir"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
	Mime     string `json:"mime"`
}

type FS struct {
	roots []string
	home  string
}

func New(roots []string) *FS {
	clean := make([]string, 0, len(roots))
	for _, r := range roots {
		if resolved, err := filepath.EvalSymlinks(r); err == nil {
			clean = append(clean, resolved)
		}
	}
	home, _ := os.UserHomeDir()
	return &FS{roots: clean, home: home}
}

func (f *FS) Home() string { return f.home }

// resolve canonicalizes p and ensures it is inside an allowed root.
// For paths that don't exist yet (write/mkdir targets) the parent must exist
// and be inside a root.
func (f *FS) resolve(p string) (string, error) {
	if p == "" || p == "~" {
		return f.home, nil
	}
	if strings.HasPrefix(p, "~/") {
		p = filepath.Join(f.home, p[2:])
	}
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("path must be absolute: %q", p)
	}
	p = filepath.Clean(p)

	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", err
		}
		parent, err2 := filepath.EvalSymlinks(filepath.Dir(p))
		if err2 != nil {
			return "", err2
		}
		resolved = filepath.Join(parent, filepath.Base(p))
	}
	for _, root := range f.roots {
		if resolved == root || strings.HasPrefix(resolved, root+string(filepath.Separator)) {
			return resolved, nil
		}
	}
	return "", ErrOutsideRoot
}

func (f *FS) List(p string) (string, []Entry, error) {
	path, err := f.resolve(p)
	if err != nil {
		return "", nil, err
	}
	items, err := os.ReadDir(path)
	if err != nil {
		return "", nil, err
	}
	entries := make([]Entry, 0, len(items))
	for _, it := range items {
		if strings.HasPrefix(it.Name(), ".") {
			continue // hidden files: revisit with a toggle later
		}
		info, err := it.Info()
		if err != nil {
			continue
		}
		entries = append(entries, Entry{
			Name:     it.Name(),
			Path:     filepath.Join(path, it.Name()),
			Dir:      it.IsDir(),
			Size:     info.Size(),
			Modified: info.ModTime().Format(time.RFC3339),
			Mime:     mimeFor(it.Name(), it.IsDir()),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Dir != entries[j].Dir {
			return entries[i].Dir
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return path, entries, nil
}

const maxReadSize = 8 << 20 // editor is for text files, not media

func (f *FS) Read(p string) ([]byte, error) {
	path, err := f.resolve(p)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxReadSize {
		return nil, fmt.Errorf("file too large to open in the editor (%d bytes)", info.Size())
	}
	return os.ReadFile(path)
}

func (f *FS) Write(p string, content []byte) error {
	path, err := f.resolve(p)
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func (f *FS) Mkdir(p string) error {
	path, err := f.resolve(p)
	if err != nil {
		return err
	}
	return os.Mkdir(path, 0o755)
}

func (f *FS) Rename(from, to string) error {
	src, err := f.resolve(from)
	if err != nil {
		return err
	}
	dst, err := f.resolve(to)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("destination already exists: %s", dst)
	}
	return os.Rename(src, dst)
}

func (f *FS) Trash(p string) error {
	path, err := f.resolve(p)
	if err != nil {
		return err
	}
	trashDir := filepath.Join(f.home, ".openos-trash")
	if err := os.MkdirAll(trashDir, 0o700); err != nil {
		return err
	}
	dest := filepath.Join(trashDir,
		fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405"), filepath.Base(path)))
	return os.Rename(path, dest)
}

func mimeFor(name string, dir bool) string {
	if dir {
		return "inode/directory"
	}
	ext := strings.ToLower(filepath.Ext(name))
	if m := mime.TypeByExtension(ext); m != "" {
		return m
	}
	switch ext {
	case ".go", ".rs", ".py", ".rb", ".sh", ".zsh", ".toml", ".yaml", ".yml",
		".ts", ".tsx", ".jsx", ".svelte", ".vue", ".c", ".h", ".cpp", ".lock",
		".conf", ".ini", ".env", ".gitignore", ".sql", "":
		return "text/plain"
	}
	return "application/octet-stream"
}
