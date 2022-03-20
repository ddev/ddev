package archive

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"github.com/drud/ddev/pkg/fileutil"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/drud/ddev/pkg/util"
	"github.com/ulikunitz/xz"
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

// UnBzip2 accepts a bzip2-compressed file and uncompresses it to the provided destination directory.
func UnBzip2(source string, destDirectory string) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}

	defer func() {
		if e := f.Close(); e != nil {
			err = e
		}
	}()
	br := bufio.NewReader(f)

	gf := bzip2.NewReader(br)

	fname := strings.TrimSuffix(filepath.Base(f.Name()), ".bz2")
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

// UnXz accepts an xz-compressed file and uncompresses it to the provided destination directory.
func UnXz(source string, destDirectory string) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}

	defer func() {
		if e := f.Close(); e != nil {
			err = e
		}
	}()
	br := bufio.NewReader(f)

	gf, err := xz.NewReader(br)
	if err != nil {
		return err
	}

	fname := strings.TrimSuffix(filepath.Base(f.Name()), ".xz")
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

// Untar accepts a tar, tar.gz, tar.bz2, tar.xz file and extracts the contents to the provided destination path.
// extractionDir is the path at which extraction should start; nothing will be extracted except the contents of
// extractionDir. If extranctionDir is empty, the entire tarball is extracted.
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

	switch {
	case strings.HasSuffix(source, "gz"):
		gf, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer util.CheckClose(gf)
		tf = tar.NewReader(gf)

	case strings.HasSuffix(source, "xz"):
		gf, err := xz.NewReader(f)
		if err != nil {
			return err
		}
		tf = tar.NewReader(gf)

	case strings.HasSuffix(source, "bz2"):
		br := bufio.NewReader(f)
		gf := bzip2.NewReader(br)
		if err != nil {
			return err
		}
		tf = tar.NewReader(gf)

	default:
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

			err = os.Chmod(fullPath, fs.FileMode(file.Mode))
			if err != nil {
				return fmt.Errorf("failed to chmod %v dir %v, err: %v", fs.FileMode(file.Mode), fullPath, err)
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
			err = os.Chmod(fullPath, fs.FileMode(file.Mode))
			if err != nil {
				return fmt.Errorf("failed to chmod %v file %v, err: %v", fs.FileMode(file.Mode), fullPath, err)
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

// Tar takes a source dir and tarballFilePath and a single exclusion path
// It creates a gzipped tarball.
// So sorry that exclusion is a single relative path. It should be a set of patterns, rfay 2021-12-15
func Tar(src string, tarballFilePath string, exclusion string) error {
	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("Unable to tar files - %v", err.Error())
	}
	separator := string(rune(filepath.Separator))

	tarball, err := os.Create(tarballFilePath)
	if err != nil {
		return fmt.Errorf("Could not create tarball file '%s', got error '%s'", tarballFilePath, err.Error())
	}
	// nolint: errcheck
	defer tarball.Close()

	mw := io.MultiWriter(tarball)

	gzw := gzip.NewWriter(mw)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	// walk path
	return filepath.WalkDir(src, func(file string, info fs.DirEntry, errArg error) error {
		// return on any error
		if errArg != nil {
			return errArg
		}

		relativePath := strings.TrimPrefix(file, src+separator)

		if exclusion != "" && strings.HasPrefix(relativePath, exclusion) {
			return nil
		}

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		fi, err := info.Info()
		if err != nil {
			return nil
		}
		if !fi.Mode().IsRegular() {
			return nil
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// open files for tarring
		f, err := os.Open(file)
		if err != nil {
			return err
		}

		// Windows filesystem has no concept of executable bit, but we're copying shell scripts
		// and they need to be executable. So if we detect a shell script
		// set its mode to executable. It seems this is what utilities like git-bash
		// and cygwin, etc. have done for years to work around the lack of mode bits on NTFS,
		// for example, see https://stackoverflow.com/a/25730108/215713
		if runtime.GOOS == "windows" {
			buffer := make([]byte, 16)
			_, _ = f.Read(buffer)
			_, _ = f.Seek(0, 0)
			if strings.HasPrefix(string(buffer), "#!") {
				header.Mode = 0755
			}
		}

		// update the name to correctly reflect the desired destination when untarring
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))
		if runtime.GOOS == "windows" {
			header.Name = strings.Replace(header.Name, `\`, `/`, -1)
		}

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		// manually close here after each file operation; deferring would cause each file close
		// to wait until all operations have completed.
		f.Close()

		return nil
	})
}

// DownloadAndExtractTarball takes an url to a tar.gz file and
// extracts into a new a temp directory and the directory
// and a cleanup function.
// It's the caller's responsibility to call the cleanup function.
func DownloadAndExtractTarball(url string, removeTopLevel bool) (string, func(), error) {
	base := filepath.Base(url)
	f, err := os.CreateTemp("", fmt.Sprintf("%s_*.tar.gz", base))
	if err != nil {
		return "", nil, fmt.Errorf("Unable to create temp file: %v", err)
	}
	defer func() {
		_ = f.Close()
	}()

	util.Success("Downloading %s", url)
	tarball := f.Name()
	defer func() {
		_ = os.Remove(tarball)
	}()

	err = util.DownloadFile(tarball, url, true)
	if err != nil {
		return "", nil, fmt.Errorf("Unable to download %v: %v", url, err)
	}
	extractedDir, cleanup, err := ExtractTarballWithCleanup(tarball, removeTopLevel)
	return extractedDir, cleanup, err
}

// ExtractTarballWithCleanup takes a tarball file and extracts it into a temp directory
// Caller is responsible for cleanup of the temp directory using the returned
// cleanup function.
// If removeTopLevel is true, the top level directory will be removed.
func ExtractTarballWithCleanup(tarball string, removeTopLevel bool) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("ddev_%s_*", filepath.Base(tarball)))
	if err != nil {
		return "", nil, fmt.Errorf("Unable to create temp dir: %v", err)
	}

	err = Untar(tarball, tmpDir, "")
	if err != nil {
		return "", nil, fmt.Errorf("Unable to untar %v: %v", tmpDir, err)
	}

	// If removeTopLevel then the guts of the tarball are the first level directory
	// Really the UnTar() function should take strip-components as an argument
	// but not going to do that right now.
	extractedDir := tmpDir
	if removeTopLevel {
		list, err := fileutil.ListFilesInDir(tmpDir)
		if err != nil {
			return "", nil, fmt.Errorf("Unable to list files in %v: %v", tmpDir, err)
		}
		if len(list) == 0 {
			return "", nil, fmt.Errorf("No files found in %v", tmpDir)
		}
		extractedDir = path.Join(tmpDir, list[0])
	}
	cleanupFunc := func() { _ = os.RemoveAll(tmpDir) }
	return extractedDir, cleanupFunc, nil
}
