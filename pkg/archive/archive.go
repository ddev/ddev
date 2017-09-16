package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/drud/ddev/pkg/util"
)

// Ungzip accepts a gzipped file and uncompresses it to the provided destination path.
func Ungzip(source string, dest string) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}

	defer func() {
		if e := f.Close(); e != nil {
			err = e
		}
	}()

	gf, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	defer func() {
		if e := gf.Close(); e != nil {
			err = e
		}
	}()

	fname := strings.TrimSuffix(filepath.Base(f.Name()), ".gz")
	exFile, err := os.Create(filepath.Join(dest, fname))
	if err != nil {
		return err
	}

	defer func() {
		if e := exFile.Close(); e != nil {
			err = e
		}
	}()

	_, err = io.Copy(exFile, gf)
	if err != nil {
		return err
	}

	err = exFile.Sync()
	if err != nil {
		return err
	}

	return nil

}

// Untar accepts a tar or tar.gz file and extracts the contents to the provided destination path.
// extractionDir is the path at which extraction should start; nothing will be extracted except the contents of
// extractionDir
func Untar(source string, dest string, extractionDir string) error {
	var tf *tar.Reader
	f, err := os.Open(source)
	if err != nil {
		return err
	}

	defer util.CheckClose(f)

	if strings.HasSuffix(source, "gz") {
		gf, err := gzip.NewReader(f)
		if err != nil {
			return err
		}

		defer util.CheckClose(gf)

		tf = tar.NewReader(gf)
	} else {
		tf = tar.NewReader(f)
	}

	for {
		file, err := tf.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Error during read of tar archive %v, err: %v", source, err)
		}

		// If we have an extractionDir and this doesn't match, skip it.
		if !strings.HasPrefix(file.Name, extractionDir) {
			continue
		}

		// If extractionDir matches file name and isn't a directory, we should be extracting a specific file.
		if file.Name == extractionDir && file.Typeflag != tar.TypeDir {
			file.Name = filepath.Base(file.Name)
		} else {
			// Transform the filename to skip the extractionDir
			file.Name = strings.TrimPrefix(file.Name, extractionDir)
		}

		// If file.Name is now empty this is the root directory we want to extract, and need not do anything.
		if file.Name == "" && file.Typeflag == tar.TypeDir {
			continue
		}

		fullPath := filepath.Join(dest, file.Name)

		// At this point only directories and block-files are handled. Symlinks and the like are ignored.
		switch file.Typeflag {
		case tar.TypeDir:
			// For a directory, if it doesn't exist, we create it.
			// nolint: vetshadow
			finfo, err := os.Stat(fullPath)
			if err == nil && finfo.IsDir() {
				continue
			}

			err = os.MkdirAll(fullPath, 0755)
			if err != nil {
				return err
			}

		case tar.TypeReg:
			fallthrough
		case tar.TypeRegA:
			// Always ensure the directory is created before trying to move the file.
			fullPathDir := filepath.Dir(fullPath)
			err = os.MkdirAll(fullPathDir, 0755)
			if err != nil {
				return fmt.Errorf("Failed to create the directory %s, err: %v", fullPathDir, err)
			}

			// For a regular file, create and copy the file.
			exFile, err := os.Create(fullPath)
			if err != nil {
				return fmt.Errorf("Failed to create file %v, err: %v", fullPath, err)
			}
			_, err = io.Copy(exFile, tf)
			_ = exFile.Close()
			if err != nil {
				return fmt.Errorf("Failed to copy to file %v, err: %v", fullPath, err)
			}
		}
	}

	return nil
}

// Unzip accepts a zip file and extracts the contents to the provided destination path.
// extractionDir is the path at which extraction should szipt; nothing will be extracted except the contents of
// extractionDir
func Unzip(source string, dest string, extractionDir string) error {
	zf, err := zip.OpenReader(source)
	if err != nil {
		return fmt.Errorf("Failed to open zipfile %s, err:%v", source, err)
	}
	defer util.CheckClose(zf)

	for _, file := range zf.File {
		// If we have an extractionDir and this doesn't match, skip it.
		if !strings.HasPrefix(file.Name, extractionDir) {
			continue
		}

		// If extractionDir matches file name and isn't a directory, we should be extracting a specific file.
		fileInfo := file.FileInfo()
		if file.Name == extractionDir && !fileInfo.IsDir() {
			file.Name = filepath.Base(file.Name)
		} else {
			// Transform the filename to skip the extractionDir
			file.Name = strings.TrimPrefix(file.Name, extractionDir)
		}

		fullPath := filepath.Join(dest, file.Name)

		if strings.HasSuffix(file.Name, "/") {
			err = os.MkdirAll(fullPath, 0777)
			if err != nil {
				return fmt.Errorf("Failed to mkdir %s, err:%v", fullPath, err)
			}
			continue
		}

		// If file.Name is now empty this is the root directory we want to extract, and need not do anything.
		if file.Name == "" {
			continue
		}

		rc, err := file.Open()
		if err != nil {
			return err
		}

		// create and copy the file.
		exFile, err := os.Create(fullPath)
		if err != nil {
			return fmt.Errorf("Failed to create file %v, err: %v", fullPath, err)
		}
		_, err = io.Copy(exFile, rc)
		_ = exFile.Close()
		if err != nil {
			return fmt.Errorf("Failed to copy to file %v, err: %v", fullPath, err)
		}
	}

	return nil
}
