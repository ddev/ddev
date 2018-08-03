package appimport

import (
	"errors"
	"fmt"
	"os"

	"strings"

	"path/filepath"

	gohomedir "github.com/mitchellh/go-homedir"
)

// ValidateAsset determines if a given asset matches the required criteria for a given asset type.
// If the path provided is a tarball, it will extract, validate, and return the extracted asset path.
func ValidateAsset(unexpandedAssetPath string, assetType string) (string, error) {
	var invalidAssetError = "%v. Please provide a valid asset path."
	extensions := []string{"tar", "gz", "tgz", "zip"}

	// Input provided via prompt or "--flag=value" is not expanded by shell. This will help ensure ~ is expanded to the user home directory.
	assetPath, err := gohomedir.Expand(unexpandedAssetPath)
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
