package fileutil

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// HashDir returns a SHA-256 hash of a directory's contents.
// It walks the directory tree, hashing each file's relative path
// and contents. The result is deterministic regardless of walk order.
func HashDir(dir string) (string, error) {
	h := sha256.New()

	type fileEntry struct {
		rel     string
		content []byte
	}

	var entries []fileEntry
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path for %s: %w", path, err)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		entries = append(entries, fileEntry{rel: rel, content: content})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to walk directory %s: %w", dir, err)
	}

	// Sort by relative path for deterministic ordering
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].rel < entries[j].rel
	})

	for _, e := range entries {
		h.Write([]byte(e.rel))
		h.Write(e.content)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// HashDirs returns a SHA-256 hash of multiple directories' contents combined.
// Additional strings (e.g., image names) can be included via extraStrings
// to ensure changes to those values also produce a different hash.
func HashDirs(dirs []string, extraStrings ...string) (string, error) {
	h := sha256.New()

	for _, s := range extraStrings {
		_, _ = io.WriteString(h, s)
	}

	for _, dir := range dirs {
		// Skip non-existent directories with an empty marker
		if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
			_, _ = io.WriteString(h, dir+":empty")
			continue
		}
		dirHash, err := HashDir(dir)
		if err != nil {
			return "", err
		}
		_, _ = io.WriteString(h, dirHash)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
