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

	// Collect relative paths first, then sort for deterministic order.
	var files []string
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
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to walk directory %s: %w", dir, err)
	}

	sort.Strings(files)

	// Stream each file into the hash to avoid holding all content in memory.
	for _, rel := range files {
		_, _ = io.WriteString(h, rel)
		f, err := os.Open(filepath.Join(dir, rel))
		if err != nil {
			return "", fmt.Errorf("failed to open %s: %w", rel, err)
		}
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return "", fmt.Errorf("failed to hash %s: %w", rel, err)
		}
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
