package appimport

import (
	"errors"
	"fmt"
	"os"
	"path"

	"log"

	"strings"

	"io"

	"path/filepath"

	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/system"
	homedir "github.com/mitchellh/go-homedir"
)

// ValidateAsset determines if a given asset matches the required criteria for a given asset type. If the path provided is a tarball, it will extract, validate, and return the extracted asset path.
func ValidateAsset(assetPath string, assetType string) (string, error) {
	var invalidAssetError = "%s. Please provide a valid asset path."

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
		assetPath = extractArchive(assetPath)
	}

	info, err := os.Stat(assetPath)
	if err != nil {
		return "", fmt.Errorf(invalidAssetError, err)
	}

	// see if we can find a .sql in the directory
	if assetType == "db" && info.IsDir() {
		files, err := findFileExt(assetPath, ".sql")
		if err != nil {
			return "", err
		}

		if len(files) > 1 {
			fmt.Println("WARNING: Multiple .sql files found. Only the first .sql file will be used.")
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

	err := copyFile(source, destination)
	if err != nil {
		return fmt.Errorf("failed to copy provided database dump to container mount: %s", err)
	}

	if strings.Contains(source, os.TempDir()) {
		os.RemoveAll(path.Dir(source))
	}

	cmdArgs := []string{
		"-f", path.Join(sitepath, ".ddev", "docker-compose.yaml"),
		"exec",
		"-T", container,
		"./import.sh",
	}

	err = dockerutil.DockerCompose(cmdArgs...)
	if err != nil {
		return fmt.Errorf("failed to execute import: %s", err)
	}
	return nil
}

func copyFile(src string, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the bytes to destination from source
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	// Commit the file contents
	err = destFile.Sync()
	if err != nil {
		return err
	}
	return nil
}

// extractArchive uses tar to extract a provided archive and returns the path of the extracted archive contents
func extractArchive(extPath string) string {
	extractDir := path.Join(os.TempDir(), "extract")
	err := os.Mkdir(extractDir, 0755)
	if err != nil {
		log.Fatal(err)
	}
	out, err := system.RunCommand(
		"tar",
		[]string{"-xzf", extPath, "-C", extractDir},
	)
	if err != nil {
		log.Fatalf("Unable to extract archive: %s. command output: %s", err, out)
	}
	return extractDir
}

// findFileExt walks a given directory searching for a given extension and returns a list of matching results.
func findFileExt(dirpath string, ext string) ([]string, error) {
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
