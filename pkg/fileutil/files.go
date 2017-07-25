package fileutil

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"strings"

	"runtime"

	"github.com/drud/ddev/pkg/util"
)

// CopyFile copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file. The file mode will be copied from the source and
// the copied data is synced/flushed to stable storage. Credit @m4ng0squ4sh https://gist.github.com/m4ng0squ4sh/92462b38df26839a3ca324697c8cba04
func CopyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer util.CheckClose(in)
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Failed to create file %v, err: %v", src, err)
	}
	defer util.CheckClose(out)
	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("Failed to copy file from %v to %v err: %v", src, dst, err)
	}

	err = out.Sync()
	if err != nil {
		return err
	}

	// os.Chmod fails on long path (> 256 characters) on windows.
	// A description of this problem with golang is at https://github.com/golang/dep/issues/774#issuecomment-311560825
	// It could end up fixed in a future version of golang.
	if runtime.GOOS != "windows" {
		si, err := os.Stat(src)
		if err != nil {
			return err
		}

		err = os.Chmod(dst, si.Mode())
		if err != nil {
			return fmt.Errorf("Failed to chmod file %v to mode %v, err=%v", dst, si.Mode(), err)
		}
	}

	return nil
}

// CopyDir recursively copies a directory tree, attempting to preserve permissions.
// Source directory must exist, destination directory must *not* exist.
// Symlinks are ignored and skipped. Credit @m4ng0squ4sh https://gist.github.com/m4ng0squ4sh/92462b38df26839a3ca324697c8cba04
func CopyDir(src string, dst string) error {
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !si.IsDir() {
		return fmt.Errorf("source is not a directory")
	}

	_, err = os.Stat(dst)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		return fmt.Errorf("destination already exists")
	}

	err = os.MkdirAll(dst, si.Mode())
	if err != nil {
		return err
	}

	entries, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err = CopyFile(srcPath, dstPath)
			if err != nil && entry.Mode()&os.ModeSymlink != 0 {
				fmt.Printf("failed to copy symlink %s, skipping...\n", srcPath)
				continue
			}
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// FileExists checks a file's existence
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

// PurgeDirectory removes all of the contents of a given
// directory, leaving the directory itself intact.
func PurgeDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}

	defer util.CheckClose(dir)

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, file := range files {
		err = os.Chmod(filepath.Join(path, file), 0777)
		if err != nil {
			return err
		}
		err = os.RemoveAll(filepath.Join(path, file))
		if err != nil {
			return err
		}
	}
	return nil
}

// FgrepStringInFile is a small hammer for looking for a literal string in a file.
// It should only be used against very modest sized files, as the entire file is read
// into a string.
func FgrepStringInFile(fullPath string, needle string) (bool, error) {
	fullFileBytes, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return false, fmt.Errorf("Fail to open file %s, err:%v ", fullPath, err)
	}
	fullFileString := string(fullFileBytes)
	return strings.Contains(fullFileString, needle), nil
}
