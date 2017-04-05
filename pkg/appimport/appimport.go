package appimport

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"strings"

	"io"

	"path/filepath"

	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/system"
	homedir "github.com/mitchellh/go-homedir"
)

// ValidateAsset determines if a given asset matches the required criteria for a given asset type.
// If the path provided is a tarball, it will extract, validate, and return the extracted asset path.
func ValidateAsset(assetPath string, assetType string) (string, error) {
	var invalidAssetError = "%v. Please provide a valid asset path."

	// Input provided via prompt or "--flag=value" is not expanded by shell. This will help ensure ~ is expanded to the user home directory.
	assetPath, err := homedir.Expand(assetPath)
	if err != nil {
		return "", fmt.Errorf(invalidAssetError, err)
	}

	// ensure we are working w/ an absolute path
	assetPath, err = filepath.Abs(assetPath)
	if err != nil {
		return "", fmt.Errorf(invalidAssetError, err)
	}

	// make sure the path exists
	if _, err = os.Stat(assetPath); os.IsNotExist(err) {
		return "", fmt.Errorf(invalidAssetError, err)
	}

	// if we have a tarball, extract and set path to the extraction point
	if strings.HasSuffix(assetPath, ".tar.gz") {
		assetPath, err = extractArchive(assetPath)
		if err != nil {
			return "", fmt.Errorf(invalidAssetError, err)
		}
	}

	info, err := os.Stat(assetPath)
	if err != nil {
		return "", fmt.Errorf(invalidAssetError, err)
	}

	// see if we can find a .sql in the directory
	if assetType == "db" && info.IsDir() {
		files, err := findFileByExtension(assetPath, ".sql")
		if err != nil {
			return "", err
		}

		if len(files) > 1 {
			fmt.Printf("WARNING: Multiple .sql files found, only single file imports are supported. Importing %s. \n", files[0])
		}
		assetPath = path.Join(assetPath, files[0])
	}

	if assetType == "files" && !info.IsDir() {
		return "", fmt.Errorf(invalidAssetError, errors.New("provided path is not a directory or archive; expecting a directory path or .tar.gz file"))
	}
	return assetPath, nil
}

// ImportSQLDump places a provided sql dump into the app data mount, and executes mysql import to the container.
func ImportSQLDump(source string, sitepath string, container string) error {
	destination := path.Join(sitepath, ".ddev", "data", "data.sql")
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return err
	}
	if !strings.HasSuffix(source, ".sql") {
		return errors.New("a database dump in .sql format must be provided")
	}

	if !dockerutil.IsRunning(container) {
		return fmt.Errorf("the %s container is not currently running", container)
	}

	err := CopyFile(source, destination)
	if err != nil {
		return fmt.Errorf("failed to copy provided database dump to container mount: %v", err)
	}

	// if we extracted an archive, clean up the extraction point
	if strings.Contains(source, os.TempDir()) {
		defer os.RemoveAll(path.Dir(source))
	}

	cmdArgs := []string{
		"-f", path.Join(sitepath, ".ddev", "docker-compose.yaml"),
		"exec",
		"-T", container,
		"./import.sh",
	}

	err = dockerutil.DockerCompose(cmdArgs...)
	if err != nil {
		return fmt.Errorf("failed to execute import: %v", err)
	}

	// remove the copied dump from container mount point
	os.Remove(path.Join(sitepath, ".ddev", "data", "data.sql"))

	return nil
}

// extractArchive uses tar to extract a provided archive and returns the path of the extracted archive contents
func extractArchive(extPath string) (string, error) {
	extractDir := path.Join(os.TempDir(), "extract")
	err := os.Mkdir(extractDir, 0755)
	if err != nil {
		return "", err
	}
	out, err := system.RunCommand(
		"tar",
		[]string{"-xzf", extPath, "-C", extractDir},
	)
	if err != nil {
		return "", fmt.Errorf("Unable to extract archive: %v. command output: %s", err, out)
	}
	return extractDir, nil
}

// findFileByExtension walks a given directory searching for a given extension and returns a list of matching results.
func findFileByExtension(dirpath string, ext string) ([]string, error) {
	var match []string
	err := filepath.Walk(dirpath, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ext) {
			match = append(match, f.Name())
		}
		return nil
	})
	if err != nil {
		return []string{}, err
	}

	if len(match) < 1 {
		return match, fmt.Errorf("no %s files found in %s", ext, dirpath)
	}

	return match, nil
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
			// Skip symlinks.
			if entry.Mode()&os.ModeSymlink != 0 {
				continue
			}

			err = CopyFile(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
