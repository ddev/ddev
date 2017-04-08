package files

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

// Untargz accepts a tar gz file and extracts the contents to the provided destination path.
func Untargz(source string, dest string) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer f.Close()

	gf, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	tf := tar.NewReader(gf)

	for {
		file, err := tf.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if file.Typeflag == tar.TypeDir {
			os.Mkdir(path.Join(dest, file.Name), 0755)
		} else {
			exFile, err := os.Create(path.Join(dest, file.Name))
			if err != nil {
				return err
			}
			defer exFile.Close()
			io.Copy(exFile, tf)
		}
	}

	return nil
}

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
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if e := out.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	err = out.Sync()
	if err != nil {
		return err
	}

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	err = os.Chmod(dst, si.Mode())
	if err != nil {
		return err
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
