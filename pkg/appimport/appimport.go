package appimport

import (
	"errors"
	"fmt"
	"os"
	"path"

	"strings"

	"path/filepath"

	"github.com/drud/drud-go/utils/dockerutil"
	"github.com/drud/drud-go/utils/system"
	homedir "github.com/mitchellh/go-homedir"
)

// ValidateAsset determines if a given asset matches the required criteria for a given asset type.
// If the path provided is a tarball, it will extract, validate, and return the extracted asset path.
func ValidateAsset(assetPath string, assetType string) (string, error) {
	var invalidAssetError = "%v. Please provide a valid asset path."
	extensions := []string{"tar.gz", "tgz", "sql.gz"}

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

	info, err := os.Stat(assetPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf(invalidAssetError, errors.New("file not found"))
	}
	if err != nil {
		return "", fmt.Errorf(invalidAssetError, err)
	}

	// this error should not be output to user. its intent is to be evaluated by code implementing
	// this function to handle an archive as needed.
	for _, ext := range extensions {
		if strings.HasSuffix(assetPath, ext) {
			return assetPath, errors.New("is archive")
		}
	}

	if assetType == "files" && !info.IsDir() {
		return "", fmt.Errorf(invalidAssetError, errors.New("provided path is not a directory or archive"))
	}

	if assetType == "db" && !strings.HasSuffix(assetPath, "sql") {
		return "", fmt.Errorf(invalidAssetError, errors.New("provided path is not a .sql file or archive"))
	}

	return assetPath, nil
}

// ImportSQLDump places a provided sql dump into the app data mount, and executes mysql import to the container.
func ImportSQLDump(composePath string, container string) error {
	if !dockerutil.IsRunning(container) {
		return fmt.Errorf("the %s container is not currently running", container)
	}

	cmdArgs := []string{
		"-f", composePath,
		"exec",
		"-T", container,
		"./import.sh",
	}

	err := dockerutil.DockerCompose(cmdArgs...)
	if err != nil {
		return fmt.Errorf("failed to execute import: %v", err)
	}

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
