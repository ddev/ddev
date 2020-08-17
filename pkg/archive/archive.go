package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/drud/ddev/pkg/util"
)

// Ungzip accepts a gzipped file and uncompresses it to the provided destination directory.
func Ungzip(source string, destDirectory string) error {
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
	exFile, err := os.Create(filepath.Join(destDirectory, fname))
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

	if err = os.MkdirAll(dest, 0755); err != nil {
		return err
	}

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

	// Define a boolean that indicates whether or not at least one
	// file matches the extraction directory.
	foundPathMatch := false
	for {
		file, err := tf.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error during read of tar archive %v, err: %v", source, err)
		}

		// If we have an extractionDir and this doesn't match, skip it.
		if !strings.HasPrefix(file.Name, extractionDir) {
			continue
		}

		// If we haven't continue-ed above, the file matches the extraction dir and this flag
		// should be ensured to be true.
		foundPathMatch = true

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
				return fmt.Errorf("failed to create the directory %s, err: %v", fullPathDir, err)
			}

			// For a regular file, create and copy the file.
			exFile, err := os.Create(fullPath)
			if err != nil {
				return fmt.Errorf("failed to create file %v, err: %v", fullPath, err)
			}
			_, err = io.Copy(exFile, tf)
			_ = exFile.Close()
			if err != nil {
				return fmt.Errorf("failed to copy to file %v, err: %v", fullPath, err)
			}
		}
	}

	// If no files matched the extraction path, return an error.
	if !foundPathMatch {
		return fmt.Errorf("failed to find files in extraction path: %s", extractionDir)
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

	if err = os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	// Define a boolean that indicates whether or not at least one
	// file matches the extraction directory.
	foundPathMatch := false
	for _, file := range zf.File {
		// If we have an extractionDir and this doesn't match, skip it.
		if !strings.HasPrefix(file.Name, extractionDir) {
			continue
		}

		// If we haven't continue-ed above, the file matches the extraction dir and this flag
		// should be ensured to be true.
		foundPathMatch = true

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

	// If no files matched the extraction path, return an error.
	if !foundPathMatch {
		return fmt.Errorf("failed to find files in extraction path: %s", extractionDir)
	}

	return nil
}

// Tar takes a source and variable writers and walks 'source' writing each file
// found to the tar writer; the purpose for accepting multiple writers is to allow
// for multiple outputs (for example a file, or md5 hash)
// From https://gist.github.com/sdomino/635a5ed4f32c93aad131#file-untargz-go
func Tar(src string, tarballFilePath string) error {

	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("Unable to tar files - %v", err.Error())
	}

	file, err := os.Create(tarballFilePath)
	if err != nil {
		return fmt.Errorf("Could not create tarball file '%s', got error '%s'", tarballFilePath, err.Error())
	}
	// nolint: errcheck
	defer file.Close()

	mw := io.MultiWriter(file)

	//gzw := gzip.NewWriter(mw)
	//defer gzw.Close()

	tw := tar.NewWriter(mw)
	defer tw.Close()

	// walk path
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		// return on any error
		if err != nil {
			return err
		}

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		if !fi.Mode().IsRegular() {
			return nil
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))
		if runtime.GOOS == "windows" {
			header.Name = strings.Replace(header.Name, `\`, `/`, -1)
		}

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// open files for taring
		f, err := os.Open(file)
		if err != nil {
			return err
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		// manually close here after each file operation; defering would cause each file close
		// to wait until all operations have completed.
		f.Close()

		return nil
	})
}
