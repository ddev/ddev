package archive

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ddev/ddev/pkg/fileutil"
	"github.com/ddev/ddev/pkg/nodeps"
	"github.com/ddev/ddev/pkg/util"
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

// isArchiveAbsolutePath reports whether a tar entry's Linkname is an absolute
// path, in any spelling. Linkname comes from the archive being extracted, so
// it cannot be trusted to follow tar's usual POSIX-style ("/") convention;
// filepath.IsAbs alone is not enough to catch every case, since its notion
// of "absolute" is host-OS-dependent: on Windows it returns false for a
// leading "/" or "\" with no drive letter, even though the resulting symlink
// still resolves rooted (to the current drive) rather than confined under
// dest. Checking all three forms explicitly catches an absolute target
// regardless of which spelling the archive uses or which OS is running.
func isArchiveAbsolutePath(p string) bool {
	return strings.HasPrefix(p, "/") || strings.HasPrefix(p, `\`) || filepath.IsAbs(p)
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

		// Prevent path traversal (ZipSlip): ensure fullPath stays within dest
		if !strings.HasPrefix(filepath.Clean(fullPath)+string(os.PathSeparator), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry %q escapes destination directory", file.Name)
		}

		// Handle directories, regular files, and symlinks
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

			err = util.Chmod(fullPath, fs.FileMode(file.Mode))
			if err != nil {
				return fmt.Errorf("failed to chmod %v dir %v, err: %v", fs.FileMode(file.Mode), fullPath, err)
			}

		case tar.TypeReg:
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
			err = util.Chmod(fullPath, fs.FileMode(file.Mode))
			if err != nil {
				return fmt.Errorf("failed to chmod %v file %v, err: %v", fs.FileMode(file.Mode), fullPath, err)
			}

		case tar.TypeSymlink:
			// Ensure the parent directory exists before creating the symlink.
			fullPathDir := filepath.Dir(fullPath)
			err = os.MkdirAll(fullPathDir, 0755)
			if err != nil {
				return fmt.Errorf("failed to create the directory %s, err: %v", fullPathDir, err)
			}

			// A dev-environment archive has no legitimate need to plant an
			// absolute symlink target, and there's no safe general way to
			// let one through: rebasing it under dest is what caused this
			// function's history of escape bugs (see GHSA-9hq4-hm3j-jmph).
			// Reject absolute targets outright; only relative targets,
			// resolved against and confined to the symlink's parent
			// directory, are allowed.
			if isArchiveAbsolutePath(file.Linkname) {
				return fmt.Errorf("symlink target %q in archive entry %q is absolute, which is not allowed", file.Linkname, file.Name)
			}
			resolvedTarget := filepath.Join(fullPathDir, file.Linkname)
			if !strings.HasPrefix(filepath.Clean(resolvedTarget)+string(os.PathSeparator), filepath.Clean(dest)+string(os.PathSeparator)) {
				return fmt.Errorf("symlink target %q in archive entry %q escapes destination directory", file.Linkname, file.Name)
			}

			// Remove any existing file/symlink at this path
			_ = os.Remove(fullPath)

			err = os.Symlink(file.Linkname, fullPath)
			if err != nil {
				return fmt.Errorf("failed to create symlink %v -> %v, err: %v", fullPath, file.Linkname, err)
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
		return fmt.Errorf("failed to open zipfile %s, err:%v", source, err)
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

		// Prevent path traversal (ZipSlip): ensure fullPath stays within dest
		if !strings.HasPrefix(filepath.Clean(fullPath)+string(os.PathSeparator), filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry %q escapes destination directory", file.Name)
		}

		if strings.HasSuffix(file.Name, "/") {
			err = os.MkdirAll(fullPath, 0777)
			if err != nil {
				return fmt.Errorf("failed to mkdir %s, err:%v", fullPath, err)
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
			return fmt.Errorf("failed to create file %v, err: %v", fullPath, err)
		}
		_, err = io.Copy(exFile, rc)
		_ = exFile.Close()
		if err != nil {
			return fmt.Errorf("failed to copy to file %v, err: %v", fullPath, err)
		}
	}

	// If no files matched the extraction path, return an error.
	if !foundPathMatch {
		return fmt.Errorf("failed to find files in extraction path: %s", extractionDir)
	}

	return nil
}

// SQLDumpReader returns a reader over the SQL content of source, which may
// be a plain .sql/.mysql file or one compressed with gzip, bzip2, or xz.
// The returned reader must be closed when done.
func SQLDumpReader(source string) (io.ReadCloser, error) {
	f, err := os.Open(source)
	if err != nil {
		return nil, err
	}
	switch {
	case strings.HasSuffix(source, "sql.gz") || strings.HasSuffix(source, "mysql.gz"):
		gf, err := gzip.NewReader(f)
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		return &readCloser{r: gf, closeFn: func() error { return errors.Join(gf.Close(), f.Close()) }}, nil
	case strings.HasSuffix(source, "sql.bz2") || strings.HasSuffix(source, "mysql.bz2"):
		return &readCloser{r: bzip2.NewReader(bufio.NewReader(f)), closeFn: f.Close}, nil
	case strings.HasSuffix(source, "sql.xz") || strings.HasSuffix(source, "mysql.xz"):
		xr, err := xz.NewReader(bufio.NewReader(f))
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		return &readCloser{r: xr, closeFn: f.Close}, nil
	default:
		return f, nil
	}
}

// readCloser adapts an io.Reader to an io.ReadCloser with a custom close function.
type readCloser struct {
	r       io.Reader
	closeFn func() error
}

func (rc *readCloser) Read(p []byte) (int, error) { return rc.r.Read(p) }
func (rc *readCloser) Close() error               { return rc.closeFn() }

// sqlMemberName reports whether an archive member would be imported into a
// database: a .sql or .mysql file at the top level of the archive, or of
// extractionDir when given. It mirrors the extraction logic of Untar/Unzip
// followed by ImportDB's top-level glob: extractionDir names a prefix within
// the archive (a directory, or a single file when a member matches it
// exactly), and the rest of the member's name is considered relative to the
// top level.
func sqlMemberName(name, extractionDir string, isDir bool) bool {
	if !strings.HasPrefix(name, extractionDir) {
		return false
	}
	if name == extractionDir && !isDir {
		name = filepath.Base(name)
	} else {
		// TrimPrefix can leave a leading "/" when extractionDir has none;
		// filepath.Join in Untar/Unzip cleans that away.
		name = strings.TrimLeft(strings.TrimPrefix(name, extractionDir), "/")
	}
	if name == "" || isDir || strings.Contains(name, "/") {
		return false
	}
	return strings.HasSuffix(name, ".sql") || strings.HasSuffix(name, ".mysql")
}

// SQLMembersInTar returns the names of the members of the (optionally
// gzip/bzip2/xz-compressed) tar archive source that a database import would
// use: .sql and .mysql files at the top level of the archive, or of
// extractionDir when given. Names are returned sorted. An error is returned
// when no member matches extractionDir.
func SQLMembersInTar(source, extractionDir string) ([]string, error) {
	tf, closeFn, err := newTarReader(source)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	var names []string
	foundExtractionPath := false
	for {
		hdr, err := tf.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error during read of tar archive %v, err: %v", source, err)
		}
		if !strings.HasPrefix(hdr.Name, extractionDir) {
			continue
		}
		foundExtractionPath = true
		if sqlMemberName(hdr.Name, extractionDir, hdr.Typeflag != tar.TypeReg) {
			names = append(names, hdr.Name)
		}
	}
	if !foundExtractionPath {
		return nil, fmt.Errorf("failed to find files in extraction path: %s", extractionDir)
	}
	slices.Sort(names)
	return names, nil
}

// SQLMembersInZip is the zip analog of SQLMembersInTar.
func SQLMembersInZip(source, extractionDir string) ([]string, error) {
	zf, err := zip.OpenReader(source)
	if err != nil {
		return nil, fmt.Errorf("failed to open zipfile %s, err:%v", source, err)
	}
	defer util.CheckClose(zf)

	var names []string
	foundExtractionPath := false
	for _, file := range zf.File {
		if !strings.HasPrefix(file.Name, extractionDir) {
			continue
		}
		foundExtractionPath = true
		if sqlMemberName(file.Name, extractionDir, file.FileInfo().IsDir()) {
			names = append(names, file.Name)
		}
	}
	if !foundExtractionPath {
		return nil, fmt.Errorf("failed to find files in extraction path: %s", extractionDir)
	}
	slices.Sort(names)
	return names, nil
}

// StreamTarMembers copies the contents of the named members of the
// (optionally gzip/bzip2/xz-compressed) tar archive source to w, in the
// order given. names should come from SQLMembersInTar.
func StreamTarMembers(source string, names []string, w io.Writer) error {
	for _, name := range names {
		tf, closeFn, err := newTarReader(source)
		if err != nil {
			return err
		}
		found := false
		for {
			hdr, err := tf.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				closeFn()
				return err
			}
			if hdr.Name == name {
				if _, err := io.Copy(w, tf); err != nil {
					closeFn()
					return err
				}
				found = true
				break
			}
		}
		closeFn()
		if !found {
			return fmt.Errorf("archive member %q not found in %s", name, source)
		}
	}
	return nil
}

// StreamZipMembers copies the contents of the named members of the zip
// archive source to w, in the order given. names should come from
// SQLMembersInZip.
func StreamZipMembers(source string, names []string, w io.Writer) error {
	zf, err := zip.OpenReader(source)
	if err != nil {
		return fmt.Errorf("failed to open zipfile %s, err:%v", source, err)
	}
	defer util.CheckClose(zf)

	byName := make(map[string]*zip.File, len(zf.File))
	for _, file := range zf.File {
		byName[file.Name] = file
	}
	for _, name := range names {
		file, ok := byName[name]
		if !ok {
			return fmt.Errorf("archive member %q not found in %s", name, source)
		}
		rc, err := file.Open()
		if err != nil {
			return err
		}
		_, err = io.Copy(w, rc)
		_ = rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// newTarReader opens source and returns a tar.Reader over its contents along
// with a function that closes the file and any decompressor.
func newTarReader(source string) (*tar.Reader, func(), error) {
	f, err := os.Open(source)
	if err != nil {
		return nil, nil, err
	}
	closeFn := func() { _ = f.Close() }
	switch {
	case strings.HasSuffix(source, "gz"):
		gf, err := gzip.NewReader(f)
		if err != nil {
			_ = f.Close()
			return nil, nil, err
		}
		return tar.NewReader(gf), func() {
			_ = gf.Close()
			_ = f.Close()
		}, nil
	case strings.HasSuffix(source, "xz"):
		xr, err := xz.NewReader(bufio.NewReader(f))
		if err != nil {
			_ = f.Close()
			return nil, nil, err
		}
		return tar.NewReader(xr), closeFn, nil
	case strings.HasSuffix(source, "bz2"):
		return tar.NewReader(bzip2.NewReader(bufio.NewReader(f))), closeFn, nil
	default:
		return tar.NewReader(f), closeFn, nil
	}
}

// Tar takes a source dir and tarballFilePath and a single exclusion path
// It creates a gzipped tarball.
// So sorry that exclusion is a single relative path. It should be a set of patterns, rfay 2021-12-15
func Tar(src string, tarballFilePath string, exclusion string) error {
	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("unable to tar files - %v", err.Error())
	}
	separator := string(rune(filepath.Separator))

	tarball, err := os.Create(tarballFilePath)
	if err != nil {
		return fmt.Errorf("could not create tarball file '%s', got error '%s'", tarballFilePath, err.Error())
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

		fi, err := info.Info()
		if err != nil {
			return nil
		}

		// Skip directories - WalkDir handles traversal
		if fi.IsDir() {
			return nil
		}

		// For symlinks, we need to read the link target
		var linkTarget string
		if fi.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(file)
			if err != nil {
				return err
			}
			// Normalize to forward slashes for tar format
			linkTarget = filepath.ToSlash(linkTarget)
		}

		// Create header - for symlinks, second arg is the link target
		header, err := tar.FileInfoHeader(fi, linkTarget)
		if err != nil {
			return err
		}

		// update the name to correctly reflect the desired destination when untarring
		header.Name = strings.TrimPrefix(strings.ReplaceAll(file, src, ""), string(filepath.Separator))
		header.Name = filepath.ToSlash(header.Name)

		// For regular files, handle file content
		if fi.Mode().IsRegular() {
			// Windows may not get zero size of file, https://github.com/golang/go/issues/23493
			// No idea why fi.Size() comes through as zero for a few files
			stat, err := os.Stat(file)
			if err != nil {
				return err
			}
			header.Size = stat.Size()

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
			if nodeps.IsWindows() {
				buffer := make([]byte, 16)
				_, _ = f.Read(buffer)
				_, _ = f.Seek(0, 0)
				if strings.HasPrefix(string(buffer), "#!") {
					header.Mode = 0755
				}
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
		} else {
			// For symlinks and other special files, just write the header
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
		}

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
		return "", nil, fmt.Errorf("unable to create temp file: %v", err)
	}
	defer func() {
		_ = f.Close()
	}()

	util.Debug("Downloading %s to %s", url, f.Name())
	tarball := f.Name()
	defer func() {
		_ = os.Remove(tarball)
	}()

	err = util.DownloadFile(tarball, url, true, "")
	if err != nil {
		return "", nil, err
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
		return "", func() {}, fmt.Errorf("unable to create temp dir: %v", err)
	}
	cleanupFunc := func() { _ = os.RemoveAll(tmpDir) }

	err = Untar(tarball, tmpDir, "")
	if err != nil {
		return "", cleanupFunc, fmt.Errorf("unable to untar %v: %v", tmpDir, err)
	}

	// If removeTopLevel then the guts of the tarball are the first level directory
	// Really the UnTar() function should take strip-components as an argument
	// but not going to do that right now.
	extractedDir := tmpDir
	if removeTopLevel {
		list, err := fileutil.ListFilesInDir(tmpDir)
		if err != nil {
			return "", cleanupFunc, fmt.Errorf("unable to list files in %v: %v", tmpDir, err)
		}
		if len(list) == 0 {
			return "", cleanupFunc, fmt.Errorf("no files found in %v", tmpDir)
		}
		extractedDir = path.Join(tmpDir, list[0])
	}
	return extractedDir, cleanupFunc, nil
}
